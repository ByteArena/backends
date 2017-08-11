FROM registry:2

ARG DOCKER_REGISTRY_TLS_PRIVATE_KEY

RUN echo -e "$DOCKER_REGISTRY_TLS_PRIVATE_KEY" | base64 -d > /etc/docker/registry.net.bytearena.com:5000.key

COPY registry.net.bytearena.com:5000.crt /etc/docker/registry.net.bytearena.com:5000.crt
COPY config.yml /etc/docker/registry/config.yml
