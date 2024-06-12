package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Identities    map[string]Identity     `json:"identities"`
	HostDefaults  HostDefaults            `json:"host_defaults"`
	HostTemplates map[string]HostTemplate `json:"host_templates"`
	Checks        map[string]Check        `json:"checks"`
	HostGroups    map[string]HostGroup    `json:"host_groups"`
	Report        Report                  `json:"report"`
}

type Identity struct {
	User       string `json:"user"`
	Key        string `json:"key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Password   string `json:"password,omitempty"`
}

type HostDefaults struct {
	Identity   string            `json:"identity"`
	HostVars   map[string]string `json:"host_vars"`
	HostChecks []string          `json:"host_checks"`
}

type HostTemplate struct {
	HostVars   map[string]string `json:"host_vars,omitempty"`
	HostChecks []string          `json:"host_checks"`
}

type HostGroup struct {
	HostVars map[string]string `json:"host_vars,omitempty"`
	Hosts    map[string]Host   `json:"hosts"`
}

type Check struct {
	Command     string      `json:"command,omitempty"`
	Service     string      `json:"service,omitempty"`
	FailWhen    string      `json:"fail_when"`
	FailValue   string      `json:"fail_value"`
	Description string      `json:"description,omitempty"`
	Graph       GraphConfig `json:"graph,omitempty"`
}

type Host struct {
	Identity     string            `json:"identity,omitempty"`
	HostTemplate string            `json:"host_template,omitempty"`
	HostVars     map[string]string `json:"host_vars,omitempty"`
	HostChecks   []string          `json:"host_checks"`
}

type CheckResult struct {
	Host      string `json:"host"`
	Check     string `json:"check"`
	Status    string `json:"status"`
	Value     string `json:"value"`
	Timestamp string `json:"timestamp"`
}

type Report struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CSS         string `json:"css"`
}

type GraphConfig struct {
	Title string `json:"title"`
	Type  string `json:"type"`
}

func getSSHAuthMethod(identity Identity) []ssh.AuthMethod {
	var authMethods []ssh.AuthMethod

	if identity.Password != "" {
		authMethods = append(authMethods, ssh.Password(identity.Password))
	}

	if identity.Key != "" {
		buffer, err := ioutil.ReadFile(filepath.Clean(identity.Key))
		if err != nil {
			log.Fatalf("unable to read private key: %v", err)
		}

		var key ssh.Signer
		if identity.Passphrase == "" {
			key, err = ssh.ParsePrivateKey(buffer)
		} else {
			key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(identity.Passphrase))
		}

		if err != nil {
			log.Fatalf("unable to parse private key: %v", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	return authMethods
}

func runCommand(user, host string, authMethods []ssh.AuthMethod, command string) (string, error) {
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), sshConfig)
	if err != nil {
		log.Printf("Failed to dial: %v", err)
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		log.Printf("Failed to run command: %v", err)
		return "", err
	}

	return string(output), nil
}

func checkServiceStatus(user, host string, authMethods []ssh.AuthMethod, service string) (string, error) {
	command := fmt.Sprintf("systemctl is-active %s", service)
	result, err := runCommand(user, host, authMethods, command)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func evaluateCondition(output string, failWhen string, failValue string) bool {
	output = strings.TrimSpace(output)
	failValue = strings.TrimSpace(failValue)

	switch failWhen {
	case ">":
		outputVal, err := strconv.ParseFloat(output, 64)
		if err != nil {
			log.Printf("Error parsing output value: %v\n", err)
			return false
		}
		failVal, err := strconv.ParseFloat(failValue, 64)
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
		failVal, err := strconv.ParseFloat(failValue, 64)
		if err != nil {
			log.Printf("Error parsing fail value: %v\n", err)
			return false
		}
		return outputVal < failVal
	case "==", "=":
		return output == failValue
	case "!=":
		return output != failValue
	case "is":
		return output == failValue
	case "is not":
		return output != failValue
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

func replaceEnvVariables(value string) string {
	const envPrefix = "${env."
	for {
		startIdx := strings.Index(value, envPrefix)
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(value[startIdx:], "}")
		if endIdx == -1 {
			break
		}
		endIdx += startIdx
		envVar := value[startIdx+len(envPrefix) : endIdx]
		envVal := os.Getenv(envVar)
		value = strings.Replace(value, value[startIdx:endIdx+1], envVal, 1)
	}
	return value
}

func loadConfig(configPath string) (Config, error) {
	var config Config
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("unable to read config file: %v", err)
	}

	configStr := replaceEnvVariables(string(configFile))

	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return config, fmt.Errorf("unable to parse config file: %v", err)
	}

	return config, nil
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

func runChecksOnHost(config Config, host string, hostConfig Host, groupVars map[string]string, wg *sync.WaitGroup, logger *log.Logger, results *[]CheckResult) {
	defer wg.Done()

	logger.Printf("Running checks on host: %s", host)

	// Combineer variabelen in de volgorde: defaults -> template -> groep -> host
	var combinedVars map[string]string
	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists {
			combinedVars = mergeVars(config.HostDefaults.HostVars, template.HostVars, groupVars, hostConfig.HostVars)
		} else {
			combinedVars = mergeVars(config.HostDefaults.HostVars, groupVars, hostConfig.HostVars)
		}
	} else {
		combinedVars = mergeVars(config.HostDefaults.HostVars, groupVars, hostConfig.HostVars)
	}

	logger.Printf("Combined vars for host %s: %v", host, combinedVars)

	// Combineer checks in de volgorde: defaults -> template -> groep -> host
	var combinedChecks []string
	combinedChecks = append(combinedChecks, config.HostDefaults.HostChecks...)
	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists {
			combinedChecks = append(combinedChecks, template.HostChecks...)
		}
	}
	combinedChecks = append(combinedChecks, hostConfig.HostChecks...)

	logger.Printf("Combined checks for host %s: %v", host, combinedChecks)

	// Combineer identiteit in de volgorde: defaults -> template -> groep -> host
	identityName := config.HostDefaults.Identity
	if hostConfig.HostTemplate != "" {
		template, exists := config.HostTemplates[hostConfig.HostTemplate]
		if exists && template.HostVars != nil {
			if id, exists := template.HostVars["identity"]; exists {
				identityName = id
			}
		}
	}
	if groupVars != nil {
		if id, exists := groupVars["identity"]; exists {
			identityName = id
		}
	}
	if hostConfig.Identity != "" {
		identityName = hostConfig.Identity
	}

	identity, exists := config.Identities[identityName]
	if !exists {
		logger.Fatalf("Identity %s not found for host %s", identityName, host)
		return
	}

	logger.Printf("Using identity for host %s: %v", host, identity)

	authMethods := getSSHAuthMethod(identity)

	for _, checkName := range combinedChecks {
		check, exists := config.Checks[checkName]
		if !exists {
			logger.Printf("Check %s not defined in config\n", checkName)
			continue
		}

		logger.Printf("Running check %s on host %s", checkName, host)

		var result string
		var err error
		var checkFailed bool

		if check.Command != "" {
			command := replaceVariables(check.Command, combinedVars)
			logger.Printf("Running command on host %s: %s", host, command)
			result, err = runCommand(identity.User, host, authMethods, command)
			if err != nil {
				logger.Printf("Failed to run command %s on host %s: %v\n", command, host, err)
				continue
			}
			checkFailed = evaluateCondition(result, check.FailWhen, check.FailValue)
		} else if check.Service != "" {
			logger.Printf("Checking service %s on host %s", check.Service, host)
			result, err = checkServiceStatus(identity.User, host, authMethods, check.Service)
			if err != nil {
				logger.Printf("Failed to check service %s status on host %s: %v\n", check.Service, host, err)
				continue
			}
			checkFailed = evaluateCondition(result, check.FailWhen, check.FailValue)
		}

		status := "passed"
		if checkFailed {
			status = "failed"
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("%s - Host: %s - Check: %s - Status: %s - Value: %s\n", timestamp, host, checkName, status, strings.TrimSpace(result))

		// Bewaar het resultaat
		*results = append(*results, CheckResult{
			Host:      host,
			Check:     checkName,
			Status:    status,
			Value:     strings.TrimSpace(result),
			Timestamp: timestamp,
		})
	}
}

func runChecks(configPath string) error {
	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	// Maak een logbestand aan voor gedetailleerde logging
	logFile, err := os.OpenFile("remote_check.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open log file: %v", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	// Gebruik een WaitGroup om te wachten tot alle goroutines klaar zijn
	var wg sync.WaitGroup

	// Verzamelen van resultaten
	var results []CheckResult

	// Verwerk elke host in de hostgroepen en voer de opgegeven checks parallel uit
	for _, group := range config.HostGroups {
		groupVars := group.HostVars
		for host, hostConfig := range group.Hosts {
			wg.Add(1)
			go runChecksOnHost(config, host, hostConfig, groupVars, &wg, logger, &results)
		}
	}

	// Wacht tot alle goroutines klaar zijn
	wg.Wait()

	// Schrijf de resultaten naar een bestand
	resultFile, err := os.Create("results.json")
	if err != nil {
		return fmt.Errorf("unable to create result file: %v", err)
	}
	defer resultFile.Close()

	if err := json.NewEncoder(resultFile).Encode(results); err != nil {
		return fmt.Errorf("unable to write results to file: %v", err)
	}

	return nil
}
