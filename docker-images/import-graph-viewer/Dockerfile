FROM ubuntu
RUN apt-get update -y && apt-get install -y ca-certificates
ADD import-graph-viewer /
EXPOSE 8000
ENTRYPOINT ["/import-graph-viewer", "--port=:8000"]
