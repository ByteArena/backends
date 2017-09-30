FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl netcat qemu-kvm

RUN curl https://glide.sh/get | sh

COPY ./ $APP_HOME

WORKDIR $APP_HOME
RUN glide --debug install

WORKDIR $APP_HOME/cmd/arena-master

RUN go build

# HEALTHCHECK --interval=1m --timeout=2s --retries=3 CMD ["curl", "--fail", "localhost:8099/health"]

CMD ["./arena-master"]
