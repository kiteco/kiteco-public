#!/usr/bin/env bash
OUTPUT_DIR=$1
if [[ -z $1 ]]; then
    OUTPUT_DIR="$(mktemp -d)"
fi

if [[ ! -d /var/kite ]]; then
    sudo mkdir -p /var/kite
    sudo chown -R $USER:staff /var/kite
fi

TEMP_S3_CACHE="$(mktemp -d)"
ORIG_S3_CACHE=$KITE_S3CACHE
export KITE_S3CACHE=$TEMP_S3_CACHE

# TODO we should probably generalize these scripts as well, and move them out of `kite-go/client/internal/kitelocal/...`
go install github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/cmds/datadeps-bindata
go build github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/cmds/build-datadeps-filemap
./build-datadeps-filemap -output="$OUTPUT_DIR"
if [ $? -eq 0 ]; then
    # Update the bindata because dataset has changed
    echo "running go generate on datadeps"
    DATADEPS_DIR="$(go list -f '{{.Dir}}' github.com/kiteco/kiteco/kite-go/client/datadeps)"
    datadeps-bindata -data="$OUTPUT_DIR/data.blob" -offsets="$OUTPUT_DIR/offsets.gob" -pkg=datadeps -output="$DATADEPS_DIR/datadeps-bindata.go"
fi

export KITE_S3CACHE=$ORIG_S3_CACHE
rm -rf $TEMP_S3_CACHE

rm -f build-datadeps-filemap
if [[ -z $1 ]]; then
    rm -rf "$OUTPUT_DIR"
fi
