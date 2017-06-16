FROM golang:1.8-stretch

ENV APP_HOME $GOPATH/src/github.com/bytearena

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install curl netcat sudo apt-utils gnupg

COPY ./ $APP_HOME

# Install nodejs
RUN curl -sL https://deb.nodesource.com/setup_6.x | sudo -E bash -
RUN apt-get install -y nodejs

WORKDIR $APP_HOME/bytearena/cmd/viz-server
RUN go get -v ./...
RUN go build
RUN cp -f $APP_HOME/bytearena/cmd/viz-server/viz-server /usr/bin

# Bundle viz fronted
COPY ./bytearena-viz /usr/bin/webclient
RUN npm --prefix /usr/bin/webclient install
RUN npm run --prefix /usr/bin/webclient build

# Startup script
COPY ./bytearena/docker/viz/start.sh $APP_HOME
WORKDIR $APP_HOME

CMD ["bash", "start.sh"]
