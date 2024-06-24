package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

func getSSHAuthMethod(identity Identity) []ssh.AuthMethod {
	var authMethods []ssh.AuthMethod

	if identity.Password != "" {
		authMethods = append(authMethods, ssh.Password(identity.Password))
	}

	if identity.Key != "" {
		buffer, err := ioutil.ReadFile(filepath.Clean(identity.Key))
		if err != nil {
			log.Fatalf("unable to read private key: %v", err)
		}

		var key ssh.Signer
		if identity.Passphrase == "" {
			key, err = ssh.ParsePrivateKey(buffer)
		} else {
			key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(identity.Passphrase))
		}

		if err != nil {
			log.Fatalf("unable to parse private key: %v", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	return authMethods
}

func runCommand(user, host string, authMethods []ssh.AuthMethod, command string) (string, error) {
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), sshConfig)
	if err != nil {
		log.Printf("Failed to dial: %v", err)
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		log.Printf("Failed to run command: %v", err)
		return "", err
	}

	return string(output), nil
}

func runLocalCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run local command: %v", err)
		return "", err
	}

	return string(output), nil
}

func checkServiceStatus(user, host string, authMethods []ssh.AuthMethod, service string) (string, error) {
	command := fmt.Sprintf("systemctl is-active %s", service)
	result, err := runCommand(user, host, authMethods, command)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func runChecksOnHost(config Config, host string, hostConfig Host, groupVars map[string]string, wg *sync.WaitGroup, logger *log.Logger, results *[]CheckResult) {
	defer wg.Done()

	logger.Printf("Running checks on host: %s", host)

	var combinedVars map[string]string
	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists {
			combinedVars = mergeVars(config.HostDefaults.HostVars, template.HostVars, groupVars, hostConfig.HostVars)
		} else {
			combinedVars = mergeVars(config.HostDefaults.HostVars, groupVars, hostConfig.HostVars)
		}
	} else {
		combinedVars = mergeVars(config.HostDefaults.HostVars, groupVars, hostConfig.HostVars)
	}

	logger.Printf("Combined vars for host %s: %v", host, combinedVars)

	var combinedChecks []string
	combinedChecks = append(combinedChecks, config.HostDefaults.HostChecks...)
	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists {
			combinedChecks = append(combinedChecks, template.HostChecks...)
		}
	}
	combinedChecks = append(combinedChecks, hostConfig.HostChecks...)

	logger.Printf("Combined checks for host %s: %v", host, combinedChecks)

	identityName := config.HostDefaults.Identity
	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists && template.HostVars != nil {
			if id, exists := template.HostVars["identity"]; exists {
				identityName = id
			}
		}
	}
	if groupVars != nil {
		if id, exists := groupVars["identity"]; exists {
			identityName = id
		}
	}
	if hostConfig.Identity != "" {
		identityName = hostConfig.Identity
	}

	identity, exists := config.Identities[identityName]
	if !exists {
		logger.Fatalf("Identity %s not found for host %s", identityName, host)
		return
	}

	logger.Printf("Using identity for host %s: %v", host, identity)

	authMethods := getSSHAuthMethod(identity)

	for _, checkName := range combinedChecks {
		check, exists := config.Checks[checkName]
		if !exists {
			logger.Printf("Check %s not defined in config\n", checkName)
			continue
		}

		logger.Printf("Running check %s on host %s", checkName, host)

		var result string
		var err error
		var checkFailed bool

		timeout := parseTimeout(check.Timeout)

		if check.Local {
			command := replaceVariables(check.Command, combinedVars)
			result, err = runLocalCommand(command)
			if err != nil {
				logger.Printf("Failed to run local command %s: %v\n", command, err)
				result = "Timeout or Command Error"
				checkFailed = true
			} else {
				checkFailed = evaluateCondition(result, check.FailWhen, check.FailValue)
			}
		} else {
			if check.Command != "" {
				command := replaceVariables(check.Command, combinedVars)
				logger.Printf("Running command on host %s: %s", host, command)
				result, err = runCommandWithTimeout(identity.User, host, authMethods, command, timeout)
				if err != nil {
					logger.Printf("Failed to run command %s on host %s: %v\n", command, host, err)
					result = "Timeout or Command Error"
					checkFailed = true
				} else {
					checkFailed = evaluateCondition(result, check.FailWhen, check.FailValue)
				}
			} else if check.Service != "" {
				logger.Printf("Checking service %s on host %s", check.Service, host)
				result, err = checkServiceStatus(identity.User, host, authMethods, check.Service)
				if err != nil {
					logger.Printf("Failed to check service %s status on host %s: %v\n", check.Service, host, err)
					result = "Timeout or Command Error"
					checkFailed = true
				} else {
					checkFailed = evaluateCondition(result, check.FailWhen, check.FailValue)
				}
			}
		}

		status := "passed"
		if checkFailed {
			status = "failed"
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("%s - Host: %s - Check: %s - Status: %s - Value: %s\n", timestamp, host, checkName, status, strings.TrimSpace(result))

		*results = append(*results, CheckResult{
			Host:      host,
			Check:     checkName,
			Status:    status,
			Value:     strings.TrimSpace(result),
			Timestamp: timestamp,
			Vars:      combinedVars,
		})
	}
}

func runChecks(configPath string) {
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("unable to read config file: %v", err)
	}

	configStr, err := substituteEnvVariables(string(configFile))
	if err != nil {
		log.Fatalf("unable to substitute env variables: %v", err)
	}

	var config Config
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		log.Fatalf("unable to parse config file: %v", err)
	}

	logFile, err := os.OpenFile("remote_check.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("unable to open log file: %v", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	var wg sync.WaitGroup
	var results []CheckResult

	for _, group := range config.HostGroups {
		groupVars := group.HostVars
		for host, hostConfig := range group.Hosts {
			wg.Add(1)
			go runChecksOnHost(config, host, hostConfig, groupVars, &wg, logger, &results)
		}
	}

	wg.Wait()

	resultFile, err := os.Create("results.json")
	if err != nil {
		log.Fatalf("unable to create result file: %v", err)
	}
	defer resultFile.Close()

	resultData := ResultFile{
		Checks:  config.Checks,
		Results: mapResults(results),
		Report:  config.Report, // Voeg de report-informatie toe
	}

	if err := json.NewEncoder(resultFile).Encode(resultData); err != nil {
		log.Fatalf("unable to write results to file: %v", err)
	}
}

func mapResults(results []CheckResult) map[string]map[string]CheckResult {
	resultMap := make(map[string]map[string]CheckResult)
	for _, result := range results {
		if _, exists := resultMap[result.Host]; !exists {
			resultMap[result.Host] = make(map[string]CheckResult)
		}
		resultMap[result.Host][result.Check] = result
	}
	return resultMap
}
