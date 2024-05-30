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
    User   string           `json:"user"`
    Key    string           `json:"key"`
    Checks map[string]Check `json:"checks"`
    Hosts  map[string]Host  `json:"hosts"`
}

type Check struct {
    Command   string `json:"command,omitempty"`
    Service   string `json:"service,omitempty"`
    FailWhen  string `json:"fail_when"`
    FailValue string `json:"fail_value"`
}

type Host struct {
    Vars   map[string]string `json:"vars,omitempty"`
    Checks []string          `json:"checks"`
}

// Functie om de privésleutel in te laden
func publicKeyFile(file string) ssh.AuthMethod {
    buffer, err := ioutil.ReadFile(file)
    if err != nil {
        log.Fatalf("unable to read private key: %v", err)
    }

    key, err := ssh.ParsePrivateKey(buffer)
    if err != nil {
        log.Fatalf("unable to parse private key: %v", err)
    }
    return ssh.PublicKeys(key)
}

// Functie om een commando uit te voeren via SSH
func runCommand(user, host, keyPath, command string) (string, error) {
    sshConfig := &ssh.ClientConfig{
        User: user,
        Auth: []ssh.AuthMethod{
            publicKeyFile(keyPath),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    }

    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), sshConfig)
    if err != nil {
        return "", err
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        return "", err
    }
    defer session.Close()

    output, err := session.CombinedOutput(command)
    if err != nil {
        return "", err
    }

    return string(output), nil
}

// Functie om een service status te controleren via SSH
func checkServiceStatus(user, host, keyPath, service string) (string, error) {
    command := fmt.Sprintf("systemctl is-active %s", service)
    result, err := runCommand(user, host, keyPath, command)
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

func runChecksOnHost(config Config, host string, hostConfig Host, wg *sync.WaitGroup) {
    defer wg.Done()
    for _, checkName := range hostConfig.Checks {
        check, exists := config.Checks[checkName]
        if !exists {
            log.Printf("Check %s not defined in config\n", checkName)
            continue
        }

        var result string
        var err error
        var checkFailed bool

        if check.Command != "" {
            command := replaceVariables(check.Command, hostConfig.Vars)
            result, err = runCommand(config.User, host, filepath.Clean(os.ExpandEnv(config.Key)), command)
            if err != nil {
                log.Printf("Failed to run command %s on host %s: %v\n", command, host, err)
                continue
            }
            checkFailed = evaluateCondition(result, check.FailWhen, check.FailValue)
        } else if check.Service != "" {
            result, err = checkServiceStatus(config.User, host, filepath.Clean(os.ExpandEnv(config.Key)), check.Service)
            if err != nil {
                log.Printf("Failed to check service %s status on host %s: %v\n", check.Service, host, err)
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
    }
}

func main() {
    // Laad de configuratie
    configFile, err := ioutil.ReadFile("config.json")
    if err != nil {
        log.Fatalf("unable to read config file: %v", err)
    }

    var config Config
    if err := json.Unmarshal(configFile, &config); err != nil {
        log.Fatalf("unable to parse config file: %v", err)
    }

    // Gebruik een WaitGroup om te wachten tot alle goroutines klaar zijn
    var wg sync.WaitGroup

    // Verwerk elke host en voer de opgegeven checks parallel uit
    for host, hostConfig := range config.Hosts {
        wg.Add(1)
        go runChecksOnHost(config, host, hostConfig, &wg)
    }

    // Wacht tot alle goroutines klaar zijn
    wg.Wait()
}
