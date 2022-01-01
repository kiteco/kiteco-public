#! /usr/bin/env sh

terraform init
terraform workspace select $1
shift
terraform apply "$@"

