#!/bin/bash

git clone -b r1.15 https://github.com/tensorflow/serving
git clone --depth 1 -b r1.15 https://github.com/tensorflow/tensorflow tf_repo

go build ./cmds/genproto
./genproto
rm genproto

sed -i -e 's/github.com\/tensorflow\/tensorflow\/tensorflow\/go\/core/github.com\/kiteco\/kiteco\/kite-golib\/protobuf\/tensorflow\/core/g' tensorflow/serving/*.go
sed -i -e 's/github.com\/tensorflow\/tensorflow\/tensorflow\/go\/core/github.com\/kiteco\/kiteco\/kite-golib\/protobuf\/tensorflow\/core/g' tensorflow/core/protobuf/*.go
sed -i -e 's/github.com\/tensorflow\/tensorflow\/tensorflow\/go\/core/github.com\/kiteco\/kiteco\/kite-golib\/protobuf\/tensorflow\/core/g' tensorflow/core/example/*.go

# This script execute more or less the steps provided with the genproto.go file :
#	git clone -b r1.15 https://github.com/tensorflow/tensorflow.git
#	git clone -b r1.14 https://github.com/tensorflow/serving.git # We now use serving from r1.15
#	go run protoc.go
#	go mod edit -replace=github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving/tensorflow/tensorflow/go/core=./proto/tensorflow/core # We do the mod manipulation with sed instead of go mod
#	cd proto/tensorflow/core && go mod init github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving/tensorflow/tensorflow/go/core && cd -
#	go build ./proto/tensorflow/serving


