#!/bin/bash

set -e

echo "Running checking"

curdir=$(cd $(dirname ${BASH_SOURCE})/..; pwd)

{
    set +e

    echo -n 'BLOCK: '
    go run $curdir/contrib/parse_comment/main.go $curdir 2> /dev/null | grep '\<BLOCK\>'
    if [ $? -eq 0 ];then
        echo 'found, exit'
    else
        echo 'not found'
    fi

    set -e
}

echo
echo 'go test:'
go clean -testcache
go test -timeout 5s -tags test -race ./... -run .

echo
echo 'go vet:'
go vet -tags test ./...

echo
echo 'errcheck:'
errcheck -tags test -ignoretests ./...

echo
echo 'golangci-run:'
golangci-lint run --build-tags test

echo
echo 'nargs:'
nargs ./...

exit 0
