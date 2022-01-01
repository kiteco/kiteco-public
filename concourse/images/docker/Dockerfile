FROM ubuntu:bionic

ARG GO_VERSION=1.15.3

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    curl \
    wget \
    gzip \
    zip unzip \
    openssl \
    libssl-dev \
    make \
    openssh-client \
    libstdc++6 \
    software-properties-common \
    openjdk-11-jre-headless \
    openjdk-8-jdk \
    makeself \
    chrpath \
    gcc \
    build-essential \
    gpg-agent \
    jq

RUN curl -sL https://deb.nodesource.com/setup_12.x | bash -
RUN apt-get install -y nodejs
RUN npm install -g n
RUN n 11.12.0

RUN apt-add-repository ppa:git-core/ppa \
  && apt-get update \
  && apt-get install -y git \
  && rm -rf /var/lib/apt/lists/*

RUN curl -s https://packagecloud.io/install/repositories/github/git-lfs/script.deb.sh | bash
RUN apt-get install -y git-lfs

RUN wget https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
RUN rm go$GO_VERSION.linux-amd64.tar.gz
ENV PATH=/usr/local/go/bin:$PATH

RUN apt-get install -y --no-install-recommends python3.7 python3-pip python3-setuptools
RUN pip3 install awscli wheel pipenv

RUN update-alternatives --set java /usr/lib/jvm/java-8-openjdk-amd64/jre/bin/java

RUN git config --global user.email "ops@kite.com"
RUN git config --global user.name "Kite Concourse"
