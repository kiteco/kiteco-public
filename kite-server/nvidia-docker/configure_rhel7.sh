#!/usr/bin/bash

set -ex

sudo yum install -y yum-utils

# nvidia drivers: https://docs.nvidia.com/cuda/cuda-installation-guide-linux/index.html#redhat-installation
sudo yum install -y kernel-devel-$(uname -r) kernel-headers-$(uname -r)
sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
sudo yum-config-manager \
    --add-repo \
    https://developer.download.nvidia.com/compute/cuda/repos/rhel7/x86_64/cuda-rhel7.repo
sudo yum clean all
sudo yum install -y nvidia-driver-latest-dkms

# Docker
sudo yum-config-manager \
    --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install docker-ce docker-ce-cli containerd.io -y
sudo systemctl start docker

# nvidia-container-runtime: https://nvidia.github.io/nvidia-container-runtime/
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-container-runtime/$distribution/nvidia-container-runtime.repo | \
  sudo tee /etc/yum.repos.d/nvidia-container-runtime.repo
sudo yum install -y nvidia-container-runtime
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

sudo yum install -y python3-pip

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
Description=autoconfigure GPUs as node-generic-resources when Docker starts

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
