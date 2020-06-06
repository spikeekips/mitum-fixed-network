#!/bin/sh

set -e
#set -x

root=$(cd $(dirname ${BASH_SOURCE})/../; pwd)

cd $root

if [ -z $1 ];then
    echo "Usage: $0 <output file>"
    exit 1
fi

echo 'starting to build:' $1

version="v0.0.1-$(git branch -q --show-current)+commit.$(git rev-parse --short HEAD)"

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
