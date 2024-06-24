package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
)

// Embed de gehele build directory
//go:embed frontend/build/*
var content embed.FS

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

	// Endpoint om de tests opnieuw uit te voeren
	http.HandleFunc("/run-tests", func(w http.ResponseWriter, r *http.Request) {
		cmd := exec.Command("go", "run", "main.go", "remote_check.go", "types.go", "helpers.go", "-mode=check", "-config="+configPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error running tests: %v", err)
			http.Error(w, string(output), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(output)
	})

	// Start de server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
