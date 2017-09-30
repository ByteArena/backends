FROM golang:1.8-alpine3.5 as builder

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

# RUN apt-get -y update
# RUN apt-get -y upgrade
# RUN apt-get -y install git openssh-server curl netcat gnupg2 sudo

RUN apk update
RUN apk add git curl sudo gnupg

RUN curl https://glide.sh/get | sh

COPY ./ $APP_HOME

WORKDIR $APP_HOME
RUN glide --debug install

WORKDIR $APP_HOME/cmd/arena-server
RUN go build
RUN cp -f $APP_HOME/cmd/arena-server/arena-server /usr/bin

FROM docker:17.07 as dockerbuilder

FROM alpine:3.5

ARG REGISTRY_ADDR

ENV PORT=7777
# ENV MQHOST=redis.net.bytearena.com
ENV MQHOST=10.0.2.10
ENV APIURL=10.0.2.10:8080
# ID will be changed at runtine
ENV ID=UNKNOW
ENV GAME_TIMEOUT=10
ENV REGISTRY_ADDR=registry.net.bytearena.com
ENV ARENA_ADDR=arena.local
ENV DOCKER_HOST=tcp://127.0.0.1:2375

RUN apk update
RUN apk add git sudo gnupg

COPY --from=builder /usr/bin/arena-server /usr/bin/arena-server
COPY --from=dockerbuilder /usr/local/bin/docker /usr/bin/docker

RUN chmod +x /usr/bin/docker

# Startup script
COPY ./docker/arena-server/start.sh .

CMD ["sh", "start.sh"]

