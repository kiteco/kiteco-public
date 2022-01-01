FROM python:2.7-slim

RUN apt-get update && apt-get install -y \
    liblapack-dev \
    libopenblas-dev \
    build-essential \
    gfortran \
    libfreetype6-dev \
    libpng-dev \
    libjpeg-dev \
    pkg-config \
    libxml2-dev \
    libxslt-dev \
    python-dev \
    libssl-dev \
    libcurl4-openssl-dev \
    zlib1g-dev \
    python-vtk

# Font for pygments examples
RUN apt-get install -y fontconfig
RUN apt-get install -y ttf-liberation

# Install packages that are order-dependent
RUN pip install numpy==1.9.2
RUN pip install scipy==0.15.1
RUN pip install matplotlib==1.4.3
RUN pip install scikit-learn==0.16.1
RUN pip install pandas==0.16.1
RUN pip install Pillow==2.7.0
RUN pip install Django==1.8.2
RUN pip install bokeh==0.9.0
RUN pip install seaborn==0.6.0
RUN pip install pyzmq==14.7.0
RUN pip install gevent==1.0.2

ADD requirements.txt /

RUN pip install -r requirements.txt

RUN mkdir /certs
RUN /usr/bin/yes 'xx' | /usr/bin/openssl req -new -newkey rsa:2048 -days 364 -nodes -x509 -keyout /certs/server.key -out /certs/server.crt

