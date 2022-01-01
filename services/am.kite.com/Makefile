PROJECT=kite-dev-XXXXXXX
GCR_REPO_URL=gcr.io/$(PROJECT)

PACKAGE_PREFIX=am-kite-com
PACKAGES=user-mux user-node

CLUSTER=prod-us-west-1
NAMESPACE=am-kite-com-$(ENV)
REGION=us-west1
CONTEXT=gke_$(PROJECT)_$(REGION)_$(CLUSTER)

TAG=$(shell git rev-parse --short HEAD)
ENV=staging

BUILD_DIR=build
K8S_BUILD_DIR=$(BUILD_DIR)/k8s/$(TAG)/$(ENV)
K8S_SRC_DIR=deployment/k8s
K8S_SRC_FILES=$(wildcard $(K8S_SRC_DIR)/*)

DOCKER_TERRAFORM_ARGS=-w /opt/terraform -v $(PWD)/deployment/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud
AWS_SECRET_CMD=aws --region=us-west-1 --output text --query 'SecretString' secretsmanager get-secret-value --secret-id
KUBECTL_SECRET_CMD=kubectl -n $(NAMESPACE) create secret generic --dry-run=client -o yaml

COMMUNITY_DB_DRIVER=postgres

dev.start:
	cd user-mux && $(MAKE)
	cd user-node && $(MAKE)
	docker-compose build
	docker-compose up -d

dev.build:
	cd user-mux && $(MAKE)
	cd user-node && $(MAKE)
	docker-compose restart user-mux
	docker-compose restart user-node

docker.login:
	gcloud auth configure-docker gcr.io

docker.build:
	cd user-mux && $(MAKE)
	cd user-node && $(MAKE)
	$(foreach pkg,$(PACKAGES),docker build -t $(PACKAGE_PREFIX)-$(pkg):$(TAG) ./$(pkg);)

docker.tag:
	$(foreach pkg,$(PACKAGES),docker tag $(PACKAGE_PREFIX)-$(pkg):$(TAG) $(GCR_REPO_URL)/$(PACKAGE_PREFIX)-$(pkg):$(TAG);)

docker.push:
	$(foreach pkg,$(PACKAGES),docker push $(GCR_REPO_URL)/$(PACKAGE_PREFIX)-$(pkg):$(TAG);)

docker.all: docker.login docker.build docker.tag docker.push

$(K8S_BUILD_DIR)/resources.yaml: $(K8S_SRC_FILES)
	mkdir -p $(K8S_BUILD_DIR)
	ytt --output-files $(K8S_BUILD_DIR) -f $(K8S_SRC_DIR)/resources.yaml -f $(K8S_SRC_DIR)/data.yaml -f $(K8S_SRC_DIR)/data-$(ENV).yaml -v tag=$(TAG) -v namespace=$(NAMESPACE) -v project=$(PROJECT)

# Output K8s Config
k8s.config: $(K8S_BUILD_DIR)/resources.yaml

terraform.shell:
	docker run $(DOCKER_TERRAFORM_ARGS) --entrypoint sh -it hashicorp/terraform:latest

# Setup infra that doesn't change deployment-to-deployment
deployment.setup:
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest init
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace new $(ENV) || \
		docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace select $(ENV)
	docker run $(DOCKER_TERRAFORM_ARGS) -it hashicorp/terraform:latest apply

# Stage a new deployment
deployment.apply: clean .license .kubectl $(K8S_BUILD_DIR)/resources.yaml
	@echo "Deploying "$(TAG)" to "$(ENV)

	$(KUBECTL_SECRET_CMD) am-kite-com-user-mux \
		--from-literal AWS_REGION='$(shell aws configure get region)' \
		--from-literal COMMUNITY_DB_URI='$(shell $(AWS_SECRET_CMD) COMMUNITY_DB_URI)' \
		--from-literal COMMUNITY_DB_DRIVER=$(COMMUNITY_DB_DRIVER) | \
		kubectl apply -f -

	$(KUBECTL_SECRET_CMD) am-kite-com-user-node \
		--from-literal AWS_REGION='$(shell aws configure get region)' \
		--from-literal COMMUNITY_DB_URI='$(shell $(AWS_SECRET_CMD) COMMUNITY_DB_URI)' \
		--from-literal COMMUNITY_DB_DRIVER=$(COMMUNITY_DB_DRIVER) \
		--from-literal STRIPE_SECRET='$(shell $(AWS_SECRET_CMD) STRIPE_SECRET)' \
		--from-literal STRIPE_WEBHOOK_SECRET='$(shell $(AWS_SECRET_CMD) STRIPE_WEBHOOK_SECRET)' \
		--from-literal OCTOBAT_SECRET='$(shell $(AWS_SECRET_CMD) OCTOBAT_SECRET)' \
		--from-literal OCTOBAT_PUBLISHABLE='$(shell $(AWS_SECRET_CMD) OCTOBAT_PUBLISHABLE)' \
		--from-literal MIXPANEL_SECRET='$(shell $(AWS_SECRET_CMD) MIXPANEL_SECRET)' \
		--from-literal DELIGHTED_SECRET='$(shell $(AWS_SECRET_CMD) DELIGHTED_SECRET)' \
		--from-literal QUICK_EMAIL_TOKEN='$(shell $(AWS_SECRET_CMD) QUICK_EMAIL_TOKEN)' \
		--from-file LICENSE_RSA_KEY=license | \
		kubectl apply -f -

	rm license

	kubectl apply -f $(K8S_BUILD_DIR)/resources.yaml

deployment.status:
	@echo "=====DEPLOYMENTS====="
	@kubectl -n $(NAMESPACE) --show-labels=true get deployments
	@echo "=====PODS====="
	@kubectl -n $(NAMESPACE) --show-labels=true get pods

.license:
	$(AWS_SECRET_CMD) LICENSE_RSA_KEY >> license

.kubectl:
	@test $(shell kubectl config current-context) == $(CONTEXT) || (echo "Initialize kubectl by running \"make kubectl\"" && false)

kubectl:
	gcloud container clusters get-credentials $(CLUSTER) --region $(REGION) --project $(PROJECT)
	kubectl config use-context gke_$(PROJECT)_$(REGION)_$(CLUSTER)

namespace:
	kubectl create namespace $(NAMESPACE)

build:
	mkdir -p build

clean:
	rm -rf build/*
