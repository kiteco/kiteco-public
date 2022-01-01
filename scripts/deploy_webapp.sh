#!/bin/bash
set -e

GOPATH=$(go env GOPATH)
KITECO="${KITECO:-$GOPATH/src/github.com/kiteco/kiteco}"

if [[ -z "$2" ]]; then
    echo "no alternative repo directory passed in"
else
    echo "alt repo directory given: $2"
    KITECO=$2
fi

REGION=us-west-1

PROD_BUCKET=www.kite.com
PROD_BUNNY_PULL_ZONE=XXXXXXX
PROD_DISTRIBUTION_ID=XXXXXXX

STAGING_BUCKET=ga-staging.kite.com
STAGING_BUNNY_PULL_ZONE=XXXXXXX
STAGING_DISTRIBUTION_ID=XXXXXXX

BUCKET=$STAGING_BUCKET
DISTRIBUTION_ID=$STAGING_DISTRIBUTION_ID
BUNNY_PULL_ZONE=$STAGING_BUNNY_PULL_ZONE

BUNNY_API_KEY=$(aws --region us-west-1 secretsmanager get-secret-value --secret-id bunny_cdn_api_key --query 'SecretString' --output text)
if [ -z "$BUNNY_API_KEY" ]
then
      echo "Can't get Bunny CDN api key from AWS"
      exit 1
fi

rm -rf $KITECO/web/app/build

if [[ $1 = "prod" ]]; then
    BUCKET=$PROD_BUCKET
    DISTRIBUTION_ID=$PROD_DISTRIBUTION_ID
    BUNNY_PULL_ZONE=$PROD_BUNNY_PULL_ZONE
    make webapp-build-prod
else
    make webapp-build-staging
fi

if [ "$SAFE_MODE" = true ]; then
    echo "deploy_webapp.sh: SAFE MODE ENABLED, EXITING BEFORE POTENTIALLY DESTRUCTIVE ACTIONS"
    exit
fi

if [[ -z "$BUCKET" ]]; then
    echo "BUCKET is not set. exiting."
    exit 1
fi

if [[ -z "$REGION" ]]; then
    echo "REGION is not set. exiting."
    exit 1
fi

if [[ -z "$DISTRIBUTION_ID" ]]; then
    echo "DISTRIBUTION_ID is not set. exiting."
    exit 1
fi

echo "using bucket: $BUCKET"
echo "using dir: $KITECO"

cd $KITECO

# remove existing files
aws s3 rm s3://$BUCKET \
    --region $REGION --recursive \
    --exclude "sitemap*.xml.gz" \
    --exclude "build_logs/*.txt" \
    --exclude "easter_egg.txt"

# sync website prod directory
# 43200 seconds == 12 hours
aws s3 sync ./web/app/build s3://$BUCKET --region $REGION --cache-control max-age=43200 \
    --exclude "index.html" \
    --exclude "62g0ni*.html"

# cp index.html with cache-control setting at 5 minutes
aws s3 cp ./web/app/build/index.html s3://$BUCKET/index.html --region $REGION --cache-control max-age=300


aws configure set preview.cloudfront true
aws cloudfront create-invalidation --distribution-id $DISTRIBUTION_ID --paths /\*

curl -H"Content-Length:0" \
     -H"Content-Type:application/json" \
     -H"AccessKey:$BUNNY_API_KEY" \
     -XPOST \
     "https://bunnycdn.com/api/pullzone/$BUNNY_PULL_ZONE/purgeCache"

if [ $? -ne 0 ]; then
    echo "Error invlidating CDN"
    exit 1
fi
