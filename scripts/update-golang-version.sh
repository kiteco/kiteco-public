#!/bin/bash

(( $# != 2 )) && { echo "Usage: $0 <current-version> <new-version>." 2>&1; exit 1; }

CURRENT_VERSION="$1"
NEW_VERSION="$2"

DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)"

# update standalone.sh
sed -i -e "s/$CURRENT_VERSION/$NEW_VERSION/g" "$DIR/standalone.sh"

# update .travis.yml
sed -i -e "s/go: $CURRENT_VERSION/go: $NEW_VERSION/g" "$DIR/../.travis.yml"

# update README.md
sed -i -e "s/Go $CURRENT_VERSION/Go $NEW_VERSION/g" "$DIR/../README.md"

# update build_libkited.sh
sed -i -e "s/$CURRENT_VERSION/$NEW_VERSION/g" "$DIR/../osx/build_libkited.sh"

# update Concourse
sed -i -e "s/GO_VERSION=$CURRENT_VERSION/GO_VERSION=$NEW_VERSION/g" "$DIR/../concourse/images/docker/Dockerfile"

# update install-golang.sh
sed -i -e "s/VERSION=$CURRENT_VERSION/VERSION=$NEW_VERSION/g" "$DIR/../devops/scripts/install-golang.sh"
