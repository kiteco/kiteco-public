GOPATH ?= ${HOME}/go
ifeq ($(origin GOPATH), undefined)
	GOPATH := ${HOME}/go
endif
export GOPATH
KITECO := github.com/kiteco/kiteco/
KITEPATH := ${GOPATH}/src/${KITECO}

.PHONY: setup build all dockerimages explore
.PRECIOUS: build/explored/%.json build/rawgraph/%.valid.json build/dockerimages/%.image

all:
	@cat ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/README.md

build: build/manifest.json

setup: ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/pip-packages ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/special/ | build/bin/dockertools
	mkdir -p dockerfiles/all
	mkdir -p dockerfiles/build
	./build/bin/dockertools files ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/pip-packages dockerfiles/all
	cp ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/special/* dockerfiles/all


# - Binaries

build/bin/explore: ${KITEPATH}/local-pipelines/python-import-exploration/explore
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/local-pipelines/python-import-exploration/explore

build/bin/dockertools: ${KITEPATH}/local-pipelines/python-import-exploration/dockertools
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/local-pipelines/python-import-exploration/dockertools

build/bin/rawgraph: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/rawgraph
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/rawgraph

build/bin/distidx: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/distidx
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/distidx

build/bin/compat: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/compat
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/compat

build/bin/symgraphs: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/symgraphs
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/symgraphs

build/bin/argspecs: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/argspecs
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/argspecs

build/bin/kwargs: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/kwargs
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/kwargs

build/bin/docs: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/docs
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/docs

build/bin/popularsignatures: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/popularsignatures
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/popularsignatures

build/bin/symbolcounts: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/symbolcounts
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/symbolcounts

build/bin/typeshed: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/typeshed
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/typeshed

build/bin/manifest: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/manifest
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/manifest

build/bin/analyze: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/analyze
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/analyze

build/bin/toplevel: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/toplevel
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/toplevel

build/bin/returntypes: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/returntypes
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/returntypes

build/bin/docs-returntypes: ${KITEPATH}/kite-go/lang/python/pythonresource/cmd/build/docs-returntypes
	mkdir -p $$(dirname $@)
	go build -o $@ ${KITECO}/kite-go/lang/python/pythonresource/cmd/build/docs-returntypes



# - Docker & Exploration
DOCKERFILES := $(wildcard dockerfiles/build/*)

# we use files as build markers for the corresponding docker images
build/dockerimages/BASE:
	mkdir -p $$(dirname $@)
	# TODO this should ideally go in a go binary
	docker build --force-rm --tag kiteco/python-import-exploration ${KITEPATH}/kite-python/kite_pkgexploration/ >$@.log 2>&1
	touch $@
build/dockerimages/%.image: dockerfiles/build/% build/dockerimages/BASE | build/bin/dockertools
	mkdir -p $$(dirname $@)
	./build/bin/dockertools buildimage $< >$@.log 2>&1
	touch $@
DOCKERIMAGES := $(DOCKERFILES:dockerfiles/build/%=build/dockerimages/%.image)
dockerimages: $(DOCKERIMAGES)
	true

build/explored/%.json: build/dockerimages/%.image | build/bin/explore
	mkdir -p $$(dirname $@)
	./build/bin/explore package --out $$(pwd)/$$(dirname $@) $* >$@.log 2>&1
EXPLORED := $(DOCKERIMAGES:build/dockerimages/%.image=build/explored/%.json)
explore: $(EXPLORE)
	true


# - Build

build/rawgraph/%.valid.json: build/explored/%.json | build/bin/rawgraph
	mkdir -p $$(dirname $@)
	./build/bin/rawgraph $< $@ >$@.log 2>&1
RAWGRAPH := $(EXPLORED:build/explored/%.json=build/rawgraph/%.valid.json)
#RAWGRAPH := $(wildcard build/rawgraph/*.valid.json)

build/stubless/%.json: build/rawgraph/%.valid.json | build/bin/symgraphs
	mkdir -p $$(dirname $@)
	./build/bin/symgraphs $@ build/data/ $< >$@.log 2>&1
STUBLESS := $(RAWGRAPH:build/rawgraph/%.valid.json=build/stubless/%.json)
build/stubless.json: $(STUBLESS) | build/bin/manifest
	mkdir -p $$(dirname $@)
	./build/bin/manifest merge -o $@ $^

# build all symgraphs in one call, because the overhead of loading the stubless symgraph is high
build/symgraphs.json: build/analyzed.json.gz build/stubless.json build/distidx.json $(RAWGRAPH) | build/bin/manifest
	mkdir -p $$(dirname $@)
	./build/bin/symgraphs --analyzed build/analyzed.json.gz --graph build/stubless.json --distidx build/distidx.json $@ build/data/ $(RAWGRAPH) >$@.log 2>&1

build/argspecs.json: build/typeshed build/symgraphs.json build/distidx.json $(RAWGRAPH) | build/bin/argspecs
	mkdir -p $$(dirname $@)
	./build/bin/argspecs --typeshed build/typeshed --graph build/symgraphs.json --distidx build/distidx.json $@ build/data/ $(RAWGRAPH) >$@.log 2>&1

build/typeshed:
	mkdir -p $$(dirname $@)
	git clone git@github.com:kiteco/typeshed.git $@

build/docs.json: build/compat.json.gz build/symgraphs.json build/distidx.json $(RAWGRAPH) | build/bin/docs
	mkdir -p $$(dirname $@)
	./build/bin/docs --compat build/compat.json.gz --graph build/symgraphs.json --distidx build/distidx.json $@ build/data/ $(RAWGRAPH) >$@.log 2>&1

build/kwargs.json: build/compat.json.gz build/symgraphs.json build/distidx.json | build/bin/kwargs
	mkdir -p $$(dirname $@)
	./build/bin/kwargs --compat build/compat.json.gz --graph build/symgraphs.json --distidx build/distidx.json $@ build/data/ >$@.log 2>&1

# symbolcounts depend on local-pipelines/python-symbol-to-hashes-index
build/symbolcounts.json: build/symgraphs.json build/distidx.json | build/bin/symbolcounts
	mkdir -p $$(dirname $@)
	./build/bin/symbolcounts --graph build/symgraphs.json --distidx build/distidx.json $@ build/data/ >$@.log 2>&1

build/docs_returntypes.json: $(RAWGRAPH) | build/bin/docs-returntypes
	mkdir -p $$(dirname $@)
	./build/bin/docs-returntypes $@ $^ >$@.log 2>&1

build/returntypes.json: build/symgraphs.json build/distidx.json build/analyzed.json.gz build/docs_returntypes.json | build/bin/returntypes
	mkdir -p $$(dirname $@)
	./build/bin/returntypes --graph build/symgraphs.json --distidx build/distidx.json --analyzed build/analyzed.json.gz --docs build/docs_returntypes.json $@ build/data/ >$@.log 2>&1

build/analyzed.json.gz: build/stubless.json build/distidx.json build/typeshed ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/pip-packages ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/special-pip-packages | build/bin/analyze
	mkdir -p $$(dirname $@)
	./build/bin/analyze --graph build/stubless.json --distidx build/distidx.json --typeshed build/typeshed/ -o $@ ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/pip-packages ${KITEPATH}/local-pipelines/python-import-exploration/packagelist/packagelists/special-pip-packages >$@.log 2>&1

build/compat.json.gz: build/symgraphs.json build/distidx.json | build/bin/compat
	mkdir -p $$(dirname $@)
	./build/bin/compat --graph build/symgraphs.json --distidx build/distidx.json $@ >$@.log 2>&1

# Popular signatures generation, more steps as it rely on some python code
# Build a new manifest with argspec and the symbol graphs
build/argspec_graph_manifest.json: build/symgraphs.json build/argspecs.json | build/bin/manifest
	mkdir -p $$(dirname $@)
	./build/bin/manifest merge -o build/argspec_graph_manifest.json build/symgraphs.json build/argspecs.json

# Generate the final resource bundle of signatures
build/popularsignatures.json: build/argspec_graph_manifest.json | build/bin/popularsignatures
	mkdir -p $$(dirname $@)
	./build/bin/popularsignatures --manifest build/argspec_graph_manifest.json --resources-output build/data --manifest-output $@ >$@.log 2>&1


# - Deployables

build/distidx.json: build/stubless.json | build/bin/distidx # use the stub-less merged graph to avoid circular dependencies
	mkdir -p $$(dirname $@)
	./build/bin/distidx --graph $< $@ >$@.log 2>&1

build/manifest_lean.json: build/symgraphs.json build/argspecs.json build/docs.json build/kwargs.json build/returntypes.json build/symbolcounts.json | build/bin/manifest
	mkdir -p $$(dirname $@)
	./build/bin/manifest merge -o $@ $^

build/manifest.json: build/manifest_lean.json build/popularsignatures.json | build/bin/manifest
	mkdir -p $$(dirname $@)
	./build/bin/manifest merge -o $@ $^

# this technically only needs symgraph, docs, and symcounts
build/toplevel.gob.gz: build/manifest_lean.json build/distidx.json | build/bin/toplevel
	mkdir -p $$(dirname $@)
	./build/bin/toplevel --manifest build/manifest_lean.json --distidx build/distidx.json $@ >$@.log 2>&1

# - More Phonies

.PHONY: images explore rawgraphs
images: ${DOCKERIMAGES}
explore: ${EXPLORED}
rawgraphs: ${RAWGRAPH}
