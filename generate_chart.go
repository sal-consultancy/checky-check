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

func generateReport(configPath string) error {
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("unable to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("unable to parse config file: %v", err)
	}

	data, err := ioutil.ReadFile("results.json")
	if err != nil {
		return fmt.Errorf("unable to read result file: %v", err)
	}

	var results []CheckResult
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("unable to parse result file: %v", err)
	}

	allChecksSet := make(map[string]struct{})
	for _, group := range config.HostGroups {
		for hostName, hostConfig := range group.Hosts {
			checks := getEffectiveChecks(config, hostName, hostConfig, group.HostVars)
			for _, check := range checks {
				allChecksSet[check] = struct{}{}
			}
		}
	}

	allChecks := make([]string, 0, len(allChecksSet))
	for check := range allChecksSet {
		allChecks = append(allChecks, check)
	}

	sort.Strings(allChecks)

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

	for _, check := range allChecks {
		htmlContent += fmt.Sprintf("<li>%s</li>", check)
	}
	htmlContent += "</ul>"

	for checkName, check := range config.Checks {
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
			continue
		}

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
			return fmt.Errorf("unable to create chart: %v", err)
		}

		fmt.Println("Chart saved as", svgFileName)

		htmlContent += fmt.Sprintf(`
		<h2>%s</h2>
		<p>%s</p>
		<pre><code>%s</code></pre>
		<p>Passed: %d, Failed: %d</p>
		<img src="%s" alt="Chart">
		<h3>Passed Hosts</h3>
		<ul>`, check.Graph.Title, check.Description, check.Command, passedCount, failedCount, svgFileName)

		for _, result := range passedHosts {
			htmlContent += fmt.Sprintf("<li>%s (Datum: %s, Waarde: %s)</li>", result.Host, result.Timestamp, result.Value)
		}
		htmlContent += "</ul>"

		htmlContent += "<h3>Failed Hosts</h3><ul>"

		for _, result := range failedHosts {
			htmlContent += fmt.Sprintf("<li>%s (Datum: %s, Waarde: %s)</li>", result.Host, result.Timestamp, result.Value)
		}
		htmlContent += "</ul>"
	}

	htmlContent += `
	</body>
	</html>`

	htmlFileName := "all_charts.html"
	err = ioutil.WriteFile(htmlFileName, []byte(htmlContent), 0644)
	if err != nil {
		return fmt.Errorf("unable to write HTML file: %v", err)
	}

	fmt.Println("HTML file saved as", htmlFileName)
	return nil
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
	percentiles := make([]int, 11)

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

func getEffectiveChecks(config Config, hostName string, hostConfig Host, groupVars map[string]string) []string {
	checkSet := make(map[string]struct{})

	for _, check := range config.HostDefaults.HostChecks {
		checkSet[check] = struct{}{}
	}

	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists {
			for _, check := range template.HostChecks {
				checkSet[check] = struct{}{}
			}
		}
	}

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
