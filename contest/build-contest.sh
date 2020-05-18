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

version="v0.0.1-proto3+commit.$(git rev-parse --short HEAD)"

diff=$(git diff -b)
if [ ! -z "${diff}" ];then
    version="$version-patched.$(git diff -b | md5sum -t - | awk '{print $1}')"
fi

echo $version
GOOS=linux GOARCH=amd64 go build \
    -race \
    -ldflags "-X github.com/spikeekips/mitum/contest/cmds.Version=${version}" \
    -v \
    -o $1 \
    ./contest/main.go

echo 'build finished:' $1
