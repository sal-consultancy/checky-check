package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"encoding/json"
	"strings"
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
	case "report":
		generateReport(*configPath)
	case "serve":
		serve(*port)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func serve(port int) {
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

	// Start de server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}



func generateReport(configPath string) {
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("unable to read config file: %v", err)
	}

	var config struct {
		Checks map[string]Check `json:"checks"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatalf("unable to parse config file: %v", err)
	}

	resultsData, err := ioutil.ReadFile("results.json")
	if err != nil {
		log.Fatalf("unable to read results file: %v", err)
	}

	var results map[string]map[string]CheckResult
	if err := json.Unmarshal(resultsData, &results); err != nil {
		log.Fatalf("unable to parse results file: %v", err)
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		log.Fatalf("unable to marshal results to JSON: %v", err)
	}

	htmlContent, err := content.ReadFile("frontend/build/index.html")
	if err != nil {
		log.Fatalf("unable to read index.html: %v", err)
	}

	reportContent := strings.Replace(string(htmlContent), "window.__CHECK_RESULTS__ = null;", fmt.Sprintf("window.__CHECK_RESULTS__ = %s;", resultsJSON), 1)

	if err := ioutil.WriteFile("report.html", []byte(reportContent), 0644); err != nil {
		log.Fatalf("unable to write report file: %v", err)
	}

	log.Println("Report generated as report.html")
}