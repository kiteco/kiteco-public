.PHONY: build push clean

build: tfserving-mux
	docker build -t kiteco/kite-server-models-stats-proxy .

push:
ifeq ($(TAG),)
	$(error Must set TAG to push)
endif
	docker tag kiteco/kite-server-models-stats-proxy kiteco/kite-server-models-stats-proxy:$(TAG)
	docker push kiteco/kite-server-models-stats-proxy:$(TAG)

clean:
	rm -rf tfserving-mux

tfserving-mux:
	env GOOS=linux go build github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving/cmds/tfserving-mux/
