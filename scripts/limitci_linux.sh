#!/usr/bin/env bash
if [[ "$TRAVIS_OS_NAME" != "linux" ]]
then
    echo "Skipping '$2' in $TRAVIS_OS_NAME"
    exit 0
fi

TARGET=master
if [[ "$TRAVIS_BRANCH" == "master" ]]
then
    TARGET="master~1"
fi

if `git diff --name-only $TARGET | grep --quiet "$1"`
then
	echo "Found changes matching $1; running '$2'"
	$2
else
	echo "No changes matching $1; skipping '$2'"
fi
