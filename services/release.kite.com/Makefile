.PHONY: build schemas

PROJECT=kite-dev-XXXXXXX
ECR_REPO_URL=gcr.io/$(PROJECT)
KITECO=$${PWD%/kiteco/**}/kiteco
PACKAGE_NAME=release-kite-com

CLUSTER=prod-us-west-1
NAMESPACE=release-kite-com-$(ENV)
NAME=release-kite-com
REGION=us-west1
CONTEXT=gke_$(PROJECT)_$(REGION)_$(CLUSTER)

TAG=$(shell git rev-parse --short HEAD)
ENV=staging

BUILD_DIR=build
K8S_BUILD_DIR=$(BUILD_DIR)/k8s/$(TAG)
K8S_SRC_DIR=deployment/k8s
K8S_SRC_FILES=$(wildcard $(K8S_SRC_DIR)/*)

SECRETS=RELEASE_DB_URI

STAGING_TAGS=$(shell kubectl -n $(NAMESPACE) get pods --selector=app=webserver,env=staging -o jsonpath='{.items[*].metadata.labels.tag}' | sort | uniq | xargs echo -n)

AWS_SECRET_CMD=aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id
KUBECTL_SECRET_CMD=kubectl -n $(NAMESPACE) create secret generic --dry-run=client -o yaml
DOCKER_TERRAFORM_ARGS=-w /opt/terraform -v $(PWD)/deployment/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud

docker.login:
	gcloud auth configure-docker gcr.io

docker.build:
	docker build -t $(PACKAGE_NAME):$(TAG) -f Dockerfile ../../

docker.tag:
	docker tag $(PACKAGE_NAME):$(TAG) $(ECR_REPO_URL)/$(PACKAGE_NAME):$(TAG)

docker.push:
	docker push $(ECR_REPO_URL)/$(PACKAGE_NAME):$(TAG)

docker.all: docker.login docker.build docker.tag docker.push

.venv:
	virtualenv .venv
	.venv/bin/pip install -e $(KITECO)/kite-python/metrics

$(K8S_BUILD_DIR)/resources.yaml: $(K8S_SRC_FILES)
	mkdir -p $(K8S_BUILD_DIR)
	ytt --output-files $(K8S_BUILD_DIR) -f $(K8S_SRC_DIR)/resources.yaml -f $(K8S_SRC_DIR)/data.yaml -f $(K8S_SRC_DIR)/data-$(ENV).yaml -v tag=$(TAG) -v namespace=$(NAMESPACE) -v project=$(PROJECT)

# Setup infra that doesn't change deployment-to-deployment
deployment.setup:
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest init
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace new $(ENV)
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace select $(ENV)
	docker run $(DOCKER_TERRAFORM_ARGS) -it hashicorp/terraform:latest apply

deployment.apply: .kubectl $(K8S_BUILD_DIR)/resources.yaml
	@echo "Deploying "$(TAG)" to "$(ENV)

	$(KUBECTL_SECRET_CMD) release-db-uri \
		--from-literal RELEASE_DB_URI='$(shell $(AWS_SECRET_CMD) RELEASE_DB_URI_GCP_SQL)' | \
		kubectl apply -f -

	kubectl apply -f $(K8S_BUILD_DIR)/resources.yaml

deployment.status:
	@echo "=====DEPLOYMENTS====="
	@kubectl -n $(NAMESPACE) --show-labels=true get deployments
	@echo "=====PODS====="
	@kubectl -n $(NAMESPACE) --show-labels=true get pods

deployment.cleanup:
	kubectl -n $(NAMESPACE) delete deployment release-kite-com-staging

terraform.shell:
	exec docker run --rm -w /opt/terraform -v $(PWD)/deployment/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud -it ljfranklin/terraform-resource:latest /bin/bash

.kubectl:
	@test $(shell kubectl config current-context) == $(CONTEXT) || (echo "Initialize kubectl by running \"make kubectl\"" && false)

kubectl:
	gcloud container clusters get-credentials $(CLUSTER) --region $(REGION) --project $(PROJECT)
	kubectl config use-context gke_$(PROJECT)_$(REGION)_$(CLUSTER)

