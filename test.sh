#!/bin/bash

set -ev

# deps
go install golang.org/x/lint/golint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# linting
go fmt *.go
golint -set_exit_status *.go
staticcheck *.go
go vet *.go

# compiling
make clean
make
