FROM golang:1.8-stretch

ARG GIT_ADMIN_KEY_PRIVATE

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

COPY . $APP_HOME

WORKDIR $APP_HOME/cmd/agent-builder

RUN go get -v ./...
RUN go build

RUN echo "$GIT_ADMIN_KEY_PRIVATE" | base64 -d > /root/git_admin_key_private
RUN chmod 600 /root/git_admin_key_private

RUN mkdir -p /root/.ssh
COPY ./docker/agent-builder/ssh_config /root/.ssh/config

CMD ["agent-builder"]
