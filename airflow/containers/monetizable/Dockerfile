FROM ubuntu:20.04

# Fixes x509: certificate signed by unknown authority when fetching from AWS
RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/*

ARG BUILD_HASH=unset
ENV BUILD_HASH=$BUILD_HASH

WORKDIR /opt/svc/
COPY build/monetizable .
ENTRYPOINT  ["./monetizable"]
