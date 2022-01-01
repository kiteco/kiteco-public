#!/bin/bash

GOROOT=/usr/local/go
GOPATH=$HOME/go
KITECO=$GOPATH/src/github.com/kiteco/kiteco
CURATION=$HOME/curation-team

# So we can find go and other installed binaries
PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# check_and_notify takes a commands return code ($1) and a message to display
# if the return code is not 0 ($2). NOTE: Use $? to check the return code of a
# command after running it.
function check_and_notify {
	if [[ $1 != 0 ]]; then
        echo "$1"
		echo "$2"
		exit $1
	fi
}

echo "verifying curation team repo and pulling most recent version"
cd $CURATION
# ensure we are on master
BRANCH=`git symbolic-ref --short -q HEAD`
if [ "$BRANCH" != "master" ]; then
	check_and_notify 1 "FATAL: on $BRANCH instead of master"
fi
git reset --hard
git pull

cd $KITECO/local-pipelines/curation-python-skeletons
echo "removing old binary ..."
make clean-curation-validation-tool
check_and_notify $? "could not remove old binary"

echo "building binary ..."
make curation-validation-tool
check_and_notify $? "could not build binary"

echo "committing changes to git ..."
cd $CURATION

git commit python-skeletons/validate-build/validate-build -m "[autocommit] updating skeleton curation validation tool"
check_and_notify $? "could not commit binary"

git push -u origin master
check_and_notify $? "could not push to master"