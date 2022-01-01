#! /usr/bin/env sh

terraform init
terraform workspace select production
terraform apply "$@"
