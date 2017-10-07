FROM golang:1.8-alpine3.5 as builder

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

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

# https://github.com/moby/tool/issues/161
# ENV PORT=7777
# ENV MQHOST=redis.net.bytearena.com
# ENV APIURL=graphql.net.bytearena.com
# ENV GAME_TIMEOUT=10
# ENV REGISTRY_ADDR=registry.net.bytearena.com
# ENV ARENA_ADDR=arena.local
# ENV DOCKER_HOST=tcp://127.0.0.1:2375

ENV PORT=7777
ENV MQHOST=10.0.5.0
ENV APIURL=10.0.5.0
ENV GAME_TIMEOUT=10
ENV REGISTRY_ADDR=10.0.5.0
ENV ARENA_ADDR=127.0.0.1
ENV DOCKER_HOST=tcp://127.0.0.1:2375

# ID will be changed at runtine
ENV ID=UNKNOW

RUN apk update
RUN apk add git sudo gnupg

COPY --from=builder /usr/bin/arena-server /usr/bin/arena-server
COPY --from=dockerbuilder /usr/local/bin/docker /usr/bin/docker

RUN chmod +x /usr/bin/docker

# Startup script
COPY ./docker/arena-server/start.sh .

CMD ["sh", "start.sh"]

