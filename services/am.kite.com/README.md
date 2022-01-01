# am.kite.com

## Dependencies:
 * Docker: https://docs.docker.com/get-docker/
 * gcloud cli: https://cloud.google.com/sdk/docs/quickstart
 * kubectl: `gcloud components install kubectl`
 * ytt: `brew tap k14s/tap; brew install ytt`

## Running locally
From this directory:
```sh
source local.sh
make
```

## Update Docker Images
To update images and push to [Container Registry](https://cloud.google.com/container-registry):
```sh
make docker.all
```

## Setup Infrastructure
Terraform is used to setup service account permissions in GCP and AWS.

```sh
make deployment.setup # for staging
make deployment.setup ENV=prod # for prod
```

## Deployment
Make sure your host has the Kite AWS VPN enabled.

- Build and publish Docker images: `make docker.all`

- Deploy staging: `make deployment.apply`

- Check status: `make deployment.status`

- Deploy prod: `make ENV=prod deployment.apply`

The required namespaces should already be set up, but
you can run `make namespace` to set up staging and `make namespace ENV=prod` for prod.
