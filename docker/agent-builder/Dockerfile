FROM golang:1.8-stretch

ARG GIT_ADMIN_KEY_PRIVATE
ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade

RUN apt-get -y install apt-transport-https ca-certificates curl software-properties-common
RUN apt-get -y install gnupg2
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"

RUN apt-get -y update
RUN apt-get -y install docker-ce

COPY . $APP_HOME

WORKDIR $APP_HOME/cmd/agent-builder

RUN go get -v ./...
RUN go build

RUN echo "$GIT_ADMIN_KEY_PRIVATE" | base64 -d > /root/git_admin_key_private
RUN chmod 600 /root/git_admin_key_private

RUN mkdir -p /root/.ssh
COPY ./docker/agent-builder/ssh_config /root/.ssh/config

CMD ["agent-builder"]
