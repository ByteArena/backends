FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

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
RUN echo "allow brtest" > /etc/qemu/bridge.conf

# HEALTHCHECK --interval=1m --timeout=2s --retries=3 CMD ["curl", "--fail", "localhost:8099/health"]

CMD ["./arena-master"]
