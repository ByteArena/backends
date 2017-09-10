FROM golang:1.8-stretch

ARG REGISTRY_ADDR

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl netcat gnupg2 sudo

RUN curl https://glide.sh/get | sh

COPY ./ $APP_HOME

WORKDIR $APP_HOME
RUN glide --debug install

# Build mq-cli
WORKDIR $APP_HOME/cmd/mq-cli
RUN go build
RUN cp -f $APP_HOME/cmd/mq-cli/mq-cli /usr/bin

WORKDIR $APP_HOME/cmd/arena-server
RUN go build
RUN cp -f $APP_HOME/cmd/arena-server/arena-server /usr/bin

# Startup script
COPY ./docker/arena-server/start.sh $APP_HOME
WORKDIR $APP_HOME

HEALTHCHECK --interval=1m --timeout=2s --retries=3 CMD ["curl", "--fail", "localhost:8099/health"]

CMD ["bash", "start.sh"]

