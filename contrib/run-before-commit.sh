#!/bin/bash

set -e
set -x

echo "Running checking"

# go test
go clean -testcache
go test -timeout 5s -tags test -race ./... -run .

# go vet
go vet -tags test ./...

# errcheck
errcheck -tags test -ignoretests ./...

# golangci-run
golangci-lint run --build-tags test

# nargs
nargs ./...

exit 0
