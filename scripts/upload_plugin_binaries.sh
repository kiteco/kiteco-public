#!/bin/bash

# Uploads plugin binaries to S3
#
# If called with "pub", it will give the files public read permission

set -e

BUCKET=kite-plugin-binaries

# Plugins to upload (refer to the go generate lines at the top of the <editor>.go files in
# kiteco/kite-go/client/internal/plugins/<editor> for these locations)
KITECO="${KITECO:-$HOME/kiteco}"
PLUGINS_DIR="$KITECO/plugins"
ST3_PLUGIN="$PLUGINS_DIR/sublimetext3-plugin/st_package_builder/target/st3/Kite.sublime-package"
PYCHARM_PLUGIN="$PLUGINS_DIR/intellij/proguard/kite-pycharm-obsfucated-*.zip"
DATE=`date +"%Y%m%d%H%M%S"`

echo "uploading plugins from $PLUGINS_DIR"

if [ "$SAFE_MODE" = true ]; then
    echo "upload_plugin_binaries.sh: SAFE MODE ENABLED, EXITING BEFORE POTENTIALLY DESTRUCTIVE ACTIONS"
    exit
fi

aws s3 cp $ST3_PLUGIN s3://$BUCKET/$DATE/Kite.sublime-package
aws s3 cp $PYCHARM_PLUGIN s3://$BUCKET/$DATE/kite-pycharm-obsfucated.zip

if [[ $1 = "pub" ]]; then
  aws s3 cp $ST3_PLUGIN s3://$BUCKET/latest/Kite.sublime-package --acl public-read
  aws s3 cp $PYCHARM_PLUGIN s3://$BUCKET/latest/kite-pycharm-obsfucated.zip --acl public-read
fi
