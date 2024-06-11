package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type CheckResult struct {
	Host      string `json:"host"`
	Check     string `json:"check"`
	Status    string `json:"status"`
	Value     string `json:"value"`
	Timestamp string `json:"timestamp"`
}

type Check struct {
	Description string `json:"description"`
	Graph       struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	} `json:"graph"`
	Command   string `json:"command"`
	FailWhen  string `json:"fail_when"`
	FailValue string `json:"fail_value"`
}

type Report struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CSS         string `json:"css"`
}

type Config struct {
	Report       Report           `json:"report"`
	Checks       map[string]Check `json:"checks"`
	HostDefaults struct {
		HostChecks []string `json:"host_checks"`
	} `json:"host_defaults"`
	HostTemplates map[string]struct {
		HostChecks []string `json:"host_checks"`
	} `json:"host_templates"`
	Hosts map[string]struct {
		HostTemplate string   `json:"host_template"`
		HostChecks   []string `json:"host_checks"`
	} `json:"hosts"`
}

func getEffectiveChecks(config Config, hostName string, hostConfig struct {
	HostTemplate string   `json:"host_template"`
	HostChecks   []string `json:"host_checks"`
}) []string {
	checkSet := make(map[string]struct{})

	// Voeg de standaard checks toe
	for _, check := range config.HostDefaults.HostChecks {
		checkSet[check] = struct{}{}
	}

	// Voeg de template checks toe
	if template, ok := config.HostTemplates[hostConfig.HostTemplate]; ok {
		for _, check := range template.HostChecks {
			checkSet[check] = struct{}{}
		}
	}

	// Voeg de specifieke host checks toe
	for _, check := range hostConfig.HostChecks {
		checkSet[check] = struct{}{}
	}

	checks := make([]string, 0, len(checkSet))
	for check := range checkSet {
		checks = append(checks, check)
	}

	sort.Strings(checks)
	return checks
}

func plotBarGroupedByValue(checkName string, check Check, results []CheckResult) (string, error) {
	valueCounts := make(map[string]int)
	for _, result := range results {
		if result.Check == checkName {
			valueCounts[result.Value]++
		}
	}

	labels := make([]string, 0, len(valueCounts))
	counts := make([]float64, 0, len(valueCounts))
	for label, count := range valueCounts {
		labels = append(labels, label)
		counts = append(counts, float64(count))
	}

	p := plot.New()
	p.Title.Text = check.Graph.Title
	p.X.Label.Text = "Value"
	p.Y.Label.Text = "Count"

	barWidth := vg.Points(20)
	bars, err := plotter.NewBarChart(plotter.Values(counts), barWidth)
	if err != nil {
		return "", err
	}

	p.Add(bars)
	p.NominalX(labels...)

	svgFileName := fmt.Sprintf("%s.svg", checkName)
	err = p.Save(6*vg.Inch, 4*vg.Inch, svgFileName)
	if err != nil {
		return "", err
	}

	return svgFileName, nil
}

func plotBarGroupedBy10Percentile(checkName string, check Check, results []CheckResult) (string, error) {
	percentiles := make([]int, 11) // 0-10, 11 groepen

	for _, result := range results {
		if result.Check == checkName {
			value, err := strconv.Atoi(result.Value)
			if err != nil {
				return "", err
			}
			bucket := int(math.Min(float64(value/10), 10))
			percentiles[bucket]++
		}
	}

	labels := make([]string, 11)
	counts := make([]float64, 11)
	for i := 0; i <= 10; i++ {
		labels[i] = fmt.Sprintf("%d-%d%%", i*10, (i+1)*10-1)
		if i == 10 {
			labels[i] = "100%"
		}
		counts[i] = float64(percentiles[i])
	}

	p := plot.New()
	p.Title.Text = check.Graph.Title
	p.X.Label.Text = "Percentage Range"
	p.Y.Label.Text = "Count"

	barWidth := vg.Points(20)
	bars, err := plotter.NewBarChart(plotter.Values(counts), barWidth)
	if err != nil {
		return "", err
	}

	p.Add(bars)
	p.NominalX(labels...)

	svgFileName := fmt.Sprintf("%s.svg", checkName)
	err = p.Save(6*vg.Inch, 4*vg.Inch, svgFileName)
	if err != nil {
		return "", err
	}

	return svgFileName, nil
}

func main() {
	// Laad de configuratie uit het JSON-bestand
	configData, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalf("unable to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatalf("unable to parse config file: %v", err)
	}

	// Laad de resultaten uit het JSON-bestand
	data, err := ioutil.ReadFile("results.json")
	if err != nil {
		log.Fatalf("unable to read result file: %v", err)
	}

	var results []CheckResult
	if err := json.Unmarshal(data, &results); err != nil {
		log.Fatalf("unable to parse result file: %v", err)
	}

	// Maak een set van alle uitgevoerde checks
	allChecksSet := make(map[string]struct{})
	for hostName, hostConfig := range config.Hosts {
		checks := getEffectiveChecks(config, hostName, hostConfig)
		for _, check := range checks {
			allChecksSet[check] = struct{}{}
		}
	}

	allChecks := make([]string, 0, len(allChecksSet))
	for check := range allChecksSet {
		allChecks = append(allChecks, check)
	}

	sort.Strings(allChecks)

	// HTML inhoud voor het rapport
	htmlContent := fmt.Sprintf(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>%s</title>
        <style>%s</style>
    </head>
    <body>
    <h1>%s</h1>
    <p>%s</p>
    <h2>Overzicht van uitgevoerde checks</h2>
    <ul>`, config.Report.Title, config.Report.CSS, config.Report.Title, config.Report.Description)

	// Voeg de lijst van alle uitgevoerde checks toe
	for _, check := range allChecks {
		htmlContent += fmt.Sprintf("<li>%s</li>", check)
	}
	htmlContent += "</ul>"

	// Verwerk elke check
	for checkName, check := range config.Checks {
		// Verzamel de waarden voor de geselecteerde check
		valueCounts := make(map[string]int)
		passedCount := 0
		failedCount := 0
		passedHosts := []CheckResult{}
		failedHosts := []CheckResult{}
		for _, result := range results {
			if result.Check == checkName {
				valueCounts[result.Value]++
				if result.Status == "passed" {
					passedCount++
					passedHosts = append(passedHosts, result)
				} else {
					failedCount++
					failedHosts = append(failedHosts, result)
				}
			}
		}

		if passedCount == 0 && failedCount == 0 {
			continue // Sla checks over die niet zijn uitgevoerd
		}

		// Maak de juiste grafiek op basis van het type
		var svgFileName string
		switch check.Graph.Type {
		case "bar_grouped_by_value":
			svgFileName, err = plotBarGroupedByValue(checkName, check, results)
		case "bar_grouped_by_10_percentile":
			svgFileName, err = plotBarGroupedBy10Percentile(checkName, check, results)
		default:
			log.Fatalf("unknown graph type: %s", check.Graph.Type)
		}
		if err != nil {
			log.Fatalf("unable to create chart: %v", err)
		}

		fmt.Println("Chart saved as", svgFileName)

		// Voeg de grafiek en beschrijving toe aan de HTML inhoud
		htmlContent += fmt.Sprintf(`
        <h2>%s</h2>
        <p>%s</p>
        <p>Passed: %d, Failed: %d</p>
        <img src="%s" alt="Chart">
        <h3>Passed Hosts</h3>
        <ul>`, check.Graph.Title, check.Description, passedCount, failedCount, svgFileName)

		// Voeg de lijst met geslaagde hosts toe
		for _, result := range passedHosts {
			htmlContent += fmt.Sprintf("<li>%s (Datum: %s, Waarde: %s)</li>", result.Host, result.Timestamp, result.Value)
		}
		htmlContent += "</ul>"

		htmlContent += "<h3>Failed Hosts</h3><ul>"

		// Voeg de lijst met mislukte hosts toe
		for _, result := range failedHosts {
			htmlContent += fmt.Sprintf("<li>%s (Datum: %s, Waarde: %s)</li>", result.Host, result.Timestamp, result.Value)
		}
		htmlContent += "</ul>"
	}

	// Sluit de HTML inhoud
	htmlContent += `
    </body>
    </html>`

	// Sla de HTML inhoud op in een bestand
	htmlFileName := "all_charts.html"
	err = ioutil.WriteFile(htmlFileName, []byte(htmlContent), 0644)
	if err != nil {
		log.Fatalf("unable to write HTML file: %v", err)
	}

	fmt.Println("HTML file saved as", htmlFileName)
}
