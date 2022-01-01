DEV_TAG=airflow_monetizable_dev:latest
KITECO=$(PWD)/../../..
DOCKER_RUN_CMD=docker run --rm -it -e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) -e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) -v $(KITECO):/go/src/github.com/kiteco/kiteco $(DEV_TAG)

ECR_REPO_URL=XXXXXXX.dkr.ecr.us-west-1.amazonaws.com
ECR_PACKAGE_NAME=kite-airflow-monetizable
TAG=$(shell git rev-parse --short HEAD)

KITECO=$${PWD%/kiteco/**}/kiteco
CWD_RELATIVE=$${PWD\#/**/kiteco}
GO_IMAGE=golang:1.15.3-buster

dev.build: build/monetizable
	docker build --build-arg BUILD_HASH=$(TAG) -t $(DEV_TAG) .

dev.shell:
	@exec $(DOCKER_RUN_CMD) /bin/bash

docker.login:
	aws ecr get-login-password --region us-west-1 | docker login --username AWS --password-stdin $(ECR_REPO_URL)

docker.build: build/monetizable
	docker build --build-arg BUILD_HASH=$(TAG) -t $(ECR_PACKAGE_NAME):$(TAG) .

docker.tag:
	docker tag $(ECR_PACKAGE_NAME):$(TAG) $(ECR_REPO_URL)/$(ECR_PACKAGE_NAME):$(TAG)

docker.push:
	docker push $(ECR_REPO_URL)/$(ECR_PACKAGE_NAME):$(TAG)

docker.all: docker.login docker.build docker.tag docker.push

build/monetizable: build main.go
	docker run --rm -e "GOPRIVATE=github.com/kiteco/*" \
		-v $(KITECO):/go/src/github.com/kiteco/kiteco \
		-v $(PWD)/build:/build \
		-w /go/src/github.com/kiteco/$(CWD_RELATIVE) \
		$(GO_IMAGE) go build -o /build/monetizable .

build:
	mkdir -p build
