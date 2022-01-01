FROM ubuntu:20.04
# Fixes x509: certificate signed by unknown authority when fetching from AWS
RUN apt-get update && apt-get install -y \
  ca-certificates \
  curl \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/svc/
COPY build/convcohort .
ENTRYPOINT  ["./convcohort"]
