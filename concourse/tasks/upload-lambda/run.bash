#!/usr/bin/env bash

set -e

func_name=telemetry-loader-elastic
staging_alias=staging
zipfile=build/telemetry-loader.zip
publishfile=build/telemetry-loader-publish.json

aws configure set region us-east-1
aws lambda update-function-code --function-name $func_name --zip-file fileb://$zipfile --publish > $publishfile
aws lambda update-alias --function-name $func_name --name $staging_alias --function-version $(jq -r .Version $publishfile)
