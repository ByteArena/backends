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

# https://github.com/moby/tool/issues/161
# ENV MQHOST=redis.net.bytearena.com
# ENV APIURL=graphql.net.bytearena.com
# ENV REGISTRY_ADDR=registry.net.bytearena.com
# ENV ARENA_ADDR=arena.local
# ENV DOCKER_HOST=tcp://127.0.0.1:2375

ENV MQHOST=10.0.5.0
ENV APIURL=http://10.0.5.0:9000/privateapi/graphql
ENV GAME_TIMEOUT=10
ENV REGISTRY_ADDR=10.0.5.0:5000
ENV DOCKER_HOST=tcp://127.0.0.1:2375
ENV AGENT_LOGS_PATH=/tmp/agent-logs

# Docker gateway (force in .bip)
ENV ARENA_ADDR=172.17.0.1

RUN apk update
RUN apk add git sudo gnupg

COPY --from=builder /usr/bin/arena-server /usr/bin/arena-server
COPY --from=dockerbuilder /usr/local/bin/docker /usr/bin/docker

RUN chmod +x /usr/bin/docker

# Startup script
COPY ./docker/arena-server/start.sh .

CMD ["sh", "start.sh"]

