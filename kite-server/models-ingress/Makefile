.PHONY: build push

build:
	docker build -t kiteco/kite-server-models-ingress .

push:
ifeq ($(TAG),)
	$(error Must set TAG to push)
endif
	docker tag kiteco/kite-server-models-ingress kiteco/kite-server-models-ingress:$(TAG)
	docker push kiteco/kite-server-models-ingress:$(TAG)
