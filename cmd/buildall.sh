#!/bin/sh
set -x

BUILDFLAGS="-v -a"

cd agent-builder && go build $BUILDFLAGS && cd .. && \
cd arena-master && go build $BUILDFLAGS && cd .. && \
cd arena-server && go build $BUILDFLAGS && cd .. && \
cd arena-trainer && go build $BUILDFLAGS && cd .. && \
cd dotgit-keystore && go build $BUILDFLAGS && cd .. && \
cd dotgit-mq-consumer && go build $BUILDFLAGS && cd .. && \
cd dotgit-ssh && go build $BUILDFLAGS && cd .. && \
cd map-builder && go build $BUILDFLAGS && cd .. && \
cd mq-cli && go build $BUILDFLAGS && cd .. && \
cd viz-server && go build $BUILDFLAGS && cd .. && \
echo "ALL BUILT !"
