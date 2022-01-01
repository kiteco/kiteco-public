.PHONY: build schemas

PROJECT=kite-dev-XXXXXXX
GCR_REPO_URL=gcr.io/$(PROJECT)

PACKAGE_PREFIX=rc-kite-com
PACKAGES=nginx convcohort metricbeat

CLUSTER=prod-us-west-1
NAMESPACE=rc-kite-com-$(ENV)
REGION=us-west1
CONTEXT=gke_$(PROJECT)_$(REGION)_$(CLUSTER)

TAG=$(shell git rev-parse --short HEAD)
ENV=staging

BUILD_DIR=build
K8S_BUILD_DIR=$(BUILD_DIR)/k8s/$(TAG)/$(ENV)
K8S_SRC_DIR=deployment/k8s
K8S_SRC_FILES=$(wildcard $(K8S_SRC_DIR)/*)

AWS_SECRET_CMD=aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id
KUBECTL_SECRET_CMD=kubectl -n $(NAMESPACE) create secret generic --dry-run=client -o yaml
DOCKER_TERRAFORM_ARGS=-w /opt/terraform -v $(PWD)/deployment/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud

dev.start:
	cd convcohort && $(MAKE)
	docker-compose up -d

dev.build:
	cd convcohort && $(MAKE)
	docker-compose restart webserver

docker.login:
	gcloud auth configure-docker gcr.io

docker.build:
	cd convcohort && $(MAKE) | true
	$(foreach pkg,$(PACKAGES),docker build -t $(PACKAGE_PREFIX)-$(pkg):$(TAG) ./$(pkg);)

docker.tag:
	$(foreach pkg,$(PACKAGES),docker tag $(PACKAGE_PREFIX)-$(pkg):$(TAG) $(GCR_REPO_URL)/$(PACKAGE_PREFIX)-$(pkg):$(TAG);)

docker.push:
	$(foreach pkg,$(PACKAGES),docker push $(GCR_REPO_URL)/$(PACKAGE_PREFIX)-$(pkg):$(TAG);)

docker.all: docker.login docker.build docker.tag docker.push

$(K8S_BUILD_DIR)/resources.yaml: $(K8S_SRC_FILES)
	mkdir -p $(K8S_BUILD_DIR)
	ytt --output-files $(K8S_BUILD_DIR) -f $(K8S_SRC_DIR)/resources.yaml -f $(K8S_SRC_DIR)/data.yaml -f $(K8S_SRC_DIR)/data-$(ENV).yaml -v tag=$(TAG) -v namespace=$(NAMESPACE) -v project=$(PROJECT)

# Setup infra that doesn't change deployment-to-deployment
deployment.setup:
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest init
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace new $(ENV) || \
		docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace select $(ENV)
	docker run $(DOCKER_TERRAFORM_ARGS) -it hashicorp/terraform:latest apply

# Stage a new deployment
deployment.apply: .kubectl $(K8S_BUILD_DIR)/resources.yaml
	@echo "Deploying "$(TAG)" to "$(ENV)

	$(KUBECTL_SECRET_CMD) rc-kite-com-convcohort \
		--from-literal CUSTOMER_IO_API_KEY='$(shell $(AWS_SECRET_CMD) CUSTOMER_IO_API_KEY)' | \
		kubectl apply -f -

	$(KUBECTL_SECRET_CMD) nginx \
		--from-literal htpasswd='$(shell $(AWS_SECRET_CMD) services/rc_kite_com/htpasswd)' | \
		kubectl apply -f -

	$(KUBECTL_SECRET_CMD) rc-kite-com-metricbeat \
		--from-literal cloud_id='$(shell $(AWS_SECRET_CMD) beats_elastic_cloud_id)' \
		--from-literal cloud_auth='$(shell $(AWS_SECRET_CMD) beats_elastic_auth_str)' | \
		kubectl apply -f -

	kubectl apply -f $(K8S_BUILD_DIR)/resources.yaml

deployment.status:
	@echo "=====DEPLOYMENTS====="
	@kubectl -n $(NAMESPACE) --show-labels=true get deployments
	@echo "=====PODS====="
	@kubectl -n $(NAMESPACE) --show-labels=true get pods

terraform.shell:
	docker run $(DOCKER_TERRAFORM_ARGS) --entrypoint sh -it hashicorp/terraform:latest

.kubectl:
	@test $(shell kubectl config current-context) == $(CONTEXT) || (echo "Initialize kubectl by running \"make kubectl\"" && false)

kubectl:
	gcloud container clusters get-credentials $(CLUSTER) --region $(REGION) --project $(PROJECT)
	kubectl config use-context gke_$(PROJECT)_$(REGION)_$(CLUSTER)

build:
	mkdir -p build

clean:
	rm -rf build/*
