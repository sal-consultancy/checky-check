#!/bin/bash
RELEASE=$(cat version.txt)
git tag "v${RELEASE}"
git-chglog -o CHANGELOG.md
cd frontend
npm install
npm run build
cd ..

# compiling for windows
GOOS=windows GOARCH=amd64 
FILE=checkycheck-${RELEASE}-${GOOS}-${GOARCH}
go build -o ${FILE}.exe main.go remote_check.go types.go helpers.go 
zip ${FILE}.zip ${FILE}.exe

# compiling for mac
GOOS=darwin GOARCH=arm64
FILE=checkycheck-${RELEASE}-${GOOS}-${GOARCH}
go build -o ${FILE}main.go remote_check.go types.go helpers.go 
zip ${FILE}.zip ${FILE}

git commit -am "rel: Bumping for releasing version v${RELEASE}"
git push

