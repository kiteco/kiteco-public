## Generate cloud credentials

gcloud iam service-accounts keys create $(HOME)/.gcloud/sa-credentials.json --iam-account <ACCOUNT NAME>@kite-prod-XXXXXXX.iam.gserviceaccount.com

## Run in Docker

docker pull ljfranklin/terraform-resource:latest
docker run -v $PWD:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -it ljfranklin/terraform-resource:latest /bin/bash