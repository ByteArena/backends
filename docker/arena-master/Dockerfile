FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl netcat

COPY ./ $APP_HOME

WORKDIR $APP_HOME/cmd/arena-master

RUN go get -v ./...
RUN go build

HEALTHCHECK --interval=1s --timeout=1s --retries=3 CMD ["curl", "--fail", "localhost:8099/health"]

CMD ["arena-master"]
