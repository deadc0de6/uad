#!/bin/bash

go fmt *.go
golint *.go
go vet *.go
