# CheckyCheck

Monitor hundreds of virtual machines by executing a single binary.

Like Ansible, it uses the SSH protocol to perform remote checks defined in the configuration file.

There is one configuration file that manages the entire setup.

The configuration file should be managed using a version control system. For a production environment, this is mandatory.

CheckyCheck is written in Go and is started by executing one command. Optionally, additional parameters can be provided to use the program in different ways.

## Configuration File

The configuration file consists of:

- identities
- host_defaults
- host_templates
- checks
- host_groups

Secrets can be specified as environment variables. In the configuration file, you can use `${env.VARIABLE}` to refer to these variables.

### Identities

You can provide identities in three ways:

#### SSH Private Key Authentication

Your key can be located anywhere.

```json
"ssh": {
    "user": "username",
    "key": "keys/id_rsa"
}
```

#### SSH Private Key Authentication with Passphrase

```json
"ssh": {
    "user": "username",
    "key": "keys/id_rsa_username",
    "passphrase": "xxx"
}
```

#### Username and Password

You can also use simple username and password authentication. Be careful not to store passwords in a version management system.

```json
"ssh": {
    "user": "username",
    "password": "password"
}
```

### Host Defaults

Default settings for hosts can be defined here and overridden by specific hosts if needed.

```json
"host_defaults": {
    "timeout": "30s",
    "retry_interval": "10s",
    "max_retries": 3
}
```

### Host Templates

Templates for common host configurations can be defined here.

```json
"host_templates": {
    "web_server": {
        "port": 22,
        "user": "webadmin",
        "identity": "keys/webadmin_id_rsa"
    },
    "db_server": {
        "port": 22,
        "user": "dbadmin",
        "identity": "keys/dbadmin_id_rsa"
    }
}
```

### Host Groups

Define groups of hosts for easier management and checking.

```json
"host_groups": {
    "production": [
        "web1.example.com",
        "web2.example.com",
        "db1.example.com"
    ],
    "staging": [
        "staging-web1.example.com",
        "staging-db1.example.com"
    ]
}
```

## Starting CheckyCheck

If you don't have any `results.json` yet, run the application first in check mode:

```sh
checkycheck.exe -config=config.json -mode=check
```

Next, you can launch the GUI:

```sh
checkycheck.exe -port=8071 -config=config.json -mode=serve
```

## Types of Checks

All checks must be present under the "checks" key in `config.json`.

### Local Checks

Some checks are performed locally. Use the `local` option in the check.

```json
"check_url": {
    "description": "Check if a variable is set in this app",
    "graph": {
        "title": "Variable",
        "type": "bar_grouped_by_value"
    },
    "command": "curl -X GET -o /dev/null -s -w \"%{http_code}\" -I ${url}",
    "fail_when": "!=",
    "fail_value": "200",
    "local": true
}
```

### Service Checks

A Linux service can be checked like this. You can also use a command for this; the choice is yours.

```json
"check_firewall_running": {
    "description": "Check if the firewall service is running",
    "graph": {
        "title": "Firewall Status",
        "type": "bar_grouped_by_value"
    },
    "service": "ufw",
    "fail_when": "=",
    "fail_value": "0"
}
```

### Multiple Fail Values

You can specify multiple fail values.

```json
"check_url": {
    "description": "Check if a variable is set in this app",
    "graph": {
        "title": "Variable",
        "type": "bar_grouped_by_value"
    },
    "command": "curl -X GET -o /dev/null -s -w \"%{http_code}\" -I ${url}",
    "fail_when": "!=",
    "fail_value": ["200", "300"],
    "local": true
}
```

## Check Options

| Parameter   | Description                             | Default |
|-------------|-----------------------------------------|---------|
| title       | Title of the check                      |         |
| description | Description of the check                |         |
| timeout     | Timeout setting for the specific check  | 30s     |
| local       | Is this check executed locally on the CheckyCheck host | false |

## Development

### Compiling for Windows

Use the following command to create a Windows executable:

```sh
GOOS=windows GOARCH=amd64 go build -o gocheckycheck-win-amd64.exe -ldflags="-X 'main.AppVersion=v0.0.1'"
```

### Compiling for Apple M Series

```sh
GOOS=darwin GOARCH=arm64 go build -o gocheckycheck-macos-arm64 -ldflags="-X 'main.AppVersion=v0.0.1'"
```

### Creating a Zip File

```sh
zip gocheckycheck-win-amd64.zip gocheckycheck-win-amd64.exe
```

### Running Tests

You can run tests to ensure the application is working correctly.

```sh
go test ./...
```

### Contributing

If you wish to contribute to the development of CheckyCheck, please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Make your changes and ensure they are well-tested.
4. Submit a pull request with a detailed description of your changes.

### License

CheckyCheck is released under the MIT License. See the LICENSE file for more details.