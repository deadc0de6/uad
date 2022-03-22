#!/bin/bash

set -ev

# linting
go fmt *.go
golint -set_exit_status *.go
staticcheck *.go
go vet *.go

# compiling
make clean
make
