#!/bin/bash

[ -z "${GIT_PRE_PUSH_SKIP}" ] || exit 0

echo "Running checking"

curdir=$(cd $(dirname ${BASH_SOURCE})/..; pwd)

echo 'BLOCK: '
output=$(mktemp)
go run $curdir/contrib/parse_comment/main.go $curdir 2> /dev/null | grep '\<BLOCK\>'
if [ $? -eq 0 ];then
    echo 'found, exit:'
    cat $output
    exit 1
else
    echo 'not found'
fi

exit 0
