.PHONY: clean

KITECO=$${PWD%/kiteco/**}/kiteco
CWD_RELATIVE=$${PWD\#/**/kiteco}
GO_IMAGE=golang:1.15.3-buster

build/user-mux: clean build main.go
	docker run --rm -e "GOPRIVATE=github.com/kiteco/*" \
		-v $(KITECO):/go/src/github.com/kiteco/kiteco \
		-v $(PWD)/build:/build \
		-w /go/src/github.com/kiteco/$(CWD_RELATIVE) \
		$(GO_IMAGE) go build -o /build/user-mux .

build:
	mkdir -p build

clean:
	rm -rf build
