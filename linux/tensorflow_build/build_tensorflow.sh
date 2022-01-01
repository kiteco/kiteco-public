#!/usr/bin/env bash
# this script builds tensorflow with AVX, AVX2, FMA, SSE4.2

BUILD_DIR="$PWD/build"
mkdir -p "$BUILD_DIR"

SRC_DIR="$PWD/tensorflow_src"
mkdir -p "$SRC_DIR"

cat > "$BUILD_DIR/run.sh" << 'EOF'
#!/bin/bash
set -e
export USER=dummy HOME=/build/.home
[ ! -d "/tensorflow_src/tensorflow" ] && git clone -b r1.15 https://github.com/tensorflow/tensorflow.git /tensorflow_src/tensorflow
cd /tensorflow_src/tensorflow
git pull
BAZEL_OPTS="--jobs 6 --config=opt --cxxopt=-D_GLIBCXX_USE_CXX11_ABI=0 --config=noaws --config=nogcp --config=nohdfs --config=noignite --config=nokafka --config=nonccl"
export CC_OPT_FLAGS="-march=x86-64 -mavx -msse4.2 -mavx2 -mfma"
bazel clean --expunge
yes "" | ./configure

bazel build ${BAZEL_OPTS} //tensorflow/tools/lib_package:libtensorflow.tar.gz
cp bazel-bin/tensorflow/tools/lib_package/libtensorflow.tar.gz /build/libtensorflow.tar.gz
EOF

chmod +x "$BUILD_DIR/run.sh"

URL="https://raw.githubusercontent.com/tensorflow/tensorflow/master/tensorflow/tools/dockerfiles/dockerfiles/devel-cpu.Dockerfile"
wget -q -O - "$URL" | docker build -t kite:tensorflow_cpu \
    --build-arg USE_PYTHON_3_NOT_2=1 \
    --build-arg CHECKOUT_TF_SRC=0 \
    --build-arg BAZEL_VERSION=0.26.1 \
    -f - .

docker run -u $(id -u):$(id -g) -t \
    --mount "src=$BUILD_DIR,target=/build,type=bind" \
    --mount "src=$SRC_DIR,target=/tensorflow_src,type=bind" \
    kite:tensorflow_cpu /build/run.sh