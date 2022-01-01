#!/bin/bash

DEST=/var/kite/data/
BUCKET=s3://kite-data/var-kite-data/

sudo s3cmd sync $BUCKET $DEST \
    --access_key=${KITE_DATA_ACCESS_KEY} \
    --secret_key=${KITE_DATA_SECRET_KEY} \
    --ssl \
    --follow-symlinks \
    --recursive \
    --human-readable-sizes \
    --no-delete-removed \
    --progress

sudo chown -R $USER:$USER $DEST
