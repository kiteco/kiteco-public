#!/usr/bin/env bash
set -e

# convenience stuff
sudo apt-get update
sudo apt-get install htop -y

# install libtensorflow
# TODO: we can have flag as to whether to enable it or not
DLHOST=https://storage.googleapis.com/tensorflow/libtensorflow
FILENAME=libtensorflow-cpu-linux-x86_64-1.15.0.tar.gz
echo "Downloading $FILENAME"
curl $DLHOST/$FILENAME -o $FILENAME
echo "Installing c libraries, this may take a while"
sudo tar -C /usr/local/ -xzf $FILENAME
rm -f $FILENAME
sudo ldconfig

{{if .CUDA}}
sudo apt-get install -y openjdk-8-jdk git python-dev python3-dev python-numpy python3-numpy build-essential \
    python-pip python3-pip python-virtualenv swig python-wheel libcurl3-dev curl
mkdir /var/kite/gputmp
cd /var/kite/gputmp
curl -O http://developer.download.nvidia.com/compute/cuda/repos/ubuntu1604/x86_64/cuda-repo-ubuntu1604_9.0.176-1_amd64.deb
sudo apt-key adv --fetch-keys http://developer.download.nvidia.com/compute/cuda/repos/ubuntu1604/x86_64/7fa2af80.pub
sudo dpkg -i ./cuda-repo-ubuntu1604_9.0.176-1_amd64.deb
sudo apt-get update
sudo apt-get install -y cuda-9-0

wget https://XXXXXXX.s3-us-west-1.amazonaws.com/cudnn-9.0-linux-x64-v7.3.1.20.tgz
sudo tar -xzvf cudnn-9.0-linux-x64-v7.3.1.20.tgz
sudo cp cuda/include/cudnn.h /usr/local/cuda/include
sudo cp cuda/lib64/libcudnn* /usr/local/cuda/lib64
sudo chmod a+r /usr/local/cuda/include/cudnn.h /usr/local/cuda/lib64/libcudnn*

echo 'export LD_LIBRARY_PATH="$LD_LIBRARY_PATH:/usr/local/cuda/lib64:/usr/local/cuda/extras/CUPTI/lib64"' >> /var/kite/bundle/env.sh
echo 'export CUDA_HOME=/usr/local/cuda' >> /var/kite/bundle/env.sh
echo 'export PATH="$PATH:/usr/local/cuda/bin"' >> /var/kite/bundle/env.sh
{{end}}

{{if .DockerML}}
# Install headless driver, we don't need Xorg, etc...
sudo add-apt-repository -y ppa:graphics-drivers/ppa
sudo apt update
sudo apt install -y nvidia-headless-440 nvidia-utils-440

# Force driver load without reboot
sudo nvidia-smi

# Install docker via instructions from docker's website
sudo apt-get install -y apt-transport-https ca-certificates curl gnupg-agent software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository -y \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# Install nvidia-container-toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo systemctl restart docker

# Install kite_ml into the container
current=$PWD
cd /var/kite/bundle

cat << 'EOF' >> Dockerfile
FROM tensorflow/tensorflow:1.15.2-gpu-py3
RUN pip install --upgrade pip
RUN pip install scipy


ENV OPENMPI_VERSION=4.0.3
ENV NCCL_VERSION=2.4.8-1+cuda10.0

RUN apt-get update && apt-get install -y --allow-downgrades --allow-change-held-packages --no-install-recommends \
        build-essential \
        cmake \
        g++-4.8 \
        git \
        curl \
        wget \
        ca-certificates \
        libnccl2=${NCCL_VERSION} \
        libnccl-dev=${NCCL_VERSION} \
        libjpeg-dev \
        libpng-dev \
        librdmacm1 \
        libibverbs1 \
        ibverbs-providers

# Install Open MPI
RUN mkdir /tmp/openmpi && \
    cd /tmp/openmpi && \
    wget https://www.open-mpi.org/software/ompi/v4.0/downloads/openmpi-${OPENMPI_VERSION}.tar.gz && \
    tar zxf openmpi-${OPENMPI_VERSION}.tar.gz && \
    cd openmpi-${OPENMPI_VERSION} && \
    ./configure --enable-orterun-prefix-by-default && \
    make -j $(nproc) all && \
    make install && \
    ldconfig && \
    rm -rf /tmp/openmpi

# Install Horovod, temporarily using CUDA stubs
RUN ldconfig /usr/local/cuda/lib64/stubs && \
    HOROVOD_CUDA_HOME=/usr/local/cuda \
    HOROVOD_GPU_ALLREDUCE=NCCL HOROVOD_GPU_BROADCAST=NCCL \
    HOROVOD_WITH_TENSORFLOW=1 HOROVOD_WITHOUT_PYTORCH=1 HOROVOD_WITHOUT_MXNET=1 \
    pip install --no-cache-dir horovod && \
    ldconfig

# Install OpenSSH for MPI to communicate between containers
RUN apt-get install -y --no-install-recommends openssh-client openssh-server && \
    mkdir -p /var/run/sshd

# Allow OpenSSH to talk to containers without asking for confirmation
RUN cat /etc/ssh/ssh_config | grep -v StrictHostKeyChecking > /etc/ssh/ssh_config.new && \
    echo "    StrictHostKeyChecking no" >> /etc/ssh/ssh_config.new && \
    mv /etc/ssh/ssh_config.new /etc/ssh/ssh_config

WORKDIR /
ADD kiteco/kite-python/kite_ml /kite_ml
RUN cd /kite_ml && pip install -e .

ENV PATH="/var/kite/bundle/bin:${PATH}"
ENV LD_LIBRARY_PATH="/usr/local/lib:${LD_LIBRARY_PATH}"
RUN curl -L "https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-gpu-linux-x86_64-1.15.0.tar.gz" | tar -C /usr/local -xz
ENV CUDA_CACHE_PATH=/cudacache
EOF

sudo docker image build -t tensorflow:local .
{{end}}

{{if .KiteML}}
# install dependencies and create/load virtualenv for kite_ml
sudo add-apt-repository ppa:deadsnakes/ppa -y
sudo apt-get update
sudo apt-get install python3.6 virtualenv -y

cd /var/kite/bundle/kiteco/kite-python/kite_ml
virtualenv -p python3.6 venv
source venv/bin/activate
pip install -r requirements.txt
pip install -e .
deactivate
{{end}}

cat << 'EOF' > /var/kite/bundle/activate.sh
source /var/kite/bundle/env.sh
if [ -d /var/kite/bundle/kiteco/kite-python/kite_ml/venv ]; then
    source /var/kite/bundle/kiteco/kite-python/kite_ml/venv/bin/activate
fi
EOF

cat << 'EOF' > /var/kite/bundle/start.sh
source /var/kite/bundle/activate.sh
bash /var/kite/bundle/run.sh
EOF

cat << 'EOF' > /home/ubuntu/.bash_aliases
source /var/kite/bundle/activate.sh
alias bundle='cd /var/kite/bundle'
alias train='cd /var/kite/bundle/kiteco/local-pipelines/lexical/train'
EOF
