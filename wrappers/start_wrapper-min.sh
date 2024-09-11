#!/bin/bash

go run main.go remote_check.go types.go helpers.go -mode=check -config=config-sal-min.json
go run main.go remote_check.go types.go helpers.go -mode=serve -config=config-sal-min.json -port=8070