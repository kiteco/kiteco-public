#!/bin/bash
# terraform get
# ./t.sh plan "westus2"
# exit

./t.sh apply "westus2"
./t.sh apply "eastus"
./t.sh apply "westeurope"

# ./alb_create.sh "westus2" "prod" cert.pfx
# ./alb_create.sh "westus2" "staging" cert.pfx
