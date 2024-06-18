**compileren voor windows**

Met het volgende commando kan je een windows executable maken.

`GOOS=windows GOARCH=amd64 go build -o gocheckycheck-win-amd64.exe -ldflags="-X 'main.AppVersion=v0.0.1'"`

Apple M series
`GOOS=darwin GOARCH=arm64 go build -o gocheckycheck-macos-arm64 -ldflags="-X 'main.AppVersion=v0.0.1'"`

`zip gocheckycheck-win-amd64.zip gocheckycheck-win-amd64.exe`

## Configuration file

Bestaat uit:

- identities
- host_defaults
- host_templates
- checks
- host_groups

Eventuele geheimen kan je bekend maken als environment variabele.

## Identities

Je kan op drie manieren identities aanbrengen:

SSH Private Key Authentication
```
"ssd": {
    "user": "username",
    "key": "keys/id_rsa",
}
```        

SSH Private Key Authentication met wachtwoord
```
"ssd": {
    "user": "username",
    "key": "keys/id_rsa_",
    "passphrase": "xxx"
}
```        

Username en wachtword
```
"ssd": {
    "user": "username",
    "password": "password"
}
```        

## Opstarten

`checkycheck.exe -mode=serve -port=8071 -config=config.json`



## Type checks

Lokale checks

```
        "check_url": {
            "description":"Controleer of er een variabele gezet wordt in deze app",
            "graph": {
                "title":"Variabele",
                "type": "bar_grouped_by_value"
            },
            "command": "curl -X GET -o /dev/null -s -w \"%{http_code}\" -I ${url}",
            "fail_when": "!=",
            "fail_value": "200",
            "local": true
        }
```

Check een service

```
        "check_firewall_running": {
            "description":"Controleer of er een variabele gezet wordt in deze app",
            "graph": {
                "title":"Variabele",
                "type": "bar_grouped_by_value"
            },
            "service": "ufw",
            "fail_when": "=",
            "fail_value": "0"
        },
```

multiple values

```
        "check_url": {
            "description":"Controleer of er een variabele gezet wordt in deze app",
            "graph": {
                "title":"Variabele",
                "type": "bar_grouped_by_value"
            },
            "command": "curl -X GET -o /dev/null -s -w \"%{http_code}\" -I ${url}",
            "fail_when": "!=",
            "fail_value": ["200","300"],
            "local": true
        }
```