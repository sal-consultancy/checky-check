package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Embed de gehele build directory
//
//go:embed frontend/build/*
var content embed.FS

var version string

var AppVersion = "development" // Standaard versie voor lokale ontwikkeling

func init() {

}

func main() {
	defaultConfigPath := os.Getenv("CHECKYCHECK_CONFIG_PATH")
	if defaultConfigPath == "" {
		defaultConfigPath = "config.yaml"
	}

	configPath := flag.String("config", defaultConfigPath, "Path to the config file or config directory")
	mode := flag.String("mode", "check", "Mode to run: check, report, or serve")
	port := flag.Int("port", 8070, "Port to run the server on")
	showVersion := flag.Bool("version", false, "Show application version") // -version flag
	flag.Parse()

	// Controleer of de -version flag is gezet
	if *showVersion {
		fmt.Printf("Application Version: %s\n", AppVersion)
		os.Exit(0) // Stop het programma na het tonen van de versie
	}

	switch *mode {
	case "check":
		if err := runChecks(*configPath); err != nil {
			log.Fatalf("%v", err)
		}

	case "serve":
		serve(*port, *configPath)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func getCommand(configPath string) *exec.Cmd {
	if executablePath, err := os.Executable(); err == nil {
		if _, statErr := os.Stat(executablePath); statErr == nil {
			return exec.Command(executablePath, "-mode=check", "-config="+configPath)
		}
	}

	binaryName := fmt.Sprintf("checkycheck-%s-%s-%s", version, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	if _, err := os.Stat(binaryName); os.IsNotExist(err) {
		// Fallback to running the Go files directly if the binary doesn't exist
		return exec.Command("go", "run", "main.go", "remote_check.go", "types.go", "helpers.go", "-mode=check", "-config="+configPath)
	}
	return exec.Command("./"+binaryName, "-mode=check", "-config="+configPath)
}

type authContext struct {
	config AuthConfig
}

func defaultAuthConfig() AuthConfig {
	return AuthConfig{
		Mode:         "none",
		UserHeader:   "X-Forwarded-User",
		EmailHeader:  "X-Forwarded-Email",
		GroupsHeader: "X-Forwarded-Groups",
		LogoutPath:   "/oauth2/sign_out",
	}
}

func normalizeAuthConfig(authConfig AuthConfig) AuthConfig {
	normalized := defaultAuthConfig()

	if mode := strings.ToLower(strings.TrimSpace(authConfig.Mode)); mode != "" {
		normalized.Mode = mode
	}
	if header := strings.TrimSpace(authConfig.UserHeader); header != "" {
		normalized.UserHeader = header
	}
	if header := strings.TrimSpace(authConfig.EmailHeader); header != "" {
		normalized.EmailHeader = header
	}
	if header := strings.TrimSpace(authConfig.GroupsHeader); header != "" {
		normalized.GroupsHeader = header
	}
	if logoutPath := strings.TrimSpace(authConfig.LogoutPath); logoutPath != "" {
		normalized.LogoutPath = logoutPath
	}

	normalized.ViewerGroups = uniqueTrimmedStrings(authConfig.ViewerGroups)
	normalized.OperatorGroups = uniqueTrimmedStrings(authConfig.OperatorGroups)
	normalized.AdminGroups = uniqueTrimmedStrings(authConfig.AdminGroups)

	return normalized
}

func loadServerAuthConfig(configPath string) AuthConfig {
	config, err := loadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load auth config from %s, falling back to auth.mode=none: %v", configPath, err)
		return defaultAuthConfig()
	}

	normalized := normalizeAuthConfig(config.Auth)
	if validationErrors := validateAuthConfig(normalized); len(validationErrors) > 0 {
		log.Printf("Warning: invalid auth configuration in %s, forcing auth.mode=proxy: %s", configPath, strings.Join(validationErrors, "; "))
		normalized.Mode = "proxy"
	}

	return normalized
}

func uniqueTrimmedStrings(values []string) []string {
	seen := make(map[string]struct{})
	var normalized []string

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func splitHeaderValues(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t':
			return true
		default:
			return false
		}
	})
}

func parseGroups(value string) []string {
	parts := splitHeaderValues(value)
	return uniqueTrimmedStrings(parts)
}

func hasAnyGroup(groups []string, required []string) bool {
	if len(groups) == 0 || len(required) == 0 {
		return false
	}

	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[strings.ToLower(strings.TrimSpace(group))] = struct{}{}
	}

	for _, group := range required {
		if _, exists := groupSet[strings.ToLower(strings.TrimSpace(group))]; exists {
			return true
		}
	}

	return false
}

func firstHeaderValue(r *http.Request, primary string, fallbacks ...string) string {
	candidates := append([]string{primary}, fallbacks...)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		value := strings.TrimSpace(r.Header.Get(candidate))
		if value != "" {
			return value
		}
	}

	return ""
}

func (a authContext) sessionFromRequest(r *http.Request) AuthSession {
	session := AuthSession{
		Mode:        a.config.Mode,
		Role:        "viewer",
		Permissions: AuthPermissions{},
	}

	if a.config.Mode != "proxy" {
		session.Authenticated = true
		session.Username = "local"
		session.Role = "admin"
		session.Permissions.View = true
		session.Permissions.Operate = true
		session.Permissions.Admin = true
		return session
	}

	session.Username = firstHeaderValue(r, a.config.UserHeader, "X-Auth-Request-User")
	session.Email = firstHeaderValue(r, a.config.EmailHeader, "X-Auth-Request-Email")
	session.Groups = parseGroups(firstHeaderValue(r, a.config.GroupsHeader, "X-Auth-Request-Groups"))
	session.Authenticated = session.Username != "" || session.Email != ""

	if !session.Authenticated {
		session.Role = "unauthenticated"
		return session
	}

	session.LogoutURL = a.config.LogoutPath

	if hasAnyGroup(session.Groups, a.config.AdminGroups) {
		session.Role = "admin"
		session.Permissions.View = true
		session.Permissions.Operate = true
		session.Permissions.Admin = true
		return session
	}

	if hasAnyGroup(session.Groups, a.config.OperatorGroups) {
		session.Role = "operator"
		session.Permissions.View = true
		session.Permissions.Operate = true
		return session
	}

	if hasAnyGroup(session.Groups, a.config.ViewerGroups) {
		session.Role = "viewer"
		session.Permissions.View = true
		return session
	}

	session.Role = "forbidden"

	return session
}

func (a authContext) requireRunPermission(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := a.sessionFromRequest(r)
		if a.config.Mode == "proxy" && !session.Authenticated {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if !session.Permissions.Operate {
			http.Error(w, "Operator role required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (a authContext) requireViewPermission(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := a.sessionFromRequest(r)
		if a.config.Mode == "proxy" && !session.Authenticated {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if !session.Permissions.View {
			http.Error(w, "Viewer role required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func serve(port int, configPath string) {
	auth := authContext{config: loadServerAuthConfig(configPath)}

	// Serve de build directory
	subFS, err := fs.Sub(content, "frontend/build")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}

	fileServer := http.FileServer(http.FS(subFS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		if _, err := fs.Stat(subFS, requestPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexBytes, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			log.Printf("Error reading embedded index.html: %v", err)
			http.Error(w, "Could not read frontend index", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexBytes)
	})

	// Endpoint voor de results file
	http.HandleFunc("/results", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile("results.json")
		if err != nil {
			log.Printf("Error reading results file: %v", err)
			http.Error(w, "Could not read results file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))

	// Endpoint voor het uitvoeren van tests
	http.HandleFunc("/run-tests", auth.requireRunPermission(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cmd := getCommand(configPath)
		err := streamCommandOutput(w, cmd)
		if err != nil {
			log.Printf("Error running tests: %v", err)
			fmt.Fprintf(w, "\n[run failed] %v\n__CHECKY_CHECK_RUN_STATUS__:failed\n", err)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			return
		}
		fmt.Fprint(w, "\n[run completed]\n__CHECKY_CHECK_RUN_STATUS__:success\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))

	http.HandleFunc("/api/run-check", auth.requireRunPermission(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request TargetedRunRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		response, err := rerunCheck(configPath, request)
		if err != nil {
			log.Printf("Error rerunning check: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))

	http.HandleFunc("/api/auth/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(auth.sessionFromRequest(r))
	})

	// Endpoint om de versie te serveren
	http.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		versionResponse := map[string]string{"version": AppVersion}
		json.NewEncoder(w).Encode(versionResponse)
	})

	http.HandleFunc("/api/preflight", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		report := buildPreflightReport(configPath)
		json.NewEncoder(w).Encode(report)
	}))

	http.HandleFunc("/api/history/runs", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 && parsedLimit <= 200 {
				limit = parsedLimit
			}
		}

		runs, err := readRecentRuns(limit)
		if err != nil {
			log.Printf("Error reading history runs: %v", err)
			http.Error(w, "Could not read history runs", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(runs)
	}))

	http.HandleFunc("/api/history/events", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		limit := 20
		var runID *int64
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 && parsedLimit <= 500 {
				limit = parsedLimit
			}
		}
		if rawRunID := r.URL.Query().Get("run_id"); rawRunID != "" {
			parsedRunID, err := strconv.ParseInt(rawRunID, 10, 64)
			if err == nil && parsedRunID > 0 {
				runID = &parsedRunID
			}
		}

		events, err := readRecentEvents(limit, runID)
		if err != nil {
			log.Printf("Error reading history events: %v", err)
			http.Error(w, "Could not read history events", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(events)
	}))

	http.HandleFunc("/api/history/sparklines", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		limit := 14
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 && parsedLimit <= 60 {
				limit = parsedLimit
			}
		}

		metrics, err := readHostSparklineMetrics(limit)
		if err != nil {
			log.Printf("Error reading sparkline metrics: %v", err)
			http.Error(w, "Could not read sparkline metrics", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(metrics)
	}))

	http.HandleFunc("/api/history/check-detail", auth.requireViewPermission(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		host := strings.TrimSpace(r.URL.Query().Get("host"))
		checkName := strings.TrimSpace(r.URL.Query().Get("check_name"))
		if host == "" || checkName == "" {
			http.Error(w, "host and check_name are required", http.StatusBadRequest)
			return
		}

		limit := 20
		if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
			if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		detail, err := readCheckHistoryDetail(host, checkName, limit)
		if err != nil {
			log.Printf("Error reading check history detail: %v", err)
			http.Error(w, "Could not read check history detail", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(detail)
	}))

	// Start de server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func streamCommandOutput(w http.ResponseWriter, cmd *exec.Cmd) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming is not supported by this response writer")
	}

	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter
	cmd.Stderr = pipeWriter

	if err := cmd.Start(); err != nil {
		pipeWriter.Close()
		pipeReader.Close()
		return err
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
		pipeWriter.Close()
	}()

	buffer := make([]byte, 4096)
	for {
		readCount, readErr := pipeReader.Read(buffer)
		if readCount > 0 {
			if _, err := w.Write(buffer[:readCount]); err != nil {
				return err
			}
			flusher.Flush()
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return <-waitCh
}
