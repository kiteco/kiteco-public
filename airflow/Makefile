ECR_REPO_URL=XXXXXXX.dkr.ecr.us-west-1.amazonaws.com
ECR_PACKAGE_NAME=kite-airflow
TAG=$(shell git rev-parse --short HEAD)

docker.login:
	aws ecr get-login-password --region us-west-1 | docker login --username AWS --password-stdin $(ECR_REPO_URL)

docker.build:
	docker build -t $(ECR_PACKAGE_NAME):$(TAG) ../ -f Dockerfile
	cd containers/monetizable && make TAG=$(TAG) docker.build

docker.tag:
	docker tag $(ECR_PACKAGE_NAME):$(TAG) $(ECR_REPO_URL)/$(ECR_PACKAGE_NAME):$(TAG)
	cd containers/monetizable && make TAG=$(TAG) docker.tag

docker.push:
	docker push $(ECR_REPO_URL)/$(ECR_PACKAGE_NAME):$(TAG)
	cd containers/monetizable && make TAG=$(TAG) docker.push

docker.all: docker.login docker.build docker.tag docker.push

deployment.apply:
	docker run --rm -w /opt/terraform -v $(PWD)/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud -it ljfranklin/terraform-resource:latest terraform apply -var tag=$(TAG)

deployment.list:
	@aws ecs list-tasks --cluster airflow | jq -r '.taskArns[]' | xargs aws ecs describe-tasks --cluster airflow --tasks | jq -r '.tasks[] | {"group": .group, "container": .containers[]} | (.group + "    " + .container.name + "    " + (.container.image | capture(".*:(?<tag>[0-9a-f]+)$$") | .tag) + "    " + .container.lastStatus)'

deployment.shell:
	exec docker run --rm -w /opt/terraform -v $(PWD)/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud -it ljfranklin/terraform-resource:latest /bin/bash

python.lint:
	flake8 kite_airflow