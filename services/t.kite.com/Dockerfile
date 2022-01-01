FROM ubuntu:20.04

RUN apt-get update && apt-get install -y \
   curl \
   && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/svc/
COPY build/t.kite.com .
COPY build/schemas ./schemas
ENTRYPOINT ["/opt/svc/t.kite.com", "-schemaDir", "/opt/svc/schemas"]