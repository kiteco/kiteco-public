#!/bin/bash

set -ex

# NVIDIA drivers
sudo add-apt-repository -y ppa:graphics-drivers/ppa
sudo apt-get update
sudo apt-get install -y nvidia-headless-440 nvidia-utils-440

# Docker
sudo apt-get install -y docker.io

# nvidia-container-runtime repository: https://nvidia.github.io/nvidia-container-runtime/
curl -s -L https://nvidia.github.io/nvidia-container-runtime/gpgkey | \
  sudo apt-key add -
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-container-runtime/$distribution/nvidia-container-runtime.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-runtime.list
sudo apt-get update

# nvidia-container-runtime & associated configuration
sudo apt-get install -y nvidia-container-runtime
sudo tee /etc/docker/daemon.json <<EOF
{
    "runtimes": {
        "nvidia": {
            "path": "/usr/bin/nvidia-container-runtime",
            "runtimeArgs": []
        }
    },
    "default-runtime": "nvidia"
}
EOF

sudo apt-get install -y python3-pip

# configure nvidia-container-runtime to use DOCKER_RESOURCE_NVIDIA-GPU
# as the Docker swarm resource environment variable to set GPUs.
# This matches what nvidia-docker-autoconf does below.
sudo python3 -m pip install toml
sudo python3 - <<EOF
import toml
confPath = '/etc/nvidia-container-runtime/config.toml'
conf = toml.load(confPath)
conf['swarm-resource'] = 'DOCKER_RESOURCE_NVIDIA-GPU'
with open(confPath, 'w') as f:
  toml.dump(conf, f)
EOF

# autoconfigure GPUs as node-generic-resources when Docker starts;
# GPUs can be added/removed after the image is built.
# nvidia-docker-autoconf is a source distribution published by Kite.
sudo python3 -m pip install nvidia-docker-autoconf
sudo tee /etc/systemd/system/nvidia-docker-autoconf.service <<EOF
[Unit]
Description=autoconfigure GPU clocks & GPUs as node-generic-resources when Docker starts

[Service]
Type=oneshot
RemainAfterExit=no
ExecStart=/usr/bin/env nvidia-docker-autoconf

[Install]
WantedBy=docker.service
EOF
sudo systemctl enable nvidia-docker-autoconf
sudo systemctl enable docker

sudo systemctl restart docker
sudo systemctl restart docker
