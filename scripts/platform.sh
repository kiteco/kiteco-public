#!/usr/bin/env bash
if [[ "$TRAVIS_OS_NAME" == $1 ]]; then
    echo "Running command '$2' in $1"
    $2
fi
