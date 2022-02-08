#!/bin/bash

set -ev

go fmt *.go
golint -set_exit_status *.go
staticcheck *.go
go vet *.go
