#!/bin/bash

apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys XXXXXXX
echo "deb https://apt.dockerproject.org/repo ubuntu-trusty main" >> /etc/apt/sources.list.d/docker.list

apt-get -y update
apt-get -y install apt-transport-https
apt-get -y install ca-certificates
apt-get -y install docker-engine
apt-get -y install linux-image-extra-$(uname -r)

usermod -aG docker ubuntu
