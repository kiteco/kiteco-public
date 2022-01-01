# Cygwin Bash.exe

set -e

export GOPATH=$(cygpath -w "$PWD/gopath")
export GO111MODULE="on"
export GOPRIVATE=github.com/kiteco/*
KITECO=$PWD/gopath/src/github.com/kiteco/kiteco/

COMMIT=$(cat version/commit)
VERSION=$(cat version/version)
PREVVERSION=$(cat version/prev)

echo "Building Windows client..."
echo "VERSION=$VERSION"
echo "COMMIT=$COMMIT"
echo "PREVVERSION=$PREVVERSION"
echo

REACT_APP_ENV=production $KITECO/windows/build_electron.sh
WINDOWS_BUILD_VERSION=$VERSION WINDOWS_PATCH_BASE=$PREVVERSION make -f $KITECO/Makefile -C $KITECO KiteSetup.exe KiteUpdateInfo.xml KitePatchUpdateInfo.xml

echo && echo

OUTDIR=$KITECO/windows/installer/builds/$VERSION

mv $OUTDIR/KiteSetup$VERSION.exe build/KiteSetup.exe
mv $OUTDIR/KiteUpdater$VERSION.exe build/KiteUpdater.exe
mv $OUTDIR/KiteUpdateInfo.xml build/KiteUpdateInfo.xml
echo "PLATFORM=windows" >> build/META
echo "VERSION=$VERSION" >> build/META
echo "COMMIT=$COMMIT" >> build/META
echo "SIGNATURE=" >> build/META
echo "build/META:"
cat build/META && echo

if [ -n "$PREVVERSION" ]; then
    mkdir -p build/deltaFrom/$PREVVERSION
    mv $OUTDIR/KitePatchUpdater$PREVVERSION-$VERSION.exe build/deltaFrom/$PREVVERSION/KiteDeltaUpdater.exe
    mv $OUTDIR/KitePatchUpdateInfo$PREVVERSION.xml build/deltaFrom/$PREVVERSION/KiteDeltaUpdateInfo.xml
    echo "DELTA_FROM[0]=$PREVVERSION" >> build/META
    echo "PLATFORM=windows" >> build/deltaFrom/$PREVVERSION/META
    echo "FROM_VERSION=$PREVVERSION" >> build/deltaFrom/$PREVVERSION/META
    echo "TO_VERSION=$VERSION" >> build/deltaFrom/$PREVVERSION/META
    echo "SIGNATURE=" >> build/deltaFrom/$PREVVERSION/META
    echo "build/deltaFrom/$PREVVERSION/META:"
    cat build/deltaFrom/$PREVVERSION/META
fi
