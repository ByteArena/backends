FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install curl netcat

COPY ./ $APP_HOME

WORKDIR $APP_HOME/bytearena/cmd/viz-server
RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/viz-server/viz-server /usr/bin

CMD /usr/bin/viz-server
