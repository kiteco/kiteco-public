# rc.kite.com

## Dependencies:

 * gcloud cli: https://cloud.google.com/sdk/docs/quickstart
 * kubectl: `gcloud components install kubectl`
 * ytt: `brew tap k14s/tap; brew install ytt`

## Running locallly

source local.sh
docker-compose up

## Deploying

Build and publish: `make docker.all`
Deploy staging: `make deployment.apply`
Check status: `make deployment.status`
Deploy prod: `make ENV=prod deployment.apply`