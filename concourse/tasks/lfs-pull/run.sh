#!/usr/bin/env bash

set -e

# Set up git config: adapted from github.com/concourse/git-resource
private_key_path=$PWD/private_key

echo "$private_key" > $private_key_path

if [ -s $private_key_path ]; then
  chmod 0600 $private_key_path

  eval $(ssh-agent) >/dev/null 2>&1
  trap "kill $SSH_AGENT_PID" EXIT

  SSH_ASKPASS=/opt/resource/askpass.sh DISPLAY= ssh-add $private_key_path >/dev/null

  mkdir -p ~/.ssh
  cat > ~/.ssh/config <<EOF
StrictHostKeyChecking no
LogLevel quiet
EOF
  if [ "$forward_agent" = "true" ]; then
    cat >> ~/.ssh/config <<EOF
ForwardAgent yes
EOF
  fi
  chmod 0600 ~/.ssh/config
fi

# Set up LFS object cache
repo="$PWD/repo"
cache="$PWD/.lfscache"
rm -rf $repo/.git/lfs
ln -s $cache $repo/.git/lfs

# LFS pull
cd $repo
git lfs pull
git reset
