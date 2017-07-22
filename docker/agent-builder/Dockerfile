FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade

RUN apt-get --allow-unauthenticated -y install apt-transport-https ca-certificates curl software-properties-common
RUN apt-get -y install gnupg2
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"

RUN apt-get -y update
RUN apt-get -y install docker-ce

RUN curl https://glide.sh/get | sh

COPY . $APP_HOME
COPY ./docker/agent-builder/start.sh $APP_HOME

WORKDIR $APP_HOME
RUN glide --debug install

WORKDIR $APP_HOME/cmd/agent-builder

RUN go build
RUN cp -f $APP_HOME/cmd/agent-builder/agent-builder /usr/bin

COPY .secrets/git_admin_key /root/git_admin_key_private
RUN chmod 600 /root/git_admin_key_private

RUN mkdir -p /root/.ssh
COPY ./docker/agent-builder/ssh_config /root/.ssh/config

HEALTHCHECK --interval=1s --timeout=1s --retries=3 CMD ["curl", "--fail", "localhost:8099/health"]

CMD ["bash", "start.sh"]
