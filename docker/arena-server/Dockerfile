FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl netcat

COPY ./ $APP_HOME

WORKDIR $APP_HOME/cmd/arena-server

RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/cmd/arena-server/arena-server /usr/bin

# Startup script
COPY ./docker/arena-server/start.sh $APP_HOME
WORKDIR $APP_HOME

CMD ["bash", "start.sh"]
