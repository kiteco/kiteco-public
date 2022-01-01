#!/bin/bash

# Setup disks
sudo mkfs -t ext4 /dev/sdc
sudo mkdir /data
sudo mount /dev/sdc /data

# Setup env
echo "export GOROOT=/usr/local/go" >> ~/.bashrc
echo "export GOPATH=\$HOME/go" >> ~/.bashrc
echo "export PATH=\$PATH:\$GOROOT/bin:\$GOPATH/bin" >> ~/.bashrc

# Setup appropriate directories for kite s3 readers
sudo mkdir /var/kite
sudo chown -R ubuntu /var/kite
sudo chown -R ubuntu /data


mkdir $HOME/go 
mkdir $HOME/go/src $HOME/go/bin $HOME/go/pkg

# Install build tools
sudo apt-get install build-essential -y

# Install python 3.6
sudo add-apt-repository ppa:deadsnakes/ppa -y
sudo apt-get update
sudo apt-get install python3.6 virtualenv -y

# Misc convenience stuff
sudo apt-get install htop -y

# install libtensorflow
DLHOST=https://storage.googleapis.com/tensorflow/libtensorflow
FILENAME=libtensorflow-cpu-linux-x86_64-1.8.0.tar.gz
echo "Downloading $FILENAME"
curl $DLHOST/$FILENAME -o $FILENAME
echo "Installing c libraries, this may take awhile"
sudo tar -C /usr/local/ -xzf $FILENAME
rm -f $FILENAME
sudo ldconfig

