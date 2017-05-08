#!/usr/bin/env bash

set -e

go get github.com/bytearena/dotgit

MQ_DIRECTORY=$GOPATH/src/github.com/bytearena/bytearena-message-broker

if [ ! -d "$MQ_DIRECTORY" ]; then
  git clone git@github.com:ByteArena/bytearena-message-broker.git $MQ_DIRECTORY
fi

./compose.sh build

