# Service release.kite.com

## Dependencies

Packages:
 * ytt
 * gcloud CLI

## Deploying

Build and push the container image: `make docker.all`

Change the staging deployment: `make deployment.apply`

See your new pods come up: `make deployment.status`

Test the new deployment on release-staging.kite.com.

Change the production deployment: `make ENV=prod deployment.apply`

See your new pods come up: `make deployment.status`

Remove staging deployment: `make deployment.cleanup`

