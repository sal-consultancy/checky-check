package main

import (
	"embed"
	"flag"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
)

// Embed de gehele build directory
//go:embed frontend/build/*
var content embed.FS

func main() {
	configPath := flag.String("config", "config.json", "Path to the config file")
	mode := flag.String("mode", "check", "Mode to run: check or report")
	flag.Parse()

	if *mode == "check" {
		runChecks(*configPath)
	} else if *mode == "report" {
		// Implement report generation
	} else {
		log.Fatalf("Unknown mode: %s", *mode)
	}

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
	log.Println("Starting server on :8070")
	if err := http.ListenAndServe(":8070", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
