#!/usr/bin/env bash
# this scripts checks all added lines of **/*.go, except for vendor/*

OS=`uname -s`

if [[ "${OS}" == "Darwin" ]]; then
    echo "checking for uses of fmt, pkg/errors in git diff..." && echo && \
        ! git diff --diff-filter=AM --name-only master..HEAD '**/*.go' |
        grep -v -e '^vendor' |
        xargs -t git diff master..HEAD |
        grep "^+" |
        grep -e 'fmt.Errorf' -e 'github.com/pkg/errors' &&
        echo no errors || ( echo && echo "failed: please use kite-golib/errors" && false )
else
    echo "checking for uses of fmt, pkg/errors in git diff..." && echo && \
        ! git diff --diff-filter=AM --name-only master..HEAD '**/*.go' |
        grep -v -e '^vendor' |
        xargs -r -t git diff master..HEAD |
        grep "^+" |
        grep -e 'fmt.Errorf' -e 'github.com/pkg/errors' &&
        echo no errors || ( echo && echo "failed: please use kite-golib/errors" && false )
fi
