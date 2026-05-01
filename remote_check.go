package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type ResolvedCheck struct {
	Name string
	Vars VarMap
}

type HostCheckTarget struct {
	Host       string
	HostConfig Host
	HostGroup  HostGroup
}

type URLExecutionResult struct {
	StatusCode int
	Body       string
	LatencyMs  int64
	Redirected bool
	Location   string
	FinalURL   string
}

func describeErrorType(errorType string) string {
	switch errorType {
	case "variable_resolution_error":
		return "Variable Resolution Error"
	case "local_timeout":
		return "Local Timeout"
	case "local_command_error":
		return "Local Command Error"
	case "timeout":
		return "Timeout"
	case "ssh_auth_error":
		return "SSH Authentication Error"
	case "ssh_connection_error":
		return "SSH Connection Error"
	case "ssh_session_error":
		return "SSH Session Error"
	case "command_error":
		return "Command Error"
	case "url_timeout":
		return "URL Timeout"
	case "url_dns_error":
		return "URL DNS Error"
	case "url_tls_error":
		return "URL TLS Error"
	case "url_request_error":
		return "URL Request Error"
	case "url_content_mismatch":
		return "URL Content Mismatch"
	default:
		return "Execution Error"
	}
}

func errorDetailsFrom(err error, output string) (string, string, string) {
	trimmedOutput := strings.TrimSpace(output)
	if executionErr, ok := err.(*CheckExecutionError); ok {
		value := trimmedOutput
		if value == "" {
			value = describeErrorType(executionErr.Type)
		}
		return value, executionErr.Type, executionErr.Message
	}

	value := trimmedOutput
	if value == "" {
		value = "Execution Error"
	}
	return value, "execution_error", err.Error()
}

func buildResultFile(config Config, checks map[string]Check, urlChecks map[string]Check, results []CheckResult, urlResults map[string]CheckResult, status string, errors []string) ResultFile {
	return ResultFile{
		Checks:      checks,
		Results:     mapResults(results),
		URLChecks:   urlChecks,
		URLResults:  urlResults,
		Report:      config.Report,
		Status:      status,
		Errors:      errors,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
}

func writeResultFile(resultData ResultFile) error {
	resultFile, err := os.Create("results.json")
	if err != nil {
		return fmt.Errorf("unable to create result file: %w", err)
	}
	defer resultFile.Close()

	if err := json.NewEncoder(resultFile).Encode(resultData); err != nil {
		return fmt.Errorf("unable to write results to file: %w", err)
	}

	return nil
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

func runLocalCommand(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), &CheckExecutionError{
			Type:    "local_timeout",
			Message: fmt.Sprintf("local command timed out after %v", timeout),
			Output:  strings.TrimSpace(string(output)),
		}
	}
	if err != nil {
		log.Printf("Failed to run local command: %v", err)
		return string(output), &CheckExecutionError{
			Type:    "local_command_error",
			Message: fmt.Sprintf("local command failed: %v", err),
			Output:  strings.TrimSpace(string(output)),
		}
	}

	return string(output), nil
}

func classifyURLRequestError(targetURL string, timeout time.Duration, err error) *CheckExecutionError {
	if errors.Is(err, context.DeadlineExceeded) {
		return &CheckExecutionError{
			Type:    "url_timeout",
			Message: fmt.Sprintf("url request to %s timed out after %v", targetURL, timeout),
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &CheckExecutionError{
			Type:    "url_timeout",
			Message: fmt.Sprintf("url request to %s timed out after %v", targetURL, timeout),
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &CheckExecutionError{
			Type:    "url_dns_error",
			Message: fmt.Sprintf("url request to %s failed DNS lookup: %v", targetURL, dnsErr),
		}
	}

	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthorityErr) {
		return &CheckExecutionError{
			Type:    "url_tls_error",
			Message: fmt.Sprintf("url request to %s failed TLS validation: %v", targetURL, unknownAuthorityErr),
		}
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return &CheckExecutionError{
			Type:    "url_tls_error",
			Message: fmt.Sprintf("url request to %s failed TLS validation: %v", targetURL, hostnameErr),
		}
	}

	var certInvalidErr x509.CertificateInvalidError
	if errors.As(err, &certInvalidErr) {
		return &CheckExecutionError{
			Type:    "url_tls_error",
			Message: fmt.Sprintf("url request to %s failed TLS validation: %v", targetURL, certInvalidErr),
		}
	}

	var systemRootsErr x509.SystemRootsError
	if errors.As(err, &systemRootsErr) {
		return &CheckExecutionError{
			Type:    "url_tls_error",
			Message: fmt.Sprintf("url request to %s failed TLS validation: %v", targetURL, systemRootsErr),
		}
	}

	var urlErr *neturl.Error
	if errors.As(err, &urlErr) {
		if strings.Contains(strings.ToLower(urlErr.Err.Error()), "x509") {
			return &CheckExecutionError{
				Type:    "url_tls_error",
				Message: fmt.Sprintf("url request to %s failed TLS validation: %v", targetURL, urlErr.Err),
			}
		}

		return &CheckExecutionError{
			Type:    "url_request_error",
			Message: fmt.Sprintf("url request to %s failed: %v", targetURL, urlErr.Err),
		}
	}

	return &CheckExecutionError{
		Type:    "url_request_error",
		Message: fmt.Sprintf("url request to %s failed: %v", targetURL, err),
	}
}

func runURLCheck(targetURL string, timeout time.Duration, followRedirects bool, expectedContains string) (URLExecutionResult, error) {
	startTime := time.Now()
	client := &http.Client{
		Timeout: timeout,
	}
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return URLExecutionResult{}, &CheckExecutionError{
			Type:    "url_request_error",
			Message: fmt.Sprintf("invalid url %q: %v", targetURL, err),
		}
	}

	response, err := client.Do(request)
	if err != nil {
		executionResult := URLExecutionResult{
			LatencyMs: time.Since(startTime).Milliseconds(),
			FinalURL:  targetURL,
		}
		return executionResult, classifyURLRequestError(targetURL, timeout, err)
	}
	defer response.Body.Close()

	finalURL := targetURL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String()
	}

	location := strings.TrimSpace(response.Header.Get("Location"))
	executionResult := URLExecutionResult{
		StatusCode: response.StatusCode,
		LatencyMs:  time.Since(startTime).Milliseconds(),
		Location:   location,
		FinalURL:   finalURL,
		Redirected: finalURL != targetURL,
	}

	if strings.TrimSpace(expectedContains) != "" {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
		if readErr != nil {
			return executionResult, &CheckExecutionError{
				Type:    "url_request_error",
				Message: fmt.Sprintf("failed to read response body from %s: %v", targetURL, readErr),
				Output:  fmt.Sprintf("%d", response.StatusCode),
			}
		}

		executionResult.Body = string(bodyBytes)
		if !strings.Contains(executionResult.Body, expectedContains) {
			return executionResult, &CheckExecutionError{
				Type:    "url_content_mismatch",
				Message: fmt.Sprintf("response body from %s did not contain %q", targetURL, expectedContains),
				Output:  fmt.Sprintf("%d", response.StatusCode),
			}
		}
	}

	return executionResult, nil
}

func checkServiceStatus(user, host string, authMethods []ssh.AuthMethod, service string, timeout time.Duration) (string, error) {
	command := fmt.Sprintf("systemctl is-active %s", service)
	result, err := runCommandWithTimeout(user, host, authMethods, command, timeout)
	if err != nil {
		if executionErr, ok := err.(*CheckExecutionError); ok && executionErr.Type == "command_error" && strings.TrimSpace(result) != "" {
			return strings.TrimSpace(result), nil
		}
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func addResolvedChecks(config Config, logger *log.Logger, resolved map[string]ResolvedCheck, order *[]string, checkGroups []string, checkNames []string) {
	for _, groupName := range checkGroups {
		checkGroup, exists := config.CheckGroups[groupName]
		if !exists {
			if logger != nil {
				logger.Printf("Check group %s not defined in config", groupName)
			}
			continue
		}

		for _, checkName := range checkGroup.Checks {
			current, exists := resolved[checkName]
			if !exists {
				*order = append(*order, checkName)
				current = ResolvedCheck{Name: checkName, Vars: make(VarMap)}
			}
			current.Vars = mergeVars(current.Vars, checkGroup.Vars)
			resolved[checkName] = current
		}
	}

	for _, checkName := range checkNames {
		if _, exists := resolved[checkName]; !exists {
			*order = append(*order, checkName)
			resolved[checkName] = ResolvedCheck{Name: checkName, Vars: make(VarMap)}
		}
	}
}

func getTopLevelString(vars VarMap, key string) (string, bool) {
	if vars == nil {
		return "", false
	}

	value, exists := vars[key]
	if !exists {
		return "", false
	}

	valueStr, ok := value.(string)
	return valueStr, ok
}

func resolveHostVars(config Config, hostConfig Host, hostGroup HostGroup) VarMap {
	var template HostTemplate
	var templateExists bool
	if hostConfig.HostTemplate != "" {
		template, templateExists = config.HostTemplates[hostConfig.HostTemplate]
	}

	combinedHostVars := mergeVars(config.HostDefaults.HostVars)
	if templateExists {
		combinedHostVars = mergeVars(combinedHostVars, template.HostVars)
	}
	combinedHostVars = mergeVars(combinedHostVars, hostGroup.HostVars, hostConfig.HostVars)
	return combinedHostVars
}

func resolveChecksForHost(config Config, hostConfig Host, hostGroup HostGroup) (map[string]ResolvedCheck, []string) {
	resolvedChecks := make(map[string]ResolvedCheck)
	var checkOrder []string

	addResolvedChecks(config, nil, resolvedChecks, &checkOrder, config.HostDefaults.CheckGroups, config.HostDefaults.HostChecks)
	if hostConfig.HostTemplate != "" {
		if template, exists := config.HostTemplates[hostConfig.HostTemplate]; exists {
			addResolvedChecks(config, nil, resolvedChecks, &checkOrder, template.CheckGroups, template.HostChecks)
		}
	}
	addResolvedChecks(config, nil, resolvedChecks, &checkOrder, hostGroup.CheckGroups, hostGroup.HostChecks)
	addResolvedChecks(config, nil, resolvedChecks, &checkOrder, hostConfig.CheckGroups, hostConfig.HostChecks)
	return resolvedChecks, checkOrder
}

func resolveIdentityName(config Config, hostConfig Host, hostGroup HostGroup) string {
	identityName := config.HostDefaults.Identity
	if hostConfig.HostTemplate != "" {
		if template, exists := config.HostTemplates[hostConfig.HostTemplate]; exists {
			if id, exists := getTopLevelString(template.HostVars, "identity"); exists {
				identityName = id
			}
		}
	}
	if id, exists := getTopLevelString(hostGroup.HostVars, "identity"); exists {
		identityName = id
	}
	if id, exists := getTopLevelString(hostConfig.HostVars, "identity"); exists {
		identityName = id
	}
	if hostConfig.Identity != "" {
		identityName = hostConfig.Identity
	}
	return identityName
}

func variableResolutionFailureResult(host string, checkName string, checkVars VarMap, err error) CheckResult {
	return CheckResult{
		Host:         host,
		Check:        checkName,
		Status:       "failed",
		Value:        "Variable Resolution Error",
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Vars:         flattenVars(checkVars),
		ErrorType:    "variable_resolution_error",
		ErrorMessage: err.Error(),
	}
}

func appendExecutionFailureResult(host string, checkName string, result string, errorType string, errorMessage string, vars map[string]string) CheckResult {
	return CheckResult{
		Host:         host,
		Check:        checkName,
		Status:       "failed",
		Value:        strings.TrimSpace(result),
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Vars:         vars,
		ErrorType:    errorType,
		ErrorMessage: errorMessage,
	}
}

func appendURLExecutionFailureResult(checkName string, result string, errorType string, errorMessage string, vars map[string]string, targetURL string, execution URLExecutionResult) CheckResult {
	checkResult := appendExecutionFailureResult("", checkName, result, errorType, errorMessage, vars)
	checkResult.URL = targetURL
	checkResult.StatusCode = execution.StatusCode
	checkResult.LatencyMs = execution.LatencyMs
	checkResult.Redirected = execution.Redirected
	checkResult.Location = execution.Location
	checkResult.FinalURL = execution.FinalURL
	return checkResult
}

func buildCheckResult(host string, checkName string, status string, value string, vars map[string]string) CheckResult {
	return CheckResult{
		Host:      host,
		Check:     checkName,
		Status:    status,
		Value:     strings.TrimSpace(value),
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Vars:      vars,
	}
}

func buildURLCheckResult(checkName string, status string, value string, vars map[string]string, targetURL string, execution URLExecutionResult) CheckResult {
	checkResult := buildCheckResult("", checkName, status, value, vars)
	checkResult.URL = targetURL
	checkResult.StatusCode = execution.StatusCode
	checkResult.LatencyMs = execution.LatencyMs
	checkResult.Redirected = execution.Redirected
	checkResult.Location = execution.Location
	checkResult.FinalURL = execution.FinalURL
	return checkResult
}

func executeHostCheck(config Config, host string, hostConfig Host, hostGroup HostGroup, checkName string, logger *log.Logger) (CheckResult, error) {
	combinedHostVars := resolveHostVars(config, hostConfig, hostGroup)
	resolvedChecks, _ := resolveChecksForHost(config, hostConfig, hostGroup)

	check, exists := config.Checks[checkName]
	if !exists {
		return CheckResult{}, fmt.Errorf("check %s not defined in config", checkName)
	}

	checkVars := mergeVars(check.Vars, resolvedChecks[checkName].Vars, combinedHostVars)
	resolvedVars, err := resolveVariables(checkVars)
	if err != nil {
		logger.Printf("Failed to resolve variables for check %s on host %s: %v", checkName, host, err)
		return variableResolutionFailureResult(host, checkName, checkVars, err), nil
	}

	identityName := resolveIdentityName(config, hostConfig, hostGroup)
	if identityName == "" {
		return CheckResult{}, fmt.Errorf("no identity configured for host %s", host)
	}

	identity, exists := config.Identities[identityName]
	if !exists {
		return CheckResult{}, fmt.Errorf("identity %s not found for host %s", identityName, host)
	}

	authMethods := getSSHAuthMethod(identity)
	timeout := parseTimeout(check.Timeout)
	resolvedFailValue := resolveFailValueValue(check.FailValue, checkVars, resolvedVars)

	var result string
	var checkFailed bool

	if check.Local {
		command := replaceVariables(check.Command, resolvedVars)
		result, err = runLocalCommand(command, timeout)
		if err != nil {
			logger.Printf("Failed to run local command %s: %v\n", command, err)
			result, errorType, errorMessage := errorDetailsFrom(err, result)
			return appendExecutionFailureResult(host, checkName, result, errorType, errorMessage, resolvedVars), nil
		}
		checkFailed = evaluateCondition(result, check.FailWhen, resolvedFailValue)
	} else if check.Command != "" {
		command := replaceVariables(check.Command, resolvedVars)
		logger.Printf("Running command on host %s: %s", host, command)
		result, err = runCommandWithTimeout(identity.User, host, authMethods, command, timeout)
		if err != nil {
			logger.Printf("Failed to run command %s on host %s: %v\n", command, host, err)
			result, errorType, errorMessage := errorDetailsFrom(err, result)
			return appendExecutionFailureResult(host, checkName, result, errorType, errorMessage, resolvedVars), nil
		}
		checkFailed = evaluateCondition(result, check.FailWhen, resolvedFailValue)
	} else if check.Service != "" {
		serviceName := replaceVariables(check.Service, resolvedVars)
		logger.Printf("Checking service %s on host %s", serviceName, host)
		result, err = checkServiceStatus(identity.User, host, authMethods, serviceName, timeout)
		if err != nil {
			logger.Printf("Failed to check service %s status on host %s: %v\n", serviceName, host, err)
			result, errorType, errorMessage := errorDetailsFrom(err, result)
			return appendExecutionFailureResult(host, checkName, result, errorType, errorMessage, resolvedVars), nil
		}
		checkFailed = evaluateCondition(result, check.FailWhen, resolvedFailValue)
	}

	status := "passed"
	if checkFailed {
		status = "failed"
	}

	return buildCheckResult(host, checkName, status, result, resolvedVars), nil
}

func executeURLCheck(checkName string, check Check, logger *log.Logger) CheckResult {
	resolvedVars, err := resolveVariables(check.Vars)
	if err != nil {
		result := variableResolutionFailureResult("", checkName, check.Vars, err)
		fmt.Printf("%s - URL Check: %s - Status: %s - Value: %s\n", result.Timestamp, checkName, result.Status, strings.TrimSpace(result.Value))
		return result
	}

	targetURL := replaceVariables(check.URL, resolvedVars)
	timeout := parseTimeout(check.Timeout)
	resolvedFailValue := resolveFailValueValue(check.FailValue, check.Vars, resolvedVars)
	logger.Printf("Running standalone URL check %s: %s", checkName, targetURL)
	fmt.Printf("Running URL check %s: %s\n", checkName, targetURL)

	executionResult, err := runURLCheck(targetURL, timeout, check.FollowRedirects, check.ExpectedContains)
	if err != nil {
		logger.Printf("Failed standalone url check %s: %v\n", checkName, err)
		result, errorType, errorMessage := errorDetailsFrom(err, fmt.Sprintf("%d", executionResult.StatusCode))
		checkResult := appendURLExecutionFailureResult(checkName, result, errorType, errorMessage, resolvedVars, targetURL, executionResult)
		fmt.Printf("%s - URL Check: %s - Status: %s - Value: %s\n", checkResult.Timestamp, checkName, checkResult.Status, strings.TrimSpace(checkResult.Value))
		return checkResult
	}

	result := fmt.Sprintf("%d", executionResult.StatusCode)
	status := "passed"
	if evaluateCondition(result, check.FailWhen, resolvedFailValue) {
		status = "failed"
	}

	checkResult := buildURLCheckResult(checkName, status, result, resolvedVars, targetURL, executionResult)
	fmt.Printf("%s - URL Check: %s - Status: %s - Value: %s\n", checkResult.Timestamp, checkName, checkResult.Status, strings.TrimSpace(checkResult.Value))
	return checkResult
}

func findHostTargetsForCheck(config Config, checkName string, hostFilter string) []HostCheckTarget {
	var targets []HostCheckTarget

	for _, groupName := range sortedHostGroupNames(config.HostGroups) {
		group := config.HostGroups[groupName]
		for _, hostName := range sortedHostNames(group.Hosts) {
			if hostFilter != "" && hostFilter != hostName {
				continue
			}

			hostConfig := group.Hosts[hostName]
			resolvedChecks, _ := resolveChecksForHost(config, hostConfig, group)
			if _, exists := resolvedChecks[checkName]; !exists {
				continue
			}

			targets = append(targets, HostCheckTarget{
				Host:       hostName,
				HostConfig: hostConfig,
				HostGroup:  group,
			})
		}
	}

	return targets
}

func sortedHostGroupNames(groups map[string]HostGroup) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedHostNames(hosts map[string]Host) []string {
	names := make([]string, 0, len(hosts))
	for name := range hosts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func runChecksOnHost(config Config, host string, hostConfig Host, hostGroup HostGroup, wg *sync.WaitGroup, logger *log.Logger, results *[]CheckResult, resultsMutex *sync.Mutex) {
	defer wg.Done()

	logger.Printf("Running checks on host: %s", host)

	resolvedChecks, checkOrder := resolveChecksForHost(config, hostConfig, hostGroup)

	logger.Printf("Resolved checks for host %s: %v", host, checkOrder)

	for _, checkName := range checkOrder {
		if _, exists := resolvedChecks[checkName]; !exists {
			continue
		}

		logger.Printf("Running check %s on host %s", checkName, host)
		checkResult, err := executeHostCheck(config, host, hostConfig, hostGroup, checkName, logger)
		if err != nil {
			logger.Printf("Failed to execute check %s on host %s: %v", checkName, host, err)
			continue
		}
		fmt.Printf("%s - Host: %s - Check: %s - Status: %s - Value: %s\n", checkResult.Timestamp, host, checkName, checkResult.Status, strings.TrimSpace(checkResult.Value))

		resultsMutex.Lock()
		*results = append(*results, checkResult)
		resultsMutex.Unlock()
	}
}

func runStandaloneURLChecks(config Config, logger *log.Logger) map[string]CheckResult {
	urlResults := make(map[string]CheckResult)
	checkNames := make([]string, 0, len(config.URLChecks))
	for checkName := range config.URLChecks {
		checkNames = append(checkNames, checkName)
	}
	sort.Strings(checkNames)

	for _, checkName := range checkNames {
		check := config.URLChecks[checkName]
		urlResults[checkName] = executeURLCheck(checkName, check, logger)
	}

	return urlResults
}

func rerunCheck(configPath string, request TargetedRunRequest) (TargetedRunResponse, error) {
	startTime := time.Now()
	previousResultData, err := loadPreviousResultFile("results.json")
	if err != nil {
		previousResultData = ResultFile{}
	}

	config, err := loadConfig(configPath)
	if err != nil {
		return TargetedRunResponse{}, err
	}
	if err := validateConfig(config); err != nil {
		return TargetedRunResponse{}, err
	}

	logFile, err := os.OpenFile("remote_check.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return TargetedRunResponse{}, fmt.Errorf("unable to open log file: %w", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)
	metadata := HistoryRunMetadata{
		RunType:    "targeted",
		TargetKind: request.Kind,
		TargetName: request.CheckName,
	}

	switch request.Kind {
	case "host_check":
		check, exists := config.Checks[request.CheckName]
		if !exists {
			return TargetedRunResponse{}, fmt.Errorf("unknown check %q", request.CheckName)
		}

		targets := findHostTargetsForCheck(config, request.CheckName, request.Host)
		if len(targets) == 0 {
			if request.Host != "" {
				return TargetedRunResponse{}, fmt.Errorf("check %q is not active on host %q", request.CheckName, request.Host)
			}
			return TargetedRunResponse{}, fmt.Errorf("check %q is not active on any host", request.CheckName)
		}

		hostResults := make([]CheckResult, 0, len(targets))
		for _, target := range targets {
			result, err := executeHostCheck(config, target.Host, target.HostConfig, target.HostGroup, request.CheckName, logger)
			if err != nil {
				return TargetedRunResponse{}, err
			}
			hostResults = append(hostResults, result)
		}

		scope := fmt.Sprintf("%d hosts", len(targets))
		if request.Host != "" {
			scope = request.Host
		}
		metadata.TargetScope = scope

		resultData := buildResultFile(config, map[string]Check{request.CheckName: check}, nil, hostResults, nil, "ok", nil)
		run, err := writeHistoryWithMetadata(resultData, previousResultData, time.Since(startTime), metadata)
		if err != nil {
			return TargetedRunResponse{}, err
		}

		return TargetedRunResponse{
			Run:         run,
			HostResults: hostResults,
		}, nil

	case "url_check":
		check, exists := config.URLChecks[request.CheckName]
		if !exists {
			return TargetedRunResponse{}, fmt.Errorf("unknown url check %q", request.CheckName)
		}

		urlResult := executeURLCheck(request.CheckName, check, logger)
		metadata.TargetScope = "controller"
		resultData := buildResultFile(config, nil, map[string]Check{request.CheckName: check}, nil, map[string]CheckResult{request.CheckName: urlResult}, "ok", nil)
		run, err := writeHistoryWithMetadata(resultData, previousResultData, time.Since(startTime), metadata)
		if err != nil {
			return TargetedRunResponse{}, err
		}

		return TargetedRunResponse{
			Run:        run,
			URLResults: []CheckResult{urlResult},
		}, nil
	default:
		return TargetedRunResponse{}, fmt.Errorf("unsupported rerun kind %q", request.Kind)
	}
}

func runChecks(configPath string) error {
	startTime := time.Now()
	previousResultData, err := loadPreviousResultFile("results.json")
	if err != nil {
		log.Printf("Warning: failed to load previous results.json: %v", err)
		previousResultData = ResultFile{}
	}

	config, err := loadConfig(configPath)
	if err != nil {
		errors := []string{err.Error()}
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = validationErrs.Messages
		}
		resultData := buildResultFile(Config{}, map[string]Check{}, map[string]Check{}, nil, nil, "config_error", errors)
		if writeErr := writeResultFile(resultData); writeErr != nil {
			return fmt.Errorf("%v; additionally failed to write results.json: %w", err, writeErr)
		}
		logHistoryWarning(writeHistory(resultData, previousResultData, time.Since(startTime)))
		return err
	}
	if err := validateConfig(config); err != nil {
		errors := []string{err.Error()}
		if validationErrs, ok := err.(ValidationErrors); ok {
			errors = validationErrs.Messages
		}
		resultData := buildResultFile(config, config.Checks, config.URLChecks, nil, nil, "config_error", errors)
		if writeErr := writeResultFile(resultData); writeErr != nil {
			return fmt.Errorf("%v; additionally failed to write results.json: %w", err, writeErr)
		}
		logHistoryWarning(writeHistory(resultData, previousResultData, time.Since(startTime)))
		return err
	}

	logFile, err := os.OpenFile("remote_check.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open log file: %w", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)

	var wg sync.WaitGroup
	var results []CheckResult
	var resultsMutex sync.Mutex

	for _, group := range config.HostGroups {
		for host, hostConfig := range group.Hosts {
			wg.Add(1)
			go runChecksOnHost(config, host, hostConfig, group, &wg, logger, &results, &resultsMutex)
		}
	}

	wg.Wait()

	urlResults := runStandaloneURLChecks(config, logger)

	resultData := buildResultFile(config, config.Checks, config.URLChecks, results, urlResults, "ok", nil)
	if err := writeResultFile(resultData); err != nil {
		return err
	}
	logHistoryWarning(writeHistory(resultData, previousResultData, time.Since(startTime)))

	return nil
}

func mapResults(results []CheckResult) map[string]map[string]CheckResult {
	resultMap := make(map[string]map[string]CheckResult)
	for _, result := range results {
		if _, exists := resultMap[result.Host]; !exists {
			resultMap[result.Host] = make(map[string]CheckResult)
		}
		resultMap[result.Host][result.Check] = result
	}
	return resultMap
}

func logHistoryWarning(err error) {
	if err != nil {
		log.Printf("Warning: failed to write history: %v", err)
	}
}
