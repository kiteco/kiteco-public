FROM debian:jessie

RUN apt-get update && apt-get install -y \
    wget \
    git

RUN mkdir /certs
RUN /usr/bin/yes 'xx' | /usr/bin/openssl req -new -newkey rsa:2048 -days 364 -nodes -x509 -keyout /certs/server.key -out /certs/server.crt
