#!/usr/bin/env bash
apt-get update && apt-get install -y jq
echo -n 'versions=' > tfvars/terraform.tfvars
cat $BUILD/version | jq -cR ". as \$version | $VERSIONS | map_values(if .==\"VERSION\" then \$version else . end)" >> tfvars/terraform.tfvars
