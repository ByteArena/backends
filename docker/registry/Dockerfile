FROM registry:2

ARG DOCKER_REGISTRY_TLS_PRIVATE_KEY

RUN echo -e "$DOCKER_REGISTRY_TLS_PRIVATE_KEY" | base64 -d > /etc/docker/registry.net.bytearena.com.key

COPY registry.net.bytearena.com.crt /etc/docker/registry.net.bytearena.com.crt
COPY config.yml /etc/docker/registry/config.yml
