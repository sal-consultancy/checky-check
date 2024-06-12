**compileren voor windows**

Met het volgende commando kan je een windows executable maken.

`GOOS=windows GOARCH=amd64 go build -o gocheckycheck-win-amd64.exe -ldflags="-X 'main.AppVersion=v0.0.1'"`

Apple M series
`GOOS=darwin GOARCH=arm64 go build -o gocheckycheck-macos-arm64 -ldflags="-X 'main.AppVersion=v0.0.1'"`

`zip gocheckycheck-win-amd64.zip gocheckycheck-win-amd64.exe`


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
