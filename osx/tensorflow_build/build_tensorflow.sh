#!/bin/bash
set -e

brew install python
pip3 install -U --user pip six numpy wheel setuptools mock 'future>=0.17.1'
pip3 install -U --user keras_applications --no-deps
pip3 install -U --user keras_preprocessing --no-deps

if [ ! -f "./bazel-0.26.1-installer-darwin-x86_64.sh" ]; then
  curl -LO https://github.com/bazelbuild/bazel/releases/download/0.26.1/bazel-0.26.1-installer-darwin-x86_64.sh
  chmod +x bazel-0.26.1-installer-darwin-x86_64.sh
  ./bazel-0.26.1-installer-darwin-x86_64.sh --user
fi
export PATH="$PATH:$HOME/bin:$HOME/Library/Python/3.7/bin"

[ ! -d tensorflow ] && git clone -b r1.15 https://github.com/tensorflow/tensorflow.git tensorflow

cd tensorflow
git pull

export CC_OPT_FLAGS="-march=x86-64 -mavx -msse4.2 -mavx2 -mfma"
bazel clean --expunge
yes "" | ./configure

bazel build --jobs 6 --config=opt --cxxopt=-D_GLIBCXX_USE_CXX11_ABI=0 --config=noaws --config=nogcp --config=nohdfs --config=noignite --config=nokafka --config=nonccl //tensorflow/tools/lib_package:libtensorflow.tar.gz
cp bazel-bin/tensorflow/tools/lib_package/libtensorflow.tar.gz /build/libtensorflow.tar.gz