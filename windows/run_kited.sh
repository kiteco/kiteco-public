#!/bin/bash

GITCOMMIT=$(git rev-parse HEAD)
KITECO=${KITECO:-$(go env GOPATH)/src/github.com/kiteco/kiteco}
KITECO_WINDOWS=$KITECO/windows

cd $KITECO_WINDOWS

echo "Checking for the copilot (Kite.exe) ..."
if [ ! -f "win-unpacked/Kite.exe" ]; then
    echo "Please run 'kite-go/windows/build_electron.sh' to build the copilot"
    exit 1
fi


echo "Found the copilot, building kited.exe (${GITCOMMIT}) ..."

# This copy of tensorflow is ignored by the .gitignore file in this directory.
# Its needed so that kited.exe can launch successfully because tensorflow is dynamically linked
# (currently directory is part of the DLL search path). Always update it just incase.
cp tensorflow/lib/tensorflow.dll ./

rm -f kited.exe

go build \
    -buildmode=exe \
    -ldflags "-X github.com/kiteco/kiteco/kite-go/client/internal/clientapp.gitCommit=${GITCOMMIT} -X github.com/kiteco/kiteco/kite-go/client/sidebar.copilotDevDir=${KITECO_WINDOWS}" \
    github.com/kiteco/kiteco/kite-go/client/cmds/kited

./kited.exe
