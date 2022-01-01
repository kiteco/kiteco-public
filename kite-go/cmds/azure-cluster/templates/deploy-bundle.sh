#!/usr/bin/env bash

set -e

BUNDLE_FILE=/var/kite/upload/bundle.tar.gz
BUNDLE_PATH=/var/kite

tar xzvf $BUNDLE_FILE -C $BUNDLE_PATH

bash /var/kite/bundle/setup.sh
source /var/kite/bundle/env.sh

echo "export INSTANCE_ID=`cat /var/kite/instance-id`" >> /var/kite/bundle/env.sh
echo "export INSTANCE_COUNT=`cat /var/kite/instance-count`" >> /var/kite/bundle/env.sh

bash /var/kite/bundle/start.sh

# stop these clusters once the bundle has successfully finished
{{range $cluster := .CleanupClusters}}
echo "Stopping {{$cluster}}"
azure-cluster stop {{$cluster}}
{{end}}
