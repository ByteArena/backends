FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install curl netcat sudo apt-utils

COPY ./ $APP_HOME

WORKDIR $APP_HOME/bytearena/cmd/arena-server
RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/arena-server/arena-server /usr/bin

# Startup script
COPY ./bytearena/docker/arena-server/start.sh $APP_HOME
WORKDIR $APP_HOME

CMD ["bash", "start.sh"]
