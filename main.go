package main

import (
	"flag"
	"log"
)

func main() {
	configPath := flag.String("config", "config.json", "path to the configuration file")
	mode := flag.String("mode", "", "mode of operation: check or report")
	flag.Parse()

	if *mode == "" {
		log.Fatal("mode is required: specify -mode=check or -mode=report")
	}

	switch *mode {
	case "check":
		err := runChecks(*configPath)
		if err != nil {
			log.Fatalf("error running checks: %v", err)
		}
	case "report":
		err := generateReport(*configPath)
		if err != nil {
			log.Fatalf("error generating report: %v", err)
		}
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}
