package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func evaluateCondition(output string, failWhen string, failValue interface{}) bool {
	output = strings.TrimSpace(output)
	failValues := parseFailValues(failValue)

	switch failWhen {
	case ">":
		outputVal, err := strconv.ParseFloat(output, 64)
		if err != nil {
			log.Printf("Error parsing output value: %v\n", err)
			return false
		}
		for _, failValStr := range failValues {
			failVal, err := strconv.ParseFloat(failValStr, 64)
			if err != nil {
				log.Printf("Error parsing fail value: %v\n", err)
				return false
			}
			if outputVal > failVal {
				return true
			}
		}
		return false
	case "<":
		outputVal, err := strconv.ParseFloat(output, 64)
		if err != nil {
			log.Printf("Error parsing output value: %v\n", err)
			return false
		}
		for _, failValStr := range failValues {
			failVal, err := strconv.ParseFloat(failValStr, 64)
			if err != nil {
				log.Printf("Error parsing fail value: %v\n", err)
				return false
			}
			if outputVal < failVal {
				return true
			}
		}
		return false
	case "==", "=":
		for _, failValStr := range failValues {
			if output == failValStr {
				return true
			}
		}
		return false
	case "!=":
		for _, failValStr := range failValues {
			if output == failValStr {
				return false
			}
		}
		return true
	case "is":
		for _, failValStr := range failValues {
			if output == failValStr {
				return true
			}
		}
		return false
	case "is not":
		for _, failValStr := range failValues {
			if output == failValStr {
				return false
			}
		}
		return true
	default:
		log.Printf("Unknown fail condition: %s\n", failWhen)
		return false
	}
}

func parseFailValues(failValue interface{}) []string {
	switch v := failValue.(type) {
	case string:
		return []string{v}
	case []interface{}:
		failVals := make([]string, len(v))
		for i, val := range v {
			failVals[i] = fmt.Sprintf("%v", val)
		}
		return failVals
	case []string:
		return v
	default:
		return []string{}
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

func runCommandWithTimeout(user, host string, authMethods []ssh.AuthMethod, command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), sshConfig)
	if err != nil {
		return "", fmt.Errorf("failed to dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	type result struct {
		output string
		err    error
	}

	ch := make(chan result, 1)
	go func() {
		output, err := session.CombinedOutput(command)
		ch <- result{output: string(output), err: err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return "", fmt.Errorf("failed to run command: %v", res.err)
		}
		return res.output, nil
	case <-ctx.Done():
		log.Printf("Command timed out after %v: %s on host %s", timeout, command, host)
		return "", fmt.Errorf("command timed out after %v", timeout)
	}
}

func parseTimeout(timeoutStr string) time.Duration {
	if timeoutStr == "" {
		return 30 * time.Second // default timeout
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Printf("Invalid timeout format: %v, using default 30s", err)
		return 30 * time.Second
	}
	return timeout
}
