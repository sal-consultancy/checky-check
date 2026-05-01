package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type ValidationErrors struct {
	Messages []string
}

type CheckExecutionError struct {
	Type    string
	Message string
	Output  string
}

type HistoryEvent struct {
	EventType    string
	Host         string
	CheckName    string
	Status       string
	Value        string
	ErrorType    string
	ErrorMessage string
}

func (v ValidationErrors) Error() string {
	return fmt.Sprintf("config validation failed:\n- %s", strings.Join(v.Messages, "\n- "))
}

func (e *CheckExecutionError) Error() string {
	return e.Message
}

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
	case "in":
		for _, failValStr := range failValues {
			if output == failValStr {
				return true
			}
		}
		return false
	case "not in":
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

var variablePattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_.-]*)\}`)

func replaceVariables(command string, vars map[string]string) string {
	for key, value := range vars {
		placeholder := fmt.Sprintf("${%s}", key)
		command = strings.ReplaceAll(command, placeholder, value)
	}
	return command
}

func mergeVars(varsList ...VarMap) VarMap {
	result := make(VarMap)
	for _, vars := range varsList {
		mergeVarMapInto(result, vars)
	}
	return result
}

func mergeVarMapInto(dst VarMap, src VarMap) {
	for key, value := range src {
		srcMap, srcIsMap := toVarMap(value)
		if !srcIsMap {
			dst[key] = cloneVarValue(value)
			continue
		}

		dstMap, dstIsMap := toVarMap(dst[key])
		if !dstIsMap {
			dstMap = make(VarMap)
		}
		mergeVarMapInto(dstMap, srcMap)
		dst[key] = dstMap
	}
}

func cloneVarValue(value interface{}) interface{} {
	if valueMap, ok := toVarMap(value); ok {
		cloned := make(VarMap)
		mergeVarMapInto(cloned, valueMap)
		return cloned
	}
	return value
}

func toVarMap(value interface{}) (VarMap, bool) {
	switch v := value.(type) {
	case VarMap:
		return v, true
	case map[string]interface{}:
		return VarMap(v), true
	default:
		return nil, false
	}
}

func flattenVars(vars VarMap) map[string]string {
	flattened := make(map[string]string)
	flattenVarsInto("", vars, flattened)
	return flattened
}

func getVarValue(vars VarMap, path string) (interface{}, bool) {
	if vars == nil {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := interface{}(vars)
	for _, part := range parts {
		currentMap, ok := toVarMap(current)
		if !ok {
			return nil, false
		}

		next, exists := currentMap[part]
		if !exists {
			return nil, false
		}
		current = next
	}

	return cloneVarValue(current), true
}

func flattenVarsInto(prefix string, vars VarMap, flattened map[string]string) {
	for key, value := range vars {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nestedVars, ok := toVarMap(value); ok {
			flattenVarsInto(fullKey, nestedVars, flattened)
			continue
		}

		flattened[fullKey] = fmt.Sprintf("%v", value)
	}
}

func resolveVariables(vars VarMap) (map[string]string, error) {
	raw := flattenVars(vars)
	resolved := make(map[string]string, len(raw))
	resolving := make(map[string]bool, len(raw))

	for key := range raw {
		if _, err := resolveVariableValue(key, raw, resolved, resolving); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}

func resolveVariableValue(key string, raw map[string]string, resolved map[string]string, resolving map[string]bool) (string, error) {
	if value, exists := resolved[key]; exists {
		return value, nil
	}
	if resolving[key] {
		return "", fmt.Errorf("variable cycle detected for %s", key)
	}

	resolving[key] = true
	value := raw[key]

	matches := variablePattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		reference := match[1]
		referenceValue, exists := raw[reference]
		if !exists {
			return "", fmt.Errorf("undefined variable reference %s", reference)
		}

		if _, err := resolveVariableValue(reference, raw, resolved, resolving); err != nil {
			return "", err
		}

		value = strings.ReplaceAll(value, match[0], resolved[reference])
		raw[reference] = referenceValue
	}

	resolving[key] = false
	resolved[key] = value
	return value, nil
}

func resolveFailValueValue(failValue interface{}, rawVars VarMap, resolvedVars map[string]string) interface{} {
	switch value := failValue.(type) {
	case string:
		matches := variablePattern.FindAllStringSubmatch(value, -1)
		if len(matches) == 1 && len(matches[0]) == 2 && matches[0][0] == value {
			if resolvedValue, exists := getVarValue(rawVars, matches[0][1]); exists {
				return resolvedValue
			}
		}
		return replaceVariables(value, resolvedVars)
	case []interface{}:
		resolvedValues := make([]interface{}, len(value))
		for index, item := range value {
			resolvedValues[index] = resolveFailValueValue(item, rawVars, resolvedVars)
		}
		return resolvedValues
	case []string:
		resolvedValues := make([]string, len(value))
		for index, item := range value {
			resolvedItem := resolveFailValueValue(item, rawVars, resolvedVars)
			resolvedValues[index] = fmt.Sprintf("%v", resolvedItem)
		}
		return resolvedValues
	default:
		return failValue
	}
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

func loadConfig(configPath string) (Config, error) {
	configPath = filepath.Clean(configPath)
	info, err := os.Stat(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("unable to read config path: %w", err)
	}

	if info.IsDir() {
		return loadConfigDir(configPath)
	}

	return loadConfigFile(configPath)
}

func loadConfigFile(configPath string) (Config, error) {
	configFile, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return Config{}, fmt.Errorf("unable to read config file: %w", err)
	}

	configStr, err := substituteEnvVariables(string(configFile))
	if err != nil {
		return Config{}, fmt.Errorf("unable to substitute env variables: %w", err)
	}

	var config Config
	switch strings.ToLower(filepath.Ext(configPath)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
			return Config{}, fmt.Errorf("unable to parse yaml config file: %w", err)
		}
	default:
		return Config{}, fmt.Errorf("unsupported config extension %q: only .yaml and .yml are supported", filepath.Ext(configPath))
	}

	return config, nil
}

func loadConfigDir(configDir string) (Config, error) {
	var configFiles []string

	err := filepath.WalkDir(configDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
			configFiles = append(configFiles, path)
		}
		return nil
	})
	if err != nil {
		return Config{}, fmt.Errorf("unable to walk config directory: %w", err)
	}

	sort.Strings(configFiles)
	if len(configFiles) == 0 {
		return Config{}, fmt.Errorf("no config files found in directory %s", configDir)
	}

	merged := Config{}
	var mergeErrors []string

	for _, configFile := range configFiles {
		fragment, err := loadConfigFile(configFile)
		if err != nil {
			return Config{}, err
		}

		mergeErrors = append(mergeErrors, mergeConfig(&merged, fragment, configFile)...)
	}

	if len(mergeErrors) > 0 {
		return Config{}, ValidationErrors{Messages: mergeErrors}
	}

	return merged, nil
}

func mergeConfig(dst *Config, src Config, source string) []string {
	var mergeErrors []string

	if dst.Identities == nil {
		dst.Identities = make(map[string]Identity)
	}
	if dst.HostTemplates == nil {
		dst.HostTemplates = make(map[string]HostTemplate)
	}
	if dst.CheckGroups == nil {
		dst.CheckGroups = make(map[string]CheckGroup)
	}
	if dst.Checks == nil {
		dst.Checks = make(map[string]Check)
	}
	if dst.URLChecks == nil {
		dst.URLChecks = make(map[string]Check)
	}
	if dst.HostGroups == nil {
		dst.HostGroups = make(map[string]HostGroup)
	}

	mergeErrors = append(mergeErrors, mergeIdentities(dst.Identities, src.Identities, source)...)
	mergeErrors = append(mergeErrors, mergeHostTemplates(dst.HostTemplates, src.HostTemplates, source)...)
	mergeErrors = append(mergeErrors, mergeCheckGroups(dst.CheckGroups, src.CheckGroups, source)...)
	mergeErrors = append(mergeErrors, mergeChecks(dst.Checks, src.Checks, source)...)
	mergeErrors = append(mergeErrors, mergeURLChecks(dst.URLChecks, src.URLChecks, source)...)
	mergeErrors = append(mergeErrors, mergeHostGroups(dst.HostGroups, src.HostGroups, source)...)
	mergeErrors = append(mergeErrors, mergeHostDefaults(&dst.HostDefaults, src.HostDefaults, source)...)
	mergeErrors = append(mergeErrors, mergeReport(&dst.Report, src.Report, source)...)

	return mergeErrors
}

func mergeIdentities(dst map[string]Identity, src map[string]Identity, source string) []string {
	var mergeErrors []string
	for name, value := range src {
		if _, exists := dst[name]; exists {
			mergeErrors = append(mergeErrors, fmt.Sprintf("duplicate identity %q defined in %s", name, source))
			continue
		}
		dst[name] = value
	}
	return mergeErrors
}

func mergeHostTemplates(dst map[string]HostTemplate, src map[string]HostTemplate, source string) []string {
	var mergeErrors []string
	for name, value := range src {
		if _, exists := dst[name]; exists {
			mergeErrors = append(mergeErrors, fmt.Sprintf("duplicate host_template %q defined in %s", name, source))
			continue
		}
		dst[name] = value
	}
	return mergeErrors
}

func mergeCheckGroups(dst map[string]CheckGroup, src map[string]CheckGroup, source string) []string {
	var mergeErrors []string
	for name, value := range src {
		if _, exists := dst[name]; exists {
			mergeErrors = append(mergeErrors, fmt.Sprintf("duplicate check_group %q defined in %s", name, source))
			continue
		}
		dst[name] = value
	}
	return mergeErrors
}

func mergeChecks(dst map[string]Check, src map[string]Check, source string) []string {
	var mergeErrors []string
	for name, value := range src {
		if _, exists := dst[name]; exists {
			mergeErrors = append(mergeErrors, fmt.Sprintf("duplicate check %q defined in %s", name, source))
			continue
		}
		dst[name] = value
	}
	return mergeErrors
}

func mergeURLChecks(dst map[string]Check, src map[string]Check, source string) []string {
	var mergeErrors []string
	for name, value := range src {
		if _, exists := dst[name]; exists {
			mergeErrors = append(mergeErrors, fmt.Sprintf("duplicate url_check %q defined in %s", name, source))
			continue
		}
		dst[name] = value
	}
	return mergeErrors
}

func mergeHostGroups(dst map[string]HostGroup, src map[string]HostGroup, source string) []string {
	var mergeErrors []string
	for name, value := range src {
		if _, exists := dst[name]; exists {
			mergeErrors = append(mergeErrors, fmt.Sprintf("duplicate host_group %q defined in %s", name, source))
			continue
		}
		dst[name] = value
	}
	return mergeErrors
}

func mergeHostDefaults(dst *HostDefaults, src HostDefaults, source string) []string {
	var mergeErrors []string

	if src.Identity != "" {
		if dst.Identity != "" && dst.Identity != src.Identity {
			mergeErrors = append(mergeErrors, fmt.Sprintf("host_defaults.identity conflict in %s", source))
		} else {
			dst.Identity = src.Identity
		}
	}

	dst.HostVars = mergeVars(dst.HostVars, src.HostVars)
	dst.HostChecks = appendUniqueStrings(dst.HostChecks, src.HostChecks)
	dst.CheckGroups = appendUniqueStrings(dst.CheckGroups, src.CheckGroups)

	return mergeErrors
}

func mergeReport(dst *Report, src Report, source string) []string {
	var mergeErrors []string

	mergeErrors = append(mergeErrors, mergeStringField(&dst.Title, src.Title, "report.title", source)...)
	mergeErrors = append(mergeErrors, mergeStringField(&dst.Subtitle, src.Subtitle, "report.subtitle", source)...)
	mergeErrors = append(mergeErrors, mergeStringField(&dst.Description, src.Description, "report.description", source)...)
	mergeErrors = append(mergeErrors, mergeStringField(&dst.Copyright, src.Copyright, "report.copyright", source)...)
	mergeErrors = append(mergeErrors, mergeStringField(&dst.CSS, src.CSS, "report.css", source)...)

	return mergeErrors
}

func mergeStringField(dst *string, src string, fieldName string, source string) []string {
	if src == "" {
		return nil
	}
	if *dst != "" && *dst != src {
		return []string{fmt.Sprintf("%s conflict in %s", fieldName, source)}
	}
	*dst = src
	return nil
}

func appendUniqueStrings(dst []string, src []string) []string {
	seen := make(map[string]bool, len(dst))
	for _, value := range dst {
		seen[value] = true
	}
	for _, value := range src {
		if !seen[value] {
			dst = append(dst, value)
			seen[value] = true
		}
	}
	return dst
}

func validateConfig(config Config) error {
	var validationErrors []string

	if config.HostDefaults.Identity != "" {
		if _, exists := config.Identities[config.HostDefaults.Identity]; !exists {
			validationErrors = append(validationErrors, fmt.Sprintf("host_defaults.identity references unknown identity %q", config.HostDefaults.Identity))
		}
	}

	for checkName, check := range config.Checks {
		validationErrors = append(validationErrors, validateCheckDefinition(fmt.Sprintf("check %q", checkName), check)...)
	}

	for checkName, check := range config.URLChecks {
		validationErrors = append(validationErrors, validateURLCheckDefinition(fmt.Sprintf("url_check %q", checkName), check)...)
	}

	for groupName, checkGroup := range config.CheckGroups {
		if len(checkGroup.Checks) == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("check_group %q has no checks", groupName))
		}
		for _, checkName := range checkGroup.Checks {
			if _, exists := config.Checks[checkName]; !exists {
				validationErrors = append(validationErrors, fmt.Sprintf("check_group %q references unknown check %q", groupName, checkName))
			}
		}
	}

	validationErrors = append(validationErrors, validateCheckReferences(
		"host_defaults",
		config.HostDefaults.CheckGroups,
		config.HostDefaults.HostChecks,
		config,
	)...)

	for templateName, template := range config.HostTemplates {
		validationErrors = append(validationErrors, validateCheckReferences(
			fmt.Sprintf("host_template %q", templateName),
			template.CheckGroups,
			template.HostChecks,
			config,
		)...)
	}

	for groupName, hostGroup := range config.HostGroups {
		validationErrors = append(validationErrors, validateCheckReferences(
			fmt.Sprintf("host_group %q", groupName),
			hostGroup.CheckGroups,
			hostGroup.HostChecks,
			config,
		)...)

		for hostName, host := range hostGroup.Hosts {
			hostContext := fmt.Sprintf("host %q in host_group %q", hostName, groupName)

			if host.HostTemplate != "" {
				if _, exists := config.HostTemplates[host.HostTemplate]; !exists {
					validationErrors = append(validationErrors, fmt.Sprintf("%s references unknown host_template %q", hostContext, host.HostTemplate))
				}
			}

			validationErrors = append(validationErrors, validateCheckReferences(
				hostContext,
				host.CheckGroups,
				host.HostChecks,
				config,
			)...)

			identityName := resolveIdentityName(config, host, hostGroup)
			if identityName == "" {
				validationErrors = append(validationErrors, fmt.Sprintf("%s has no resolved identity", hostContext))
			} else if _, exists := config.Identities[identityName]; !exists {
				validationErrors = append(validationErrors, fmt.Sprintf("%s resolves to unknown identity %q", hostContext, identityName))
			}

			resolvedChecks, checkOrder := resolveChecksForHost(config, host, hostGroup)
			hostVars := resolveHostVars(config, host, hostGroup)

			for _, checkName := range checkOrder {
				check, exists := config.Checks[checkName]
				if !exists {
					continue
				}

				checkVars := mergeVars(check.Vars, resolvedChecks[checkName].Vars, hostVars)
				resolvedVars, err := resolveVariables(checkVars)
				if err != nil {
					validationErrors = append(validationErrors, fmt.Sprintf("%s check %q has invalid variable resolution: %v", hostContext, checkName, err))
					continue
				}

				validationErrors = append(validationErrors, validateTemplateVariables(
					fmt.Sprintf("%s check %q command", hostContext, checkName),
					check.Command,
					resolvedVars,
				)...)
				validationErrors = append(validationErrors, validateTemplateVariables(
					fmt.Sprintf("%s check %q service", hostContext, checkName),
					check.Service,
					resolvedVars,
				)...)
				validationErrors = append(validationErrors, validateTemplateVariables(
					fmt.Sprintf("%s check %q url", hostContext, checkName),
					check.URL,
					resolvedVars,
				)...)
			}
		}
	}

	for checkName, check := range config.URLChecks {
		resolvedVars, err := resolveVariables(check.Vars)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("url_check %q has invalid variable resolution: %v", checkName, err))
			continue
		}

		validationErrors = append(validationErrors, validateTemplateVariables(
			fmt.Sprintf("url_check %q url", checkName),
			check.URL,
			resolvedVars,
		)...)
	}

	if len(validationErrors) > 0 {
		return ValidationErrors{Messages: validationErrors}
	}

	return nil
}

func validateCheckDefinition(context string, check Check) []string {
	var validationErrors []string
	definedTargets := 0
	if strings.TrimSpace(check.Command) != "" {
		definedTargets++
	}
	if strings.TrimSpace(check.Service) != "" {
		definedTargets++
	}
	if strings.TrimSpace(check.URL) != "" {
		definedTargets++
	}

	if definedTargets == 0 {
		return []string{fmt.Sprintf("%s must define command, service, or url", context)}
	}
	if definedTargets > 1 {
		return []string{fmt.Sprintf("%s cannot define more than one of command, service, or url", context)}
	}
	if strings.TrimSpace(check.URL) == "" && check.FollowRedirects {
		validationErrors = append(validationErrors, fmt.Sprintf("%s cannot use follow_redirects without url", context))
	}
	if strings.TrimSpace(check.URL) == "" && strings.TrimSpace(check.ExpectedContains) != "" {
		validationErrors = append(validationErrors, fmt.Sprintf("%s cannot use expected_contains without url", context))
	}
	return validationErrors
}

func validateURLCheckDefinition(context string, check Check) []string {
	validationErrors := validateCheckDefinition(context, check)
	if len(validationErrors) > 0 {
		return validationErrors
	}
	if strings.TrimSpace(check.URL) == "" {
		return []string{fmt.Sprintf("%s must define url", context)}
	}
	if check.Local {
		return []string{fmt.Sprintf("%s cannot use local because url checks already run from the controller", context)}
	}
	return nil
}

func validateCheckReferences(context string, checkGroups []string, checks []string, config Config) []string {
	var validationErrors []string

	for _, groupName := range checkGroups {
		if _, exists := config.CheckGroups[groupName]; !exists {
			validationErrors = append(validationErrors, fmt.Sprintf("%s references unknown check_group %q", context, groupName))
		}
	}

	for _, checkName := range checks {
		if _, exists := config.Checks[checkName]; !exists {
			validationErrors = append(validationErrors, fmt.Sprintf("%s references unknown check %q", context, checkName))
		}
	}

	return validationErrors
}

func validateTemplateVariables(context string, value string, vars map[string]string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	var validationErrors []string
	matches := variablePattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}

		reference := match[1]
		if _, exists := vars[reference]; !exists {
			validationErrors = append(validationErrors, fmt.Sprintf("%s references undefined variable %q", context, reference))
		}
	}

	return validationErrors
}

func classifyDialError(err error) *CheckExecutionError {
	message := err.Error()
	errorType := "ssh_connection_error"
	if strings.Contains(message, "unable to authenticate") || strings.Contains(message, "no supported methods remain") {
		errorType = "ssh_auth_error"
	}

	return &CheckExecutionError{
		Type:    errorType,
		Message: message,
	}
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
		return "", classifyDialError(fmt.Errorf("failed to dial: %v", err))
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", &CheckExecutionError{
			Type:    "ssh_session_error",
			Message: fmt.Sprintf("failed to create session: %v", err),
		}
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
			return res.output, &CheckExecutionError{
				Type:    "command_error",
				Message: fmt.Sprintf("failed to run command: %v", res.err),
				Output:  strings.TrimSpace(res.output),
			}
		}
		return res.output, nil
	case <-ctx.Done():
		log.Printf("Command timed out after %v: %s on host %s", timeout, command, host)
		return "", &CheckExecutionError{
			Type:    "timeout",
			Message: fmt.Sprintf("command timed out after %v", timeout),
		}
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

func loadPreviousResultFile(path string) (ResultFile, error) {
	resultFile, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return ResultFile{}, nil
		}
		return ResultFile{}, err
	}

	var resultData ResultFile
	if err := json.Unmarshal(resultFile, &resultData); err != nil {
		return ResultFile{}, err
	}

	return resultData, nil
}

func writeHistory(resultData ResultFile, previousResultData ResultFile, duration time.Duration) error {
	_, err := writeHistoryWithMetadata(resultData, previousResultData, duration, HistoryRunMetadata{RunType: "full"})
	return err
}

func writeHistoryWithMetadata(resultData ResultFile, previousResultData ResultFile, duration time.Duration, metadata HistoryRunMetadata) (HistoryRun, error) {
	if metadata.RunType == "" {
		metadata.RunType = "full"
	}

	if err := os.MkdirAll("history", 0755); err != nil {
		return HistoryRun{}, fmt.Errorf("unable to create history directory: %w", err)
	}

	db, err := sql.Open("sqlite", filepath.Clean("history/checkycheck_history.db"))
	if err != nil {
		return HistoryRun{}, fmt.Errorf("unable to open history database: %w", err)
	}
	defer db.Close()

	if err := initHistorySchema(db); err != nil {
		return HistoryRun{}, err
	}

	runSummaryJSON, err := buildRunErrorSummaryJSON(resultData)
	if err != nil {
		return HistoryRun{}, fmt.Errorf("unable to build run error summary: %w", err)
	}

	hostCount, checkCount, passedCount, failedCount := summarizeResultCounts(resultData)
	events := buildHistoryEvents(resultData, previousResultData, metadata)
	sparklineMetrics := buildSparklineMetrics(resultData, metadata)

	tx, err := db.Begin()
	if err != nil {
		return HistoryRun{}, fmt.Errorf("unable to start history transaction: %w", err)
	}
	defer tx.Rollback()

	insertRunResult, err := tx.Exec(`
		INSERT INTO runs (
			generated_at,
			status,
			run_type,
			target_kind,
			target_name,
			target_scope,
			host_count,
			check_count,
			passed_count,
			failed_count,
			duration_ms,
			error_summary
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		resultData.GeneratedAt,
		resultData.Status,
		metadata.RunType,
		metadata.TargetKind,
		metadata.TargetName,
		metadata.TargetScope,
		hostCount,
		checkCount,
		passedCount,
		failedCount,
		duration.Milliseconds(),
		runSummaryJSON,
	)
	if err != nil {
		return HistoryRun{}, fmt.Errorf("unable to insert run history: %w", err)
	}

	runID, err := insertRunResult.LastInsertId()
	if err != nil {
		return HistoryRun{}, fmt.Errorf("unable to determine run history id: %w", err)
	}

	for _, event := range events {
		if _, err := tx.Exec(`
			INSERT INTO events (
				run_id,
				event_time,
				event_type,
				host,
				check_name,
				status,
				value,
				error_type,
				error_message
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			runID,
			resultData.GeneratedAt,
			event.EventType,
			event.Host,
			event.CheckName,
			event.Status,
			event.Value,
			event.ErrorType,
			event.ErrorMessage,
		); err != nil {
			return HistoryRun{}, fmt.Errorf("unable to insert history event: %w", err)
		}
	}

	for _, metric := range sparklineMetrics {
		if _, err := tx.Exec(`
			INSERT INTO run_metrics (
				run_id,
				generated_at,
				host,
				check_name,
				numeric_value,
				status
			) VALUES (?, ?, ?, ?, ?, ?)
		`,
			runID,
			resultData.GeneratedAt,
			metric.Host,
			metric.CheckName,
			metric.Value,
			metric.Status,
		); err != nil {
			return HistoryRun{}, fmt.Errorf("unable to insert history metric: %w", err)
		}
	}

	if err := pruneHistory(tx); err != nil {
		return HistoryRun{}, err
	}

	if err := tx.Commit(); err != nil {
		return HistoryRun{}, fmt.Errorf("unable to commit history transaction: %w", err)
	}

	return HistoryRun{
		ID:           runID,
		GeneratedAt:  resultData.GeneratedAt,
		Status:       resultData.Status,
		RunType:      metadata.RunType,
		TargetKind:   metadata.TargetKind,
		TargetName:   metadata.TargetName,
		TargetScope:  metadata.TargetScope,
		HostCount:    hostCount,
		CheckCount:   checkCount,
		PassedCount:  passedCount,
		FailedCount:  failedCount,
		DurationMs:   duration.Milliseconds(),
		ErrorSummary: buildRunErrorSummary(resultData),
	}, nil
}

func initHistorySchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			generated_at TEXT NOT NULL,
			status TEXT NOT NULL,
			run_type TEXT NOT NULL DEFAULT 'full',
			target_kind TEXT NOT NULL DEFAULT '',
			target_name TEXT NOT NULL DEFAULT '',
			target_scope TEXT NOT NULL DEFAULT '',
			host_count INTEGER NOT NULL,
			check_count INTEGER NOT NULL,
			passed_count INTEGER NOT NULL,
			failed_count INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			error_summary TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id INTEGER NOT NULL,
			event_time TEXT NOT NULL,
			event_type TEXT NOT NULL,
			host TEXT NOT NULL DEFAULT '',
			check_name TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			value TEXT NOT NULL DEFAULT '',
			error_type TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runs_generated_at ON runs(generated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_events_event_time ON events(event_time)`,
		`CREATE INDEX IF NOT EXISTS idx_events_host_check ON events(host, check_name)`,
		`CREATE TABLE IF NOT EXISTS run_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id INTEGER NOT NULL,
			generated_at TEXT NOT NULL,
			host TEXT NOT NULL,
			check_name TEXT NOT NULL,
			numeric_value REAL NOT NULL,
			status TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_run_metrics_host_check_time ON run_metrics(host, check_name, generated_at)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("unable to initialize history schema: %w", err)
		}
	}

	if err := ensureHistoryColumn(db, "runs", "run_type", "ALTER TABLE runs ADD COLUMN run_type TEXT NOT NULL DEFAULT 'full'"); err != nil {
		return err
	}
	if err := ensureHistoryColumn(db, "runs", "target_kind", "ALTER TABLE runs ADD COLUMN target_kind TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureHistoryColumn(db, "runs", "target_name", "ALTER TABLE runs ADD COLUMN target_name TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureHistoryColumn(db, "runs", "target_scope", "ALTER TABLE runs ADD COLUMN target_scope TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	return nil
}

func ensureHistoryColumn(db *sql.DB, table string, column string, alterStatement string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("unable to inspect history schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("unable to read history schema info: %w", err)
		}
		if name == column {
			return nil
		}
	}

	if _, err := db.Exec(alterStatement); err != nil {
		return fmt.Errorf("unable to alter history schema: %w", err)
	}

	return nil
}

func buildRunErrorSummary(resultData ResultFile) map[string]int {
	summary := make(map[string]int)

	if resultData.Status == "config_error" {
		summary["config_error"] = len(resultData.Errors)
	} else {
		for _, hostResults := range resultData.Results {
			for _, result := range hostResults {
				if result.ErrorType == "" {
					continue
				}
				summary[result.ErrorType]++
			}
		}
		for _, result := range resultData.URLResults {
			if result.ErrorType == "" {
				continue
			}
			summary[result.ErrorType]++
		}
	}

	if len(summary) == 0 {
		summary["none"] = 0
	}

	return summary
}

func openHistoryDB() (*sql.DB, error) {
	dbPath := filepath.Clean("history/checkycheck_history.db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := initHistorySchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func buildRunErrorSummaryJSON(resultData ResultFile) (string, error) {
	summary := buildRunErrorSummary(resultData)
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}

	return string(summaryJSON), nil
}

func summarizeResultCounts(resultData ResultFile) (int, int, int, int) {
	hostCount := len(resultData.Results)
	checkCount := len(resultData.Checks) + len(resultData.URLChecks)
	passedCount := 0
	failedCount := 0

	for _, hostResults := range resultData.Results {
		for _, result := range hostResults {
			if result.Status == "failed" {
				failedCount++
			} else {
				passedCount++
			}
		}
	}

	for _, result := range resultData.URLResults {
		if result.Status == "failed" {
			failedCount++
		} else {
			passedCount++
		}
	}

	return hostCount, checkCount, passedCount, failedCount
}

func buildSparklineMetrics(resultData ResultFile, metadata HistoryRunMetadata) []HistorySparklineMetric {
	if metadata.RunType != "full" {
		return nil
	}

	var metrics []HistorySparklineMetric
	for host, hostResults := range resultData.Results {
		for checkName, result := range hostResults {
			check, exists := resultData.Checks[checkName]
			if !exists || !check.Sparkline.Enabled {
				continue
			}

			numericValue, err := strconv.ParseFloat(strings.TrimSpace(result.Value), 64)
			if err != nil {
				continue
			}

			metrics = append(metrics, HistorySparklineMetric{
				GeneratedAt: resultData.GeneratedAt,
				Host:        host,
				CheckName:   checkName,
				Value:       numericValue,
				Status:      result.Status,
			})
		}
	}

	for checkName, result := range resultData.URLResults {
		check, exists := resultData.URLChecks[checkName]
		if !exists || !check.Sparkline.Enabled {
			continue
		}

		numericValue, err := strconv.ParseFloat(strings.TrimSpace(result.Value), 64)
		if err != nil {
			continue
		}

		metrics = append(metrics, HistorySparklineMetric{
			GeneratedAt: resultData.GeneratedAt,
			Host:        "url_checks",
			CheckName:   checkName,
			Value:       numericValue,
			Status:      result.Status,
		})
	}

	return metrics
}

func buildHistoryEvents(current ResultFile, previous ResultFile, metadata HistoryRunMetadata) []HistoryEvent {
	if current.Status == "config_error" {
		events := make([]HistoryEvent, 0, len(current.Errors))
		for _, message := range current.Errors {
			events = append(events, HistoryEvent{
				EventType:    "config_error",
				Status:       "failed",
				ErrorType:    "config_error",
				ErrorMessage: message,
			})
		}
		return events
	}

	if metadata.RunType == "targeted" {
		var events []HistoryEvent
		for host, hostResults := range current.Results {
			for checkName, result := range hostResults {
				events = append(events, HistoryEvent{
					EventType:    "targeted_result",
					Host:         host,
					CheckName:    checkName,
					Status:       result.Status,
					Value:        result.Value,
					ErrorType:    result.ErrorType,
					ErrorMessage: result.ErrorMessage,
				})
			}
		}
		for checkName, result := range current.URLResults {
			events = append(events, HistoryEvent{
				EventType:    "targeted_result",
				Host:         "url_checks",
				CheckName:    checkName,
				Status:       result.Status,
				Value:        result.Value,
				ErrorType:    result.ErrorType,
				ErrorMessage: result.ErrorMessage,
			})
		}
		return events
	}

	var events []HistoryEvent
	for host, hostResults := range current.Results {
		events = appendHistoryEvents(events, host, hostResults, previous.Results[host])
	}
	events = appendHistoryEvents(events, "url_checks", current.URLResults, previous.URLResults)

	return events
}

func appendHistoryEvents(events []HistoryEvent, host string, currentResults map[string]CheckResult, previousResults map[string]CheckResult) []HistoryEvent {
	for checkName, currentResult := range currentResults {
		previousResult, hadPrevious := previousResults[checkName]

		switch {
		case currentResult.Status == "failed" && (!hadPrevious || previousResult.Status != "failed"):
			events = append(events, HistoryEvent{
				EventType:    "failed",
				Host:         host,
				CheckName:    checkName,
				Status:       currentResult.Status,
				Value:        currentResult.Value,
				ErrorType:    currentResult.ErrorType,
				ErrorMessage: currentResult.ErrorMessage,
			})
		case currentResult.Status == "passed" && hadPrevious && previousResult.Status == "failed":
			events = append(events, HistoryEvent{
				EventType:    "recovered",
				Host:         host,
				CheckName:    checkName,
				Status:       currentResult.Status,
				Value:        currentResult.Value,
				ErrorType:    currentResult.ErrorType,
				ErrorMessage: currentResult.ErrorMessage,
			})
		case currentResult.Status == "failed" && hadPrevious && previousResult.Status == "failed" && previousResult.ErrorType != currentResult.ErrorType:
			events = append(events, HistoryEvent{
				EventType:    "issue_changed",
				Host:         host,
				CheckName:    checkName,
				Status:       currentResult.Status,
				Value:        currentResult.Value,
				ErrorType:    currentResult.ErrorType,
				ErrorMessage: currentResult.ErrorMessage,
			})
		case currentResult.Status == "failed" && hadPrevious && previousResult.Status == "failed" && previousResult.Value != currentResult.Value:
			events = append(events, HistoryEvent{
				EventType:    "value_changed",
				Host:         host,
				CheckName:    checkName,
				Status:       currentResult.Status,
				Value:        currentResult.Value,
				ErrorType:    currentResult.ErrorType,
				ErrorMessage: currentResult.ErrorMessage,
			})
		}
	}

	return events
}

func pruneHistory(tx *sql.Tx) error {
	statements := []string{
		`DELETE FROM events WHERE event_time < datetime('now', '-30 day')`,
		`DELETE FROM run_metrics WHERE generated_at < datetime('now', '-90 day')`,
		`DELETE FROM runs WHERE generated_at < datetime('now', '-90 day')`,
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("unable to prune history: %w", err)
		}
	}

	return nil
}

func readRecentRuns(limit int) ([]HistoryRun, error) {
	db, err := openHistoryDB()
	if err != nil {
		return nil, fmt.Errorf("unable to open history database: %w", err)
	}
	if db == nil {
		return []HistoryRun{}, nil
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, generated_at, status, run_type, target_kind, target_name, target_scope, host_count, check_count, passed_count, failed_count, duration_ms, error_summary
		FROM runs
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("unable to query recent runs: %w", err)
	}
	defer rows.Close()

	var runs []HistoryRun
	for rows.Next() {
		var run HistoryRun
		var errorSummaryJSON string
		if err := rows.Scan(
			&run.ID,
			&run.GeneratedAt,
			&run.Status,
			&run.RunType,
			&run.TargetKind,
			&run.TargetName,
			&run.TargetScope,
			&run.HostCount,
			&run.CheckCount,
			&run.PassedCount,
			&run.FailedCount,
			&run.DurationMs,
			&errorSummaryJSON,
		); err != nil {
			return nil, fmt.Errorf("unable to scan recent run: %w", err)
		}

		run.ErrorSummary = make(map[string]int)
		if err := json.Unmarshal([]byte(errorSummaryJSON), &run.ErrorSummary); err != nil {
			return nil, fmt.Errorf("unable to parse run error summary: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating runs: %w", err)
	}

	return runs, nil
}

func readRecentEvents(limit int, runID *int64) ([]HistoryEventRecord, error) {
	db, err := openHistoryDB()
	if err != nil {
		return nil, fmt.Errorf("unable to open history database: %w", err)
	}
	if db == nil {
		return []HistoryEventRecord{}, nil
	}
	defer db.Close()

	query := `
		SELECT id, run_id, event_time, event_type, host, check_name, status, value, error_type, error_message
		FROM events
	`
	var rows *sql.Rows
	if runID != nil {
		query += `
		WHERE run_id = ?
		ORDER BY id DESC
		LIMIT ?
	`
		rows, err = db.Query(query, *runID, limit)
	} else {
		query += `
		ORDER BY id DESC
		LIMIT ?
	`
		rows, err = db.Query(query, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to query recent events: %w", err)
	}
	defer rows.Close()

	var events []HistoryEventRecord
	for rows.Next() {
		var event HistoryEventRecord
		if err := rows.Scan(
			&event.ID,
			&event.RunID,
			&event.EventTime,
			&event.EventType,
			&event.Host,
			&event.CheckName,
			&event.Status,
			&event.Value,
			&event.ErrorType,
			&event.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("unable to scan recent event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

func readHostSparklineMetrics(limit int) (map[string]map[string][]HistorySparklineMetric, error) {
	db, err := openHistoryDB()
	if err != nil {
		return nil, fmt.Errorf("unable to open history database: %w", err)
	}
	if db == nil {
		return map[string]map[string][]HistorySparklineMetric{}, nil
	}
	defer db.Close()

	rows, err := db.Query(`
		WITH ranked_metrics AS (
			SELECT
				run_id,
				generated_at,
				host,
				check_name,
				numeric_value,
				status,
				ROW_NUMBER() OVER (
					PARTITION BY host, check_name
					ORDER BY generated_at DESC, run_id DESC
				) AS rn
			FROM run_metrics
		)
		SELECT run_id, generated_at, host, check_name, numeric_value, status
		FROM ranked_metrics
		WHERE rn <= ?
		ORDER BY host, check_name, generated_at ASC, run_id ASC
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("unable to query sparkline metrics: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string][]HistorySparklineMetric)
	for rows.Next() {
		var metric HistorySparklineMetric
		if err := rows.Scan(
			&metric.RunID,
			&metric.GeneratedAt,
			&metric.Host,
			&metric.CheckName,
			&metric.Value,
			&metric.Status,
		); err != nil {
			return nil, fmt.Errorf("unable to scan sparkline metric: %w", err)
		}

		if _, exists := result[metric.Host]; !exists {
			result[metric.Host] = make(map[string][]HistorySparklineMetric)
		}
		result[metric.Host][metric.CheckName] = append(result[metric.Host][metric.CheckName], metric)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sparkline metrics: %w", err)
	}

	return result, nil
}

func readCheckHistoryDetail(host string, checkName string, limit int) (CheckHistoryDetail, error) {
	db, err := openHistoryDB()
	if err != nil {
		return CheckHistoryDetail{}, fmt.Errorf("unable to open history database: %w", err)
	}
	if db == nil {
		return CheckHistoryDetail{
			Host:      host,
			CheckName: checkName,
			Metrics:   []HistorySparklineMetric{},
			Events:    []HistoryEventRecord{},
		}, nil
	}
	defer db.Close()

	metricRows, err := db.Query(`
		SELECT run_id, generated_at, host, check_name, numeric_value, status
		FROM run_metrics
		WHERE host = ? AND check_name = ?
		ORDER BY generated_at DESC, run_id DESC
		LIMIT ?
	`, host, checkName, limit)
	if err != nil {
		return CheckHistoryDetail{}, fmt.Errorf("unable to query check history metrics: %w", err)
	}
	defer metricRows.Close()

	var metrics []HistorySparklineMetric
	for metricRows.Next() {
		var metric HistorySparklineMetric
		if err := metricRows.Scan(
			&metric.RunID,
			&metric.GeneratedAt,
			&metric.Host,
			&metric.CheckName,
			&metric.Value,
			&metric.Status,
		); err != nil {
			return CheckHistoryDetail{}, fmt.Errorf("unable to scan check history metric: %w", err)
		}
		metrics = append(metrics, metric)
	}
	if err := metricRows.Err(); err != nil {
		return CheckHistoryDetail{}, fmt.Errorf("error iterating check history metrics: %w", err)
	}

	for left, right := 0, len(metrics)-1; left < right; left, right = left+1, right-1 {
		metrics[left], metrics[right] = metrics[right], metrics[left]
	}

	eventRows, err := db.Query(`
		SELECT id, run_id, event_time, event_type, host, check_name, status, value, error_type, error_message
		FROM events
		WHERE host = ? AND check_name = ?
		ORDER BY id DESC
		LIMIT ?
	`, host, checkName, limit)
	if err != nil {
		return CheckHistoryDetail{}, fmt.Errorf("unable to query check history events: %w", err)
	}
	defer eventRows.Close()

	var events []HistoryEventRecord
	for eventRows.Next() {
		var event HistoryEventRecord
		if err := eventRows.Scan(
			&event.ID,
			&event.RunID,
			&event.EventTime,
			&event.EventType,
			&event.Host,
			&event.CheckName,
			&event.Status,
			&event.Value,
			&event.ErrorType,
			&event.ErrorMessage,
		); err != nil {
			return CheckHistoryDetail{}, fmt.Errorf("unable to scan check history event: %w", err)
		}
		events = append(events, event)
	}
	if err := eventRows.Err(); err != nil {
		return CheckHistoryDetail{}, fmt.Errorf("error iterating check history events: %w", err)
	}

	return CheckHistoryDetail{
		Host:      host,
		CheckName: checkName,
		Metrics:   metrics,
		Events:    events,
	}, nil
}

func buildPreflightReport(configPath string) PreflightReport {
	report := PreflightReport{
		OverallStatus: "ok",
		ConfigPath:    filepath.Clean(configPath),
	}

	if workingDir, err := os.Getwd(); err == nil {
		report.WorkingDir = workingDir
		report.Checks = append(report.Checks, PreflightCheck{
			Name:    "Working directory",
			Status:  "ok",
			Message: workingDir,
		})
	} else {
		report.WorkingDir = "unknown"
		report.Checks = append(report.Checks, PreflightCheck{
			Name:    "Working directory",
			Status:  "error",
			Message: fmt.Sprintf("Could not determine current working directory: %v", err),
		})
	}

	addCheck := func(name string, err error, successMessage string) {
		check := PreflightCheck{Name: name}
		if err != nil {
			check.Status = "error"
			check.Message = err.Error()
			report.OverallStatus = "error"
		} else {
			check.Status = "ok"
			check.Message = successMessage
		}
		report.Checks = append(report.Checks, check)
	}

	if configInfo, err := os.Stat(report.ConfigPath); err != nil {
		addCheck("Config path", err, "")
	} else {
		configType := "file"
		if configInfo.IsDir() {
			configType = "directory"
		}
		addCheck("Config path", nil, fmt.Sprintf("Using %s %s", configType, report.ConfigPath))
	}

	envChecks := collectConfigEnvChecks(report.ConfigPath)
	report.Checks = append(report.Checks, envChecks...)
	for _, check := range envChecks {
		if check.Status == "error" {
			report.OverallStatus = "error"
		} else if check.Status == "warning" && report.OverallStatus == "ok" {
			report.OverallStatus = "warning"
		}
	}

	config, configErr := loadConfig(report.ConfigPath)
	addCheck("Config load", configErr, "Configuration loaded successfully.")

	if configErr == nil {
		validationErr := validateConfig(config)
		if validationErr != nil {
			addCheck("Config validation", validationErr, "")
		} else {
			addCheck("Config validation", nil, "Config validation passed.")
		}

		identityChecks := collectIdentityChecks(config)
		report.Checks = append(report.Checks, identityChecks...)
		for _, check := range identityChecks {
			if check.Status == "error" {
				report.OverallStatus = "error"
			} else if check.Status == "warning" && report.OverallStatus == "ok" {
				report.OverallStatus = "warning"
			}
		}
	}

	if configInfoErr := ensureWritableFile("results.json"); configInfoErr != nil {
		addCheck("Results file", configInfoErr, "")
	} else {
		addCheck("Results file", nil, "results.json is writable.")
	}

	if logErr := ensureWritableFile("remote_check.log"); logErr != nil {
		addCheck("Remote check log", logErr, "")
	} else {
		addCheck("Remote check log", nil, "remote_check.log is writable.")
	}

	if historyErr := ensureHistoryWritable(); historyErr != nil {
		addCheck("History storage", historyErr, "")
	} else {
		addCheck("History storage", nil, "History directory and database path are writable.")
	}

	if executablePath, err := os.Executable(); err != nil {
		addCheck("Server binary", err, "")
	} else if _, err := os.Stat(executablePath); err != nil {
		addCheck("Server binary", err, "")
	} else {
		addCheck("Server binary", nil, fmt.Sprintf("Binary available at %s.", executablePath))
	}

	return report
}

func collectConfigEnvChecks(configPath string) []PreflightCheck {
	var files []string

	info, err := os.Stat(configPath)
	if err != nil {
		return nil
	}

	if info.IsDir() {
		_ = filepath.WalkDir(configPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".yaml", ".yml":
				files = append(files, path)
			}
			return nil
		})
	} else {
		files = append(files, configPath)
	}

	re := regexp.MustCompile(`\$\{env\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	seen := make(map[string]bool)
	var checks []PreflightCheck

	for _, path := range files {
		content, readErr := os.ReadFile(filepath.Clean(path))
		if readErr != nil {
			continue
		}

		matches := re.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) != 2 {
				continue
			}
			envName := match[1]
			if seen[envName] {
				continue
			}
			seen[envName] = true

			value := os.Getenv(envName)
			if value == "" {
				checks = append(checks, PreflightCheck{
					Name:    fmt.Sprintf("Env %s", envName),
					Status:  "error",
					Message: fmt.Sprintf("Environment variable %s is not set.", envName),
				})
				continue
			}

			checks = append(checks, PreflightCheck{
				Name:    fmt.Sprintf("Env %s", envName),
				Status:  "ok",
				Message: fmt.Sprintf("Environment variable %s is available.", envName),
			})
		}
	}

	sort.Slice(checks, func(i, j int) bool {
		return checks[i].Name < checks[j].Name
	})

	return checks
}

func collectIdentityChecks(config Config) []PreflightCheck {
	var identityNames []string
	for name := range config.Identities {
		identityNames = append(identityNames, name)
	}
	sort.Strings(identityNames)

	var checks []PreflightCheck
	for _, name := range identityNames {
		identity := config.Identities[name]
		if strings.TrimSpace(identity.Key) != "" {
			keyPath := filepath.Clean(identity.Key)
			keyContents, err := os.ReadFile(keyPath)
			if err != nil {
				checks = append(checks, PreflightCheck{
					Name:    fmt.Sprintf("Identity %s", name),
					Status:  "error",
					Message: fmt.Sprintf("Could not read private key %s: %v", keyPath, err),
				})
				continue
			}

			if strings.TrimSpace(identity.Passphrase) != "" {
				if _, err := ssh.ParsePrivateKeyWithPassphrase(keyContents, []byte(identity.Passphrase)); err != nil {
					checks = append(checks, PreflightCheck{
						Name:    fmt.Sprintf("Identity %s", name),
						Status:  "error",
						Message: fmt.Sprintf("Private key %s could not be unlocked with the configured passphrase: %v", keyPath, err),
					})
					continue
				}

				checks = append(checks, PreflightCheck{
					Name:    fmt.Sprintf("Identity %s", name),
					Status:  "ok",
					Message: fmt.Sprintf("Private key %s is readable and unlocks for user %s.", keyPath, identity.User),
				})
				continue
			}

			if _, err := ssh.ParsePrivateKey(keyContents); err != nil {
				checks = append(checks, PreflightCheck{
					Name:    fmt.Sprintf("Identity %s", name),
					Status:  "error",
					Message: fmt.Sprintf("Private key %s could not be parsed without a passphrase: %v", keyPath, err),
				})
				continue
			}

			checks = append(checks, PreflightCheck{
				Name:    fmt.Sprintf("Identity %s", name),
				Status:  "ok",
				Message: fmt.Sprintf("Private key %s is readable for user %s.", keyPath, identity.User),
			})
			continue
		}

		if strings.TrimSpace(identity.Password) != "" {
			checks = append(checks, PreflightCheck{
				Name:    fmt.Sprintf("Identity %s", name),
				Status:  "ok",
				Message: fmt.Sprintf("Password authentication is configured for user %s.", identity.User),
			})
			continue
		}

		checks = append(checks, PreflightCheck{
			Name:    fmt.Sprintf("Identity %s", name),
			Status:  "warning",
			Message: "No private key or password is configured.",
		})
	}

	return checks
}

func ensureWritableFile(path string) error {
	file, err := os.OpenFile(filepath.Clean(path), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

func ensureHistoryWritable() error {
	if err := os.MkdirAll("history", 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Clean("history/checkycheck_history.db"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}
