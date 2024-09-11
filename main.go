package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"encoding/json"
)

// Embed de gehele build directory
//go:embed frontend/build/*
var content embed.FS

var version string

var AppVersion = "development" // Standaard versie voor lokale ontwikkeling

func init() {
	data, err := ioutil.ReadFile("version.txt")
	if err != nil {
		log.Fatalf("Failed to read version.txt: %v", err)
	}
	version = strings.TrimSpace(string(data))
}

func main() {
	configPath := flag.String("config", "config.json", "Path to the config file")
	mode := flag.String("mode", "check", "Mode to run: check, report, or serve")
	port := flag.Int("port", 8070, "Port to run the server on")
	flag.Parse()

	switch *mode {
	case "check":
		runChecks(*configPath)

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

	http.Handle("/", http.FileServer(http.FS(subFS)))

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


	// Endpoint om de versie te serveren
	http.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		versionResponse := map[string]string{"version": AppVersion}
		json.NewEncoder(w).Encode(versionResponse)
	})

	// Start de server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
