#!/usr/bin/env bash
# this script builds tcmalloc in a docker container

set -e

BUILD_DIR="$PWD/build"
mkdir -p "$BUILD_DIR"

cat > "$BUILD_DIR/run.sh" << 'EOF'
#!/bin/bash
set -e
apt update && apt install -y git make build-essential autoconf libtool
git clone https://github.com/gperftools/gperftools.git gperftools
cd gperftools
./autogen.sh
./configure --enable-sized-delete --enable-minimal
make

cp ".libs/libtcmalloc_minimal.a" ".libs/libtcmalloc_minimal_debug.so" /build/
chmod -R a+r+w+X /build
EOF

chmod +x "$BUILD_DIR/run.sh"

docker run -t \
    --mount "src=$BUILD_DIR,target=/build,type=bind" \
    ubuntu:bionic /build/run.sh

cp "./build/libtcmalloc_minimal.a" "./build/libtcmalloc_minimal_debug.so" .
rm -rf build
echo "Static library for linking into kited is at $PWD/libtcmalloc_minimal.a"
echo "Shared library for CI is at $PWD/libtcmalloc_minimal_debug.so."