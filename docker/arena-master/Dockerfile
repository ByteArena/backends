FROM golang:1.8-stretch

ARG BRIDGE

ENV APP_HOME $GOPATH/src/github.com/bytearena/backends

RUN \
        apt-get update && \
        apt-get install -y qemu-kvm kvm qemu-utils bridge-utils dnsmasq uml-utilities iptables wget net-tools && \
        apt-get autoclean && \
        apt-get autoremove && \
        rm -rf /var/lib/apt/lists/*

RUN curl https://glide.sh/get | sh

COPY ./ $APP_HOME

WORKDIR $APP_HOME
RUN glide --debug install

WORKDIR $APP_HOME/cmd/arena-master

RUN go build

RUN mkdir /etc/qemu/
RUN echo "allow ${BRIDGE}" > /etc/qemu/bridge.conf

# HEALTHCHECK --interval=1m --timeout=2s --retries=3 CMD ["curl", "--fail", "localhost:8099/health"]

COPY ./docker/arena-master/start.sh .
RUN chmod +x start.sh

CMD ["./start.sh"]
