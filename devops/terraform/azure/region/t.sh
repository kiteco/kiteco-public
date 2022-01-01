#!/bin/bash
terraform $1 -var "region=$2" -state "state/$2.tfstate"
