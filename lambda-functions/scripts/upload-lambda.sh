#!/bin/bash

# Uploads lambda function to S3
# Usage: ./upload-lambda.sh <function name> <region> <function file>


set -e

GOPATH=$(go env GOPATH)
KITECO=$GOPATH/src/github.com/kiteco/kiteco
LAMBDA_DIR="$KITECO/lambda-functions"
COMMON_DIR="$LAMBDA_DIR/common"
TMP_DIR="s3toes"
LAMBDA_FUNCTION=$1
REGION=$2
LAMBDA_FILE=$3
LAMBDA_NAME=${LAMBDA_FILE%.*}

echo "updating lambda function $LAMBDA_FUNCTION"

# copy lambda function script files into temp dir
cd $LAMBDA_DIR 
mkdir $TMP_DIR
cp $LAMBDA_FILE $TMP_DIR/iter_actions.py
cp $COMMON_DIR/* $TMP_DIR

# install dependencies
cd $TMP_DIR
pip3 install requests -t .
pip3 install requests_aws4auth -t .
pip3 install elasticsearch -t .

# create and upload zip
ZIPFILE="$LAMBDA_NAME.zip"
zip -r $ZIPFILE *
aws configure set region $REGION
aws lambda update-function-code --function-name $LAMBDA_FUNCTION --zip-file "fileb://$ZIPFILE"

# cleanup
cd ..
rm -rf $TMP_DIR
