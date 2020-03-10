#!/bin/bash

[ -z "${GIT_PRE_COMMIT_SKIP}" ] || exit 0

set -e

echo "Running checking"

curdir=$(cd $(dirname ${BASH_SOURCE})/..; pwd)

bash $curdir/contrib/check-BLOCK.sh

echo
echo 'go test:'
go clean -testcache
#go test -timeout 1m -tags test -race ./... -run .
go clean -testcache; for i in $(find . -type d -d 1 | grep -v '.git\|.circleci'); do go test -timeout 10s -tags test -race ./$i... -run .; done

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
