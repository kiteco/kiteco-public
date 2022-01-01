.PHONY: build push clean

build: server
	docker build -t kiteco/kite-server-metadata .

push:
ifeq ($(TAG),)
	$(error Must set TAG to push)
endif
	docker tag kiteco/kite-server-metadata kiteco/kite-server-metadata:$(TAG)
	docker push kiteco/kite-server-metadata:$(TAG)

clean:
	rm -rf server

service/service_grpc.pb.go: service.proto
	protoc service.proto --go-grpc_out=.

server: service/service_grpc.pb.go server.go
	env GOOS=linux go build server.go
