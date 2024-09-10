#!/bin/bash

go run main.go remote_check.go types.go helpers.go -mode=check -config=config-sal.json
go run main.go remote_check.go types.go helpers.go -mode=serve -config=config-sal.json -port=8070