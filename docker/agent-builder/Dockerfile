FROM golang:latest

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

COPY . $APP_HOME

WORKDIR $APP_HOME/cmd/agent-builder

RUN go get -v ./...
RUN go build

CMD ["agent-builder"]
