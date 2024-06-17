package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"strconv"
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

	var resultFile ResultFile
	if err := json.Unmarshal(data, &resultFile); err != nil {
		return fmt.Errorf("unable to parse result file: %v", err)
	}

	allChecks := make([]string, 0, len(resultFile.Checks))
	for check := range resultFile.Checks {
		allChecks = append(allChecks, check)
	}

	sort.Strings(allChecks)

	htmlContent := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>%s</title>
		<style>
			%s
			.chart-container {
				width: 400px;
				margin: auto;
			}
		</style>
		<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
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

	for checkName, check := range resultFile.Checks {
		valueCounts := make(map[string]int)
		passedCount := 0
		failedCount := 0
		passedHosts := []CheckResult{}
		failedHosts := []CheckResult{}
		for host, hostResults := range resultFile.Results {
			result, exists := hostResults[checkName]
			if exists {
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

		chartData := ""
		if check.Graph.Show {
			switch check.Graph.Type {
			case "bar_grouped_by_value":
				chartData, err = generateBarGroupedByValueChart(checkName, check, valueCounts)
			case "bar_grouped_by_10_percentile":
				chartData, err = generateBarGroupedBy10PercentileChart(checkName, check, resultFile.Results)
			case "pie_grouped_by_value":
				chartData, err = generatePieGroupedByValueChart(checkName, check, valueCounts)
			default:
				log.Fatalf("unknown graph type: %s", check.Graph.Type)
			}
			if err != nil {
				return fmt.Errorf("unable to create chart: %v", err)
			}

			fmt.Println("Chart data generated for", checkName)
		}

		htmlContent += fmt.Sprintf(`
		<h2>%s</h2>
		<p>%s</p>
		<pre><code>%s</code></pre>
		<p>Passed: %d, Failed: %d</p>`, check.Title, check.Description, check.Command, passedCount, failedCount)

		if check.Graph.Show {
			htmlContent += chartData
		}

		htmlContent += `<h3>Passed Hosts</h3><ul>`

		for _, result := range passedHosts {
			htmlContent += fmt.Sprintf("<li>%s (Datum: %s, Waarde: %s, Vars: %v)</li>", result.Host, result.Timestamp, result.Value, result.Vars)
		}
		htmlContent += "</ul>"

		htmlContent += "<h3>Failed Hosts</h3><ul>"

		for _, result := range failedHosts {
			htmlContent += fmt.Sprintf("<li>%s (Datum: %s, Waarde: %s, Vars: %v)</li>", result.Host, result.Timestamp, result.Value, result.Vars)
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

func generateBarGroupedByValueChart(checkName string, check Check, valueCounts map[string]int) (string, error) {
	labels := make([]string, 0, len(valueCounts))
	counts := make([]int, 0, len(valueCounts))
	for label, count := range valueCounts {
		labels = append(labels, label)
		counts = append(counts, count)
	}

	chartData := fmt.Sprintf(`
	<div class="chart-container">
		<canvas id="%s" width="400" height="300"></canvas>
	</div>
	<script>
		var ctx = document.getElementById('%s').getContext('2d');
		var chart = new Chart(ctx, {
			type: 'bar',
			data: {
				labels: %s,
				datasets: [{
					label: '%s',
					data: %s,
					backgroundColor: 'rgba(75, 192, 192, 0.2)',
					borderColor: 'rgba(75, 192, 192, 1)',
					borderWidth: 1,
					barPercentage: 0.5,
					categoryPercentage: 0.5
				}]
			},
			options: {
				scales: {
					y: {
						beginAtZero: true
					}
				}
			}
		});
	</script>`, checkName, checkName, marshalJSON(labels), check.Graph.Title, marshalJSON(counts))

	return chartData, nil
}

func generateBarGroupedBy10PercentileChart(checkName string, check Check, results map[string]map[string]CheckResult) (string, error) {
	percentiles := make([]int, 11)

	for _, hostResults := range results {
		result, exists := hostResults[checkName]
		if exists {
			value, err := strconv.Atoi(result.Value)
			if err != nil {
				return "", err
			}
			bucket := int(math.Min(float64(value/10), 10))
			percentiles[bucket]++
		}
	}

	labels := make([]string, 11)
	counts := make([]int, 11)
	for i := 0; i <= 10; i++ {
		labels[i] = fmt.Sprintf("%d-%d%%", i*10, (i+1)*10-1)
		if i == 10 {
			labels[i] = "100%"
		}
		counts[i] = percentiles[i]
	}

	chartData := fmt.Sprintf(`
	<div class="chart-container">
		<canvas id="%s" width="400" height="300"></canvas>
	</div>
	<script>
		var ctx = document.getElementById('%s').getContext('2d');
		var chart = new Chart(ctx, {
			type: 'bar',
			data: {
				labels: %s,
				datasets: [{
					label: '%s',
					data: %s,
					backgroundColor: 'rgba(153, 102, 255, 0.2)',
					borderColor: 'rgba(153, 102, 255, 1)',
					borderWidth: 1,
					barPercentage: 0.5,
					categoryPercentage: 0.5
				}]
			},
			options: {
				scales: {
					y: {
						beginAtZero: true
					}
				}
			}
		});
	</script>`, checkName, checkName, marshalJSON(labels), check.Graph.Title, marshalJSON(counts))

	return chartData, nil
}

func generatePieGroupedByValueChart(checkName string, check Check, valueCounts map[string]int) (string, error) {
	labels := make([]string, 0, len(valueCounts))
	counts := make([]int, 0, len(valueCounts))
	for label, count := range valueCounts {
		labels = append(labels, label)
		counts = append(counts, count)
	}

	colors := generateColorPalette(len(labels))

	chartData := fmt.Sprintf(`
	<div class="chart-container">
		<canvas id="%s" width="400" height="300"></canvas>
	</div>
	<script>
		var ctx = document.getElementById('%s').getContext('2d');
		var chart = new Chart(ctx, {
			type: 'pie',
			data: {
				labels: %s,
				datasets: [{
					label: '%s',
					data: %s,
					backgroundColor: %s
				}]
			},
			options: {
				responsive: true,
				plugins: {
					legend: {
						position: 'top',
					},
					tooltip: {
						callbacks: {
							label: function(context) {
								var label = context.label || '';
								if (label) {
									label += ': ';
								}
								label += context.raw;
								return label;
							}
						}
					}
				}
			}
		});
	</script>`, checkName, checkName, marshalJSON(labels), check.Graph.Title, marshalJSON(counts), marshalJSON(colors))

	return chartData, nil
}

func generateColorPalette(size int) []string {
	colors := make([]string, size)
	for i := 0; i < size; i++ {
		colors[i] = fmt.Sprintf("hsl(%d, 70%%, 50%%)", int(float64(i)/float64(size)*360))
	}
	return colors
}

func marshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("unable to marshal JSON: %v", err)
	}
	return string(data)
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

func main() {
	configPath := "config.json"
	err := generateReport(configPath)
	if err != nil {
		log.Fatalf("Error generating report: %v", err)
	}
}
