#!/usr/bin/env bash

set -e

func_name=telemetry-loader-elastic
staging_alias=staging
prod_alias=production
publishfile=build/telemetry-loader-publish.json

aws configure set region us-east-1

aws lambda get-alias --function-name $func_name  --name $staging_alias > staging_alias.json
aws lambda update-alias --function-name $func_name --name $prod_alias --function-version $(jq -r .FunctionVersion staging_alias.json)
