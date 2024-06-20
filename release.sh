#!/bin/bash
RELEASE=serveit3.1
git tag ${RELEASE}
git-chglog -o CHANGELOG.md
cd frontend
npm run build
cd ..
go build -o checkycheck-${RELEASE}.exe main.go remote_check.go types.go helpers.go 
zip checkycheck-${RELEASE}.zip checkycheck-${RELEASE}.exe
git commit -am "rel: Releasing version ${RELAEASE}"
git push

