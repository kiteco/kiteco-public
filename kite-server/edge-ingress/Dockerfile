FROM envoyproxy/envoy:v1.15-latest

VOLUME /etc/envoy/ssl

COPY envoy.yaml /etc/envoy/envoy.yaml
COPY ssl/ /etc/envoy/ssl/
