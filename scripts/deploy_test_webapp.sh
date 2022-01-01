#!/bin/bash
set -e

KITECO="${KITECO:-$GOPATH/src/github.com/kiteco/kiteco}"

REACT_APP_TEST_BACKEND="${REACT_APP_TEST_BACKEND:-https://staging.kite.com}"

TESTING_BUCKET=web-dev.kite.com

DISTRIBUTION_ID=XXXXXXX

REGION=us-west-1

rm -rf $KITECO/web/app/build

BUCKET=$TESTING_BUCKET

echo "using backend: $REACT_APP_TEST_BACKEND"
make REACT_APP_TEST_BACKEND=$REACT_APP_TEST_BACKEND webapp-build-testing

if [[ -z "$AWS_ACCESS_KEY_ID" ]]; then
    echo "AWS_ACCESS_KEY_ID is not set. exiting."
    exit 1
fi

if [[ -z "$AWS_SECRET_ACCESS_KEY" ]]; then
    echo "AWS_SECRET_ACCESS_KEY is not set. exiting."
    exit 1
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
    --exclude "XXXXXXX*.html"

# cp index.html with cache-control setting at 5 minutes
aws s3 cp ./web/app/build/index.html s3://$BUCKET/index.html --region $REGION --cache-control max-age=300

# cloudfront cli access is in preview, need to enable
aws configure set preview.cloudfront true
# invalidate everything in the cache on deploy

aws cloudfront create-invalidation --distribution-id $DISTRIBUTION_ID --paths /\*
