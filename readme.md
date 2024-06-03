**compileren voor windows**

Met het volgende commando kan je een windows executable maken.

`GOOS=windows GOARCH=amd64 go build -o gocheckycheck-win-amd64.exe -ldflags="-X 'main.AppVersion=v0.0.1'"`

Apple M series
`GOOS=darwin GOARCH=arm64 go build -o gocheckycheck-macos-arm64 -ldflags="-X 'main.AppVersion=v0.0.1'"`

`zip gocheckycheck-win-amd64.zip gocheckycheck-win-amd64.exe`