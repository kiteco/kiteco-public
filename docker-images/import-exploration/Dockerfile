FROM debian:jessie

RUN apt-get update && apt-get install -y \
    wget \
    python \
    unzip \
    libopenblas-dev \
    liblapack-dev \
    build-essential \
    gfortran \
    libfreetype6-dev \
    libpng-dev \
    pkg-config \
    libxml2-dev \
    libxslt-dev \
    python-dev \
    libssl-dev \
    libcurl4-openssl-dev \
    libx11-dev \
    libsqlite3-dev \
    libjpeg-dev \
    libpng-dev \
    libmysqlclient-dev \
    python-gtk2 \
    python-qt4 \
    python-pygame \
    python-gobject

RUN apt-get install -y \
    python-numpy \
    python-scipy \
    python-scikits-learn \
    python-pandas \
    python-matplotlib \
    python-pip

RUN apt-get install -y \
    python-flask \
    python-django-south \
    python-twisted \
    python-sqlalchemy \
    python-mock \
    python-nose \
    python-requests \
    python-sympy \
    python-django \
    python-yaml \
    python-simplejson \
    python-pyramid \
    python-bottle \
    python-pillow \
    python-beautifulsoup \
    python-lxml \
    python-zmq \
    python-gevent \
    python-bson \
    python-httplib2 \
    python-nltk \
    python-webpy \
    python-pycurl \
    python-mysqldb \
    python-pymongo \
    python-jinja2 \
    python-werkzeug \
    python-googleapi \
    python-googlecloudapis \
    python-google-apputils \
    python-nova \
    python-novaclient \
    python-pyside \
    python-pyglet \
    python-cherrypy \
    python-docutils \
    python-pylons \
    python-vtk \
    python-boto \
    python-networkx \
    python-thrift \
    python-opengl \
    python-quantum

# install stuff that can only be installed by pip
RUN pip install \
    seaborn \
    bokeh \
    fabric

# Install PyQt5
RUN apt-get install -y \
    python-pyqt5

RUN apt-get install -y software-properties-common python-software-properties

# Additional Python packages
RUN apt-get install -y \
    python-mechanize \
    python-markdown \
    python-opencv \
    libldap2-dev \
    libsasl2-dev \
    python-biopython

# Install more python packages
RUN pip install redis \
    selenium \
    oauth2 \
    python-twitter \
    scrapy \
    tweepy \
    cherrypy \
    pytest \
    Celery \
    transaction \
    configparser \
    reportlab \
    webapp2 \
    python-ldap \
    psutil \
    autograd

RUN pip install --upgrade \
    pandas \
    django \
    pip

# Install utilities needed for cffi
RUN apt-get install -y libffi-dev \
    libssl-dev

# Install from package list
ADD pip-install.py /tmp
ADD packages /tmp
RUN python /tmp/pip-install.py /tmp/packages

# Install tensorflow
# https://www.tensorflow.org/versions/r0.8/get_started/os_setup.html#pip-installation
RUN wget https://storage.googleapis.com/tensorflow/linux/cpu/tensorflow-0.7.1-cp27-none-linux_x86_64.whl
RUN sudo pip install --upgrade tensorflow-0.7.1-cp27-none-linux_x86_64.whl

# More python packages that do not install properly from the packages list for some reason
# also these need to be installed after the main package list because ... magic!
RUN pip install --upgrade \
    bleach \
    brownie \
    theano \
    gensim \
    smart_open \
    bz2file \
    scikit-image \
    scikit-learn \
    troposphere \
    awacs

# Install reflectutils
RUN pip install reflectutils==0.25

# Install the entrypoint script
ADD entrypoint.py /
ENTRYPOINT ["/usr/bin/python", "/entrypoint.py"]

# Install kite-python
ADD kite-python.tar /tmp
RUN cd /tmp/kite-python && /usr/bin/python setup.py install
