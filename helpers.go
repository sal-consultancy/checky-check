package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"os"
    "regexp"
)

func evaluateCondition(output string, failWhen string, failValue interface{}) bool {
	output = strings.TrimSpace(output)

	switch failWhen {
	case ">":
		outputVal, err := strconv.ParseFloat(output, 64)
		if err != nil {
			log.Printf("Error parsing output value: %v\n", err)
			return false
		}
		failVal, err := strconv.ParseFloat(failValue.(string), 64)
		if err != nil {
			log.Printf("Error parsing fail value: %v\n", err)
			return false
		}
		return outputVal > failVal
	case "<":
		outputVal, err := strconv.ParseFloat(output, 64)
		if err != nil {
			log.Printf("Error parsing output value: %v\n", err)
			return false
		}
		failVal, err := strconv.ParseFloat(failValue.(string), 64)
		if err != nil {
			log.Printf("Error parsing fail value: %v\n", err)
			return false
		}
		return outputVal < failVal
	case "==", "=":
		if fv, ok := failValue.([]interface{}); ok {
			for _, val := range fv {
				if output == val.(string) {
					return true
				}
			}
			return false
		}
		return output == failValue.(string)
	case "!=":
		if fv, ok := failValue.([]interface{}); ok {
			for _, val := range fv {
				if output == val.(string) {
					return false
				}
			}
			return true
		}
		return output != failValue.(string)
	case "is":
		return output == failValue.(string)
	case "is not":
		return output != failValue.(string)
	default:
		log.Printf("Unknown fail condition: %s\n", failWhen)
		return false
	}
}

func replaceVariables(command string, vars map[string]string) string {
	for key, value := range vars {
		placeholder := fmt.Sprintf("${%s}", key)
		command = strings.ReplaceAll(command, placeholder, value)
	}
	return command
}

func mergeVars(varsList ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, vars := range varsList {
		for k, v := range vars {
			result[k] = v
		}
	}
	return result
}

func substituteEnvVariables(configStr string) (string, error) {
    re := regexp.MustCompile(`\$\{env\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
    matches := re.FindAllStringSubmatch(configStr, -1)

    for _, match := range matches {
        if len(match) != 2 {
            continue
        }
        envVar := match[1]
        envValue := os.Getenv(envVar)
        if envValue == "" {
            return "", fmt.Errorf("environment variable %s not set", envVar)
        }
        configStr = strings.ReplaceAll(configStr, match[0], envValue)
    }

    return configStr, nil
}