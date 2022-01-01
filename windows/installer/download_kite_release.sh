#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "$BASH_SOURCE[0]")"; pwd)"
echo "$DIR"

DIR_OLD="$DIR/current_build_bin/patch-setup-old"
DIR_NEW="$DIR/current_build_bin/patch-setup-new"

type wget >/dev/null 2>&1 || { echo "wget command wasn't found. Install with 'choco install wget'."; exit 1; }

VERSION="$1"
[ -z "$VERSION" ] && echo "Version not passed as argument" && exit 2

NEW_SETUP="$2"
[ -z "$NEW_SETUP" ] && echo "Path to new setup file not passed as argument" && exit 3

echo "Downloading KiteSetup of version $VERSION"
wget -q -O "$DIR/KiteSetup-$VERSION.exe" "https://kite-downloads.s3-us-west-1.amazonaws.com/windows/$VERSION/KiteSetup.exe" || { echo "Download failed with exit code $?. Terminating."; exit 4; }

rm -rf "$DIR_OLD" "$DIR_NEW"
mkdir "$DIR_OLD" "$DIR_NEW"

echo "Extracting old setup..."
( cd "$DIR_OLD"
  "$DIR/../tools/third_party/7zip/7z.exe" x -bb0 -bd -y -x'!$PLUGINSDIR' -x'!vc_redist.x64.exe' -x'!KiteSetupSplashScreen*' "$(cygpath -w $DIR/KiteSetup-$VERSION.exe)" > /dev/null
)

echo "Extracting new setup..."
( cd "$DIR_NEW"
  "$DIR/../tools/third_party/7zip/7z.exe" x -bb0 -bd -y -x'!$PLUGINSDIR' -x'!vc_redist.x64.exe' -x'!KiteSetupSplashScreen*' "$NEW_SETUP" > /dev/null
)

echo "Creating the patch files..."
cd "$DIR"
rm -rf "$DIR/KiteSetup-$VERSION.exe" "$DIR/patchFiles.nsi" "$DIR/patchFiles"
${DIR}/../tools/third_party/vpatch/nsisPatchGen.exe "$(cygpath -w $DIR_OLD)" "$(cygpath -w $DIR_NEW)"
