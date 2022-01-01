#!/usr/bin/env bash

set -e # exit if any command fails

LOGPREFIX="♦︎"
KITECO="${KITECO:-$HOME/kiteco}"

# parse args
while [[ $# > 0 ]]
do
    key="$1"
    case $key in
        --ignore-git)
        IGNORE_GIT=1
        shift
        ;;
        --no-upload)
        NO_UPLOAD=1
        shift
        ;;
        --testing)
        TESTING=1
        shift
        ;;
        --version)
        VERSION="$2"
        shift
        shift
        ;;
        *)
        # unknown option
        echo "$LOGPREFIX unkown option:" $key
        exit 1
        ;;
    esac
done

if [ -n "$TESTING" ]
then
    echo "$LOGPREFIX Skipping keychain lock..."
    PRIVKEY="/Volumes/kite_secrets/update_key/dsa_priv.pem"
else
    trap lock_keychain EXIT # lock keychain on exit
    # check for the secrets dir
    if [ ! -d "$SOLNESS_SECRETS" ]; then
        echo "$LOGPREFIX Secrets not mounted at $SOLNESS_SECRETS"
        exit 1
    fi
    security unlock-keychain -p $(cat "$SOLNESS_SECRETS/host_pass") $KEYCHAIN
    PRIVKEY="$SOLNESS_SECRETS/macos/update_key/dsa_priv.pem"
fi

# check whether we are on master
require_master () {
    if [ $(git rev-parse --abbrev-ref HEAD) != "master" ]
    then
        echo "$LOGPREFIX You must be on master to make a new release."
        exit 1
    fi
}

# check whether the working tree is clean
# from SO link: /a/3879077/554487
require_clean_work_tree () {
    # Update the index
    git update-index -q --ignore-submodules --refresh
    err=0

    # Disallow unstaged changes in the working tree
    if ! git diff-files --quiet --ignore-submodules --
    then
        echo >&2 "$LOGPREFIX cannot $1: you have unstaged changes."
        git diff-files --name-status -r --ignore-submodules -- >&2
        err=1
    fi

    # Disallow uncommitted changes in the index
    if ! git diff-index --cached --quiet HEAD --ignore-submodules --
    then
        echo >&2 "$LOGPREFIX cannot $1: your index contains uncommitted changes."
        git diff-index --cached --name-status -r --ignore-submodules HEAD -- >&2
        err=1
    fi

    if [ $err = 1 ]
    then
        echo >&2 "$LOGPREFIX Please commit or stash them."
        exit 1
    fi
}

update_submodules () {
    git submodule update --init
}

if [ -n "$IGNORE_GIT" ]
then
    echo "$LOGPREFIX Ignoring git status..."
else
    require_master
    require_clean_work_tree "make new release"
    update_submodules
    echo "$LOGPREFIX Git status looks good..."
fi


# check that we have the private key file
if [ ! -f "$PRIVKEY" ]
then
	echo "$LOGPREFIX You must have the Kite DSA private key volume mounted to make a new release."
	exit 1
fi

# check that the private key file is the correct one
if [ $(md5 -q "$PRIVKEY") != "XXXXXXX" ]
then
	echo "$LOGPREFIX You must have the Kite DSA private key volume mounted to make a new release."
	exit 1
fi
echo "$LOGPREFIX DSA private key found..."

if [ -n "$TESTING" ]
then
    echo "$LOGPREFIX Using local release server..."
    # set RDB environment variables for local release server
    export RELEASE_DB_DRIVER='sqlite3'
    export RELEASE_DB_URI='/tmp/releasedb'
else
    if grep -q stagingrelease.kite.com /etc/hosts
    then
        echo "$LOGPREFIX Found stagingrelease.kite.com in /etc/hosts..."
    else
        echo "$LOGPREFIX You should add the following entry to /etc/hosts:"
        echo "172.86.1.21   stagingrelease.kite.com"
        exit 1
    fi

    # set RDB environment variables for staging release server
    export RELEASE_DB_DRIVER='postgres'
    export RELEASE_DB_URI='postgres://XXXXXXX:XXXXXXX@XXXXXXX/release'
fi

echo "$LOGPREFIX Everything looks good.  Let's do this."

cd "$KITECO/osx"
echo "$LOGPREFIX Building release tool"
go build github.com/kiteco/kiteco/kite-go/cmds/release

# compute versions
mkdir -p tmp_prepare/release_bin
mkdir -p tmp_prepare/version
mkdir -p tmp_prepare/slack
ln -s "$KITECO" tmp_prepare/kiteco
ln -s "$KITECO/osx/release" tmp_prepare/release_bin/release
(
  # subshell prevents using the prod release DB permanently
  eval $(ssh -i ~/.ssh/kite-prod.pem XXXXXXX 'bash --login -c "env | grep ^RELEASE_DB_DRIVER="' | sed 's/.*/export &/')
  eval $(ssh -i ~/.ssh/kite-prod.pem XXXXXXX 'bash --login -c "env | grep ^RELEASE_DB_URI="' | sed 's/.*/export &/')
  cd tmp_prepare
  platform=mac $KITECO/concourse/tasks/prepare-release/run.bash
)
PREVVERSION=$(cat tmp_prepare/version/prev)
VERSION=$(cat tmp_prepare/version/version)
# make sure we delete the symbolic link, and not the contents of $KITECO
rm tmp_prepare/kiteco
rm -r tmp_prepare

DELTA="$PREVVERSION-$VERSION.delta"

echo "$LOGPREFIX Setting Kite.app's version in Info.plist to $VERSION"
plutil -replace CFBundleShortVersionString -string $VERSION $KITECO/osx/Kite/Info.plist
plutil -replace CFBundleVersion -string $VERSION $KITECO/osx/Kite/Info.plist

echo "$LOGPREFIX Building Kite.app"
echo "$LOGPREFIX BUILD_DIR: $BUILD_DIR"
BUILD_DIR=build
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR
xcodebuild -scheme Kite -configuration Release APP_VERSION=$VERSION CODE_SIGN_IDENTITY="Developer ID Application" OTHER_CODE_SIGN_FLAGS="--timestamp" -derivedDataPath $BUILD_DIR

echo "$LOGPREFIX Reverting Kite.app's version, for dev"
git checkout $KITECO/osx/Kite/Info.plist

RELEASE_DIR=releases
if [ ! -n "$TESTING" ]
then
    rm -rf $RELEASE_DIR
fi
mkdir -p $RELEASE_DIR

DMGFILE="Kite-$VERSION.dmg"
DMGSRC="tmp_dmg_src"
echo "$LOGPREFIX creating Kite.dmg"
rm -f $DMGFILE
rm -rf $DMGSRC
for DISK in $(diskutil list | grep Kite | awk '{ print $6 }' | sed -E 's/(disk[0-9]+).*/\1/')
do
    diskutil unmountDisk $DISK
done

mkdir -p $DMGSRC

cleanup () {
    echo "$LOGPREFIX Cleaning up..."
    if [ -n "$TESTING" ]
    then
        NEXT_DIR="$RELEASE_DIR/$VERSION"
        mkdir -p $NEXT_DIR
        mv "$DMGSRC/Kite.app" $NEXT_DIR
    else
        rm -f $DMGFILE $DELTA 
        rm -rf $RELEASE_DIR
    fi
    rm -rf $BUILD_DIR $DMGSRC
    rm -f ./release $DMGARCHIVE
}

trap cleanup EXIT

# NOTE: You can't use here cp -r because it does not preserve extended file attributes.
# If you need to copy/zip/gzip/bzip/etc, consider using ditto.
mv $BUILD_DIR/Build/Products/Release/Kite.app $DMGSRC
$KITECO/scripts/create-dmg/create-dmg \
    --volname "Kite" \
    --window-pos 200 120 \
    --window-size 562 330 \
    --background "$KITECO/osx/dmg_bg.png" \
    --icon-size 100 \
    --icon Kite.app 125 140 \
    --hide-extension Kite.app \
    --app-drop-link 400 140 \
    $DMGFILE \
    $DMGSRC

echo "$LOGPREFIX Signing update with DSA"
SIGNATURE=$($KITECO/scripts/sign_update.sh $DMGFILE $PRIVKEY)

echo "$LOGPREFIX Creating Kite.app archive"
DMGARCHIVE="Kite.tar.gz"
tar -czvf $DMGARCHIVE -C $DMGSRC .

echo "$LOGPREFIX Checking git hash"
GITHASH=$(git rev-parse HEAD)

if [ -n "$TESTING" ]
then
    echo "$LOGPREFIX Skipping notarization..."
else
    echo "$LOGPREFIX Notarizing DMG"
    xcrun altool \
        --notarize-app \
        --primary-bundle-id com.kite.Kite \
        -u "product-engineering@kite.com" \
        -p "@keychain:AC_PASSWORD" \
        --file $DMGFILE
fi

MAC_BUCKET='s3://kite-downloads/mac'
CURRENT_DIR="$RELEASE_DIR/$PREVVERSION"
CURRENT_APP="$CURRENT_DIR/Kite.app"

if [ -n "$TESTING" ]
then
    if [ ! -d "$CURRENT_APP" ]
    then
        echo "$LOGPREFIX $CURRENT_APP does not exist, skipping creating delta update..."
        SKIP_DELTA=1
    fi
else
    KEY="$PREVVERSION/Kite.tar.gz"
    aws s3 ls "$MAC_BUCKET/$KEY" || not_exist=true
    if [ $not_exist ]
    then
        echo "$LOGPREFIX $MAC_BUCKET/$KEY does not exist, skipping creating delta update..."
        SKIP_DELTA=1
    else
        mkdir -p $CURRENT_DIR
        aws s3 cp "$MAC_BUCKET/$KEY" "$CURRENT_DIR/Kite.tar.gz"
        tar -xzvf "$CURRENT_DIR/Kite.tar.gz" -C $CURRENT_DIR
    fi
fi

if [ ! -n "$SKIP_DELTA" ]
then
    echo "$LOGPREFIX creating delta patch from $PREVVERSION to $VERSION..."
    $KITECO/scripts/create_sparkle_delta.sh --from-app $CURRENT_APP --to-app "$DMGSRC/kite.app" --delta "$DELTA"

    echo "$LOGPREFIX signing delta patch with dsa"
    DELTASIGNATURE=$($KITECO/scripts/sign_update.sh "$DELTA" "$PRIVKEY")
fi

if [ -n "$NO_UPLOAD" ]
then
    (
        # add release
        set -ax
        _PLATFORM=mac
        _VERSION=$VERSION
        _GIT_HASH=$GITHASH
        _CANARY_PERCENTAGE=100
        _SIGNATURE=$SIGNATURE
        ./release add
    )
    echo "$LOGPREFIX Not uploading binary, leaving it at $DMGFILE"
    echo "$LOGPREFIX   DSA signature: $SIGNATURE"
    echo "$LOGPREFIX   Git hash: $GITHASH"
    echo "$LOGPREFIX   Added release $VERSION to release server"
    if [ ! -n "$SKIP_DELTA" ]
    then
        # add delta
        (
            set -ax
            _NUM_DELTAS=1
            _PLATFORM_DELTA_0=mac
            _FROM_VERSION_DELTA_0=$PREVVERSION
            _VERSION_DELTA_0=$VERSION
            _SIGNATURE_DELTA_0=$DELTASIGNATURE
            ./release addDeltas
        )
        echo "$LOGPREFIX Not uploading delta file, leaving it at $RELEASE_DIR/$DELTA"
        echo "$LOGPREFIX Added delta $DELTA to release server"
    fi
else
    # upload release
    aws s3 cp $DMGFILE "$MAC_BUCKET/$VERSION/Kite.dmg" --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers --cache-control max-age=604800
    aws s3 cp $DMGARCHIVE "$MAC_BUCKET/$VERSION/Kite.tar.gz" --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers --cache-control max-age=604800
    # add release
    (
        set -ax
        _PLATFORM=mac
        _VERSION=$VERSION
        _GIT_HASH=$GITHASH
        _CANARY_PERCENTAGE=100
        _SIGNATURE=$SIGNATURE
        ./release add
    )

    if [ ! -n "$SKIP_DELTA" ] && [ -f $DELTA ]
    then
        # upload delta
        aws s3 cp $DELTA "$MAC_BUCKET/$VERSION/deltaFrom/$PREVVERSION/Kite.delta" --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers --cache-control max-age=604800
        # add delta
        (
            set -ax
            _NUM_DELTAS=1
            _PLATFORM_DELTA_0=mac
            _FROM_VERSION_DELTA_0=$PREVVERSION
            _VERSION_DELTA_0=$VERSION
            _SIGNATURE_DELTA_0=$DELTASIGNATURE
            ./release addDeltas
        )
    fi
fi


if [ -n "$TESTING" ]
then
    exit 0
fi
