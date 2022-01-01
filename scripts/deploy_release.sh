#!/bin/bash

BRANCH=
HOST=
while getopts 'b:h:' flag; do
    case "${flag}" in
        b) BRANCH=${OPTARG} ;;
        h) HOST=${OPTARG} ;;
    esac
done

if [[ -z "$BRANCH" ]]; then
    echo "branch (-b) is not set. exiting."
    exit 1
fi

if [[ -z "$HOST" ]]; then
    echo "host (-h) is not set. exiting."
    exit 1
fi

fab pull_release:$BRANCH push_release:$BRANCH,hosts="$HOST" deploy_release:$BRANCH,hosts="$HOST"
