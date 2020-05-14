#!/bin/sh

set -e
#set -x

echo 'starting to  build:' $1

version="v0.0.1-proto3+commit.$(git rev-parse --short HEAD)"

diff=$(git diff -b)
if [ ! -z "${diff}" ];then
    version="$version-patched.$(git diff -b | md5sum -t - | awk '{print $1}')"
fi

GOOS=linux GOARCH=amd64 go build \
    -race \
    -ldflags="-X 'main.Version=${version}'" \
    -v \
    -o $1 \
    ./contest/runner/main.go

echo 'build finished:' $1
