#!/bin/bash

GOPATH=$HOME/go
KITECO=$GOPATH/src/github.com/kiteco/kiteco
BUCKET=kite.com
REGION=us-west-1
DISTRIBUTION_ID=XXXXXXX

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

cd $KITECO
# remove existing files
aws s3 rm s3://$BUCKET --region $REGION --recursive
# sync website prod directory
aws s3 sync ./website/prod s3://$BUCKET --region $REGION

# cloudfront cli access is in preview, need to enable
aws configure set preview.cloudfront true
# invalidate everything in the cache on deploy
aws cloudfront create-invalidation --distribution-id $DISTRIBUTION_ID --paths /*
