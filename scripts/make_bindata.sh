#!/usr/bin/env bash

# Script for running the various make commands to generate bindata

# NOTE: part of the reason why this is here and not in slackbuildbot is because make seems to be
# unhappy about being run from subprocess.Popen and doesn't get run in the correct working
# directory

set -e

# set KITECO to $HOME/kiteco if it's not already specified
KITECO="${KITECO:-$HOME/kiteco}"
echo "make_bindata.sh: KITECO set to ${KITECO}"
cd $KITECO
