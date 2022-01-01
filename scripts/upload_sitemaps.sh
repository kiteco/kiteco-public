#!/usr/bin/env bash

set -e

BUNNY_API_KEY=$(aws --region us-west-1 secretsmanager get-secret-value --secret-id XXXXXXX --query 'SecretString' --output text)
if [ -z "$BUNNY_API_KEY" ]
then
      echo "Can't get Bunny CDN api key from AWS"
      exit 1
fi

go build github.com/kiteco/kiteco/kite-go/cmds/site-map-generator
mkdir -p $HOME/sitemaps
./site-map-generator -generate -upload -prod -dir=$HOME/sitemaps -bucket=www.kite.com
rm -f site-map-generator

curl -H"Content-Length:0" \
     -H"Content-Type:application/json" \
     -H"AccessKey:$BUNNY_API_KEY" \
     -XPOST \
     "https://bunnycdn.com/api/pullzone/$DISTRIBUTION_ID/purgeCache"
