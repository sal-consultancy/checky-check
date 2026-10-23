package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"
)

type actionJob struct {
	ID     string `json:"id"`
	Host   string `json:"host"`
	Status string `json:"status"`
	Output string `json:"output"`
	User   string `json:"user"`
}

func registerActions(path string, auth authContext) {
	var mu sync.Mutex
	jobs := map[string]actionJob{}
	active := map[string]bool{}
	http.HandleFunc("/api/action-definitions", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", 405)
			return
		}
		cfg, e := loadConfig(path)
		if e != nil {
			http.Error(w, "Configuration unavailable", 500)
			return
		}
		definitions := map[string]CorrectiveAction{}
		for id, a := range cfg.Actions {
			a.Command = ""
			definitions[id] = a
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(definitions)
	}))
	http.HandleFunc("/api/actions", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		session := auth.sessionFromRequest(r)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			mu.Lock()
			defer mu.Unlock()
			job, ok := jobs[r.URL.Query().Get("id")]
			if ok && job.User != session.Username && !session.Permissions.Admin {
				http.Error(w, "Forbidden", 403)
				return
			}
			if !ok {
				http.Error(w, "Unknown job", 404)
				return
			}
			json.NewEncoder(w).Encode(job)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", 405)
			return
		}
		var req struct {
			Host   string `json:"host"`
			Check  string `json:"check"`
			Action string `json:"action"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		cfg, err := loadConfig(path)
		if err != nil {
			http.Error(w, "Configuration unavailable", 500)
			return
		}
		action, ok := cfg.Actions[req.Action]
		linked := false
		always := false
		for _, ref := range cfg.Checks[req.Check].Actions {
			if ref.Action == req.Action && (ref.When == "failed" || ref.When == "always") {
				linked = true
				always = always || ref.When == "always"
			}
		}
		if !slices.Contains(action.AllowedRoles, session.Role) {
			http.Error(w, "Action role required", 403)
			return
		}
		targets := findHostTargetsForCheck(cfg, req.Check, req.Host)
		if !ok || !linked || action.Type != "command" || action.Command == "" || req.Host == "" || len(targets) != 1 {
			http.Error(w, "Action unavailable", 400)
			return
		}
		target := targets[0]
		mu.Lock()
		if active[req.Host] {
			mu.Unlock()
			http.Error(w, "Host already has an active action", 409)
			return
		}
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		job := actionJob{ID: id, Host: req.Host, Status: "running", User: session.Username}
		jobs[id] = job
		active[req.Host] = true
		mu.Unlock()
		go func() {
			output := ""
			status := "failed"
			defer func() {
				mu.Lock()
				job.Status = status
				job.Output = output
				jobs[id] = job
				delete(active, req.Host)
				mu.Unlock()
				log.Printf("Corrective action user=%q host=%q check=%q action=%q status=%s", job.User, req.Host, req.Check, req.Action, status)
			}()
			result, e := executeHostCheck(cfg, req.Host, target.HostConfig, target.HostGroup, req.Check, log.Default())
			if e != nil || (!always && result.Status != "failed") || result.ErrorType != "" {
				output = "Check no longer requires correction or could not be evaluated"
				return
			}
			identity := cfg.Identities[resolveIdentityName(cfg, target.HostConfig, target.HostGroup)]
			timeout := 30 * time.Minute
			if action.Timeout != "" {
				d, e := time.ParseDuration(action.Timeout)
				if e != nil || d <= 0 {
					output = "Invalid action timeout"
					return
				}
				timeout = d
			}
			if isAWSSSMIdentity(identity) {
				output, e = runSSMCommand(identity, req.Host, target.HostConfig.Target, action.Command, timeout)
			} else {
				output, e = runCommandWithTimeout(identity.User, req.Host, getSSHAuthMethod(identity), action.Command, timeout)
			}
			if e != nil {
				output += "\n" + e.Error()
				return
			}
			verify := action.Verify
			checkName := verify.Check
			if checkName == "" {
				checkName = req.Check
			}
			if len(findHostTargetsForCheck(cfg, checkName, req.Host)) != 1 {
				output += "\nVerification check not active on host"
				return
			}
			deadline := time.Now().Add(10 * time.Minute)
			if verify.Timeout != "" {
				d, err := time.ParseDuration(verify.Timeout)
				if err != nil || d <= 0 {
					output += "\nInvalid verification timeout"
					return
				}
				deadline = time.Now().Add(d)
			}
			disconnected := !verify.WaitForReconnect
			for {
				response, err := rerunCheck(path, TargetedRunRequest{Kind: "host_check", Host: req.Host, CheckName: checkName})
				if err == nil && len(response.HostResults) == 1 {
					result := response.HostResults[0]
					if result.ErrorType != "" {
						disconnected = true
					}
					if disconnected && result.Status == "passed" && result.ErrorType == "" {
						status = "completed"
						return
					}
				}
				if !verify.WaitForReconnect || time.Now().After(deadline) {
					output += "\nVerification failed or timed out"
					return
				}
				time.Sleep(5 * time.Second)
			}
		}()
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(job)
	}))
}

func validateActions(c Config) error {
	for id, a := range c.Actions {
		if a.Type != "command" || a.Command == "" || a.Label == "" || len(a.AllowedRoles) == 0 {
			return fmt.Errorf("action %q requires type command, command, label and allowed_roles", id)
		}
		for _, role := range a.AllowedRoles {
			if role != "admin" && role != "operator" && role != "viewer" {
				return fmt.Errorf("action %q has invalid role %q", id, role)
			}
		}
		for _, value := range []string{a.Timeout, a.Verify.Timeout} {
			if value != "" {
				d, e := time.ParseDuration(value)
				if e != nil || d <= 0 {
					return fmt.Errorf("action %q has invalid timeout", id)
				}
			}
		}
		if a.Verify.Check != "" {
			if _, ok := c.Checks[a.Verify.Check]; !ok {
				return fmt.Errorf("action %q references unknown verification check", id)
			}
		}
	}
	for name, check := range c.Checks {
		for _, ref := range check.Actions {
			if _, ok := c.Actions[ref.Action]; !ok {
				return fmt.Errorf("check %q references unknown action %q", name, ref.Action)
			}
			if ref.When != "failed" && ref.When != "always" {
				return fmt.Errorf("check %q action requires when: failed or always", name)
			}
		}
	}
	return nil
}
