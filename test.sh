#!/bin/bash

set -ev

go fmt *.go
golint -set_exit_status *.go
go vet *.go
