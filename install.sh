#!/usr/bin/env bash

set -e

go get github.com/bytearena/dotgit

MQ_DIRECTORY=$GOPATH/src/github.com/bytearena/bytearena-message-broker

if [ ! -d "$MQ_DIRECTORY" ]; then
  git clone https://github.com/ByteArena/bytearena-message-broker.git $MQ_DIRECTORY
fi

VIZCLIENT_DIRECTORY=$GOPATH/src/github.com/bytearena/bytearena/cmd/viz-server/webclient

if [ ! -d "$VIZCLIENT_DIRECTORY" ]; then
  git clone https://github.com/ByteArena/bytearena-viz.git $VIZCLIENT_DIRECTORY
  cd $VIZCLIENT_DIRECTORY
  npm install
  npm run build
  cd -
fi

./compose.sh build

