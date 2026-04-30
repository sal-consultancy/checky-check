package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
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
	configPath := flag.String("config", "config.yaml", "Path to the config file or config directory")
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

func serve(port int, configPath string) {
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
	http.HandleFunc("/results", func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile("results.json")
		if err != nil {
			log.Printf("Error reading results file: %v", err)
			http.Error(w, "Could not read results file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	// Endpoint voor het uitvoeren van tests
	http.HandleFunc("/run-tests", func(w http.ResponseWriter, r *http.Request) {
		cmd := getCommand(configPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error running tests: %v", err)
			http.Error(w, fmt.Sprintf("Error running tests: %v\nOutput: %s", err, output), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	})

	http.HandleFunc("/api/run-check", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// Endpoint om de versie te serveren
	http.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		versionResponse := map[string]string{"version": AppVersion}
		json.NewEncoder(w).Encode(versionResponse)
	})

	http.HandleFunc("/api/history/runs", func(w http.ResponseWriter, r *http.Request) {
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
	})

	http.HandleFunc("/api/history/events", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// Start de server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
