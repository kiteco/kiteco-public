#!/bin/bash

set -e

GOPATH=$HOME/go
KITECO=$GOPATH/src/github.com/kiteco/kiteco
LAMBDA_DIR="$KITECO/lambda-functions"
TMP_DIR="completion-performance"
LAMBDA_FUNCTION="completion-performance"
REGION="us-west-1"
LAMBDA_FILE="completion-performance.py"
LAMBDA_NAME=${LAMBDA_FILE%.*}

echo "updating lambda function $LAMBDA_FUNCTION"

# copy lambda function script files into temp dir
cd "$LAMBDA_DIR"
mkdir -p "$TMP_DIR"
cp $LAMBDA_FILE $TMP_DIR/main.py

# install dependencies
cd "$TMP_DIR"
pip3 install requests -t .
pip3 install requests_aws4auth -t .
pip3 install elasticsearch -t .

# create and upload zip
ZIPFILE="$LAMBDA_NAME.zip"
zip -r $ZIPFILE *
aws configure set region $REGION
aws lambda update-function-code --function-name "$LAMBDA_FUNCTION" --zip-file "fileb://$ZIPFILE"

# cleanup
cd ..
rm -rf "$TMP_DIR"
