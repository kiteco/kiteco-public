#!/bin/bash
terraform output -state "state/$1.tfstate" $2
