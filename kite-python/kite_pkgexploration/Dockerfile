FROM kiteco/python-import-exploration-base

RUN apt update

# install system Python2 for PyQt4
# install prerequisites from https://github.com/pyenv/pyenv/wiki/Common-build-problems
# install ca-certificates so curl doesn't fail to install pyenv
RUN apt install --no-install-recommends -y \
    python-pip \
    make build-essential libssl-dev zlib1g-dev libbz2-dev \
    libreadline-dev libsqlite3-dev wget curl llvm libncurses5-dev libncursesw5-dev \
    xz-utils tk-dev libffi-dev liblzma-dev python-openssl git \
    ca-certificates
# install pyenv using https://github.com/pyenv/pyenv-installer
RUN curl https://pyenv.run | bash
# set PATH -- this persists into any containers based on this image
ENV PATH="/root/.pyenv/shims:/root/.pyenv/bin:${PATH}"

# prerequisites for TKinter
RUN apt install --no-install-recommends -y tk-dev

# clean up
RUN rm -rf /var/lib/apt/lists/*

# install Pythons
RUN pyenv install 2.7.16
RUN pyenv install 3.7.3

# update global pip/setuptools
RUN pyenv global system
RUN python -m pip install --upgrade pip
RUN python -m pip install setuptools

# do this at the end so rebuilds are fast
COPY . /root/kite_pkgexploration

# install kite.pkgexploration to global Python2
RUN pyenv global system
RUN python -m pip install -e /root/kite_pkgexploration/

# default to pyenv Python2/3
RUN pyenv global 3.7.3 2.7.16
# install kite.pkgexploration to pyenv Python2/3
RUN python3 -m pip install -e /root/kite_pkgexploration/
RUN python2 -m pip install -e /root/kite_pkgexploration/
