FROM golang:1.8-stretch

ARG REGISTRY_ADDR

ENV APP_HOME $GOPATH/src/github.com/bytearena/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install git openssh-server curl netcat gnupg2 sudo

RUN apt-get -y install apt-transport-https ca-certificates curl software-properties-common
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"

RUN apt-get -y update
RUN apt-get -y install docker-ce

RUN mkdir /etc/docker
RUN echo "{ \"insecure-registries\" : [\"$REGISTRY_ADDR\"] }" | sudo tee /etc/docker/daemon.json

COPY ./ $APP_HOME

WORKDIR $APP_HOME/cmd/arena-server

RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/cmd/arena-server/arena-server /usr/bin

# Startup script
COPY ./docker/arena-server/start.sh $APP_HOME
WORKDIR $APP_HOME

CMD ["bash", "start.sh"]

