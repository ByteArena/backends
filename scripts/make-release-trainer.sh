#!/usr/bin/env bash

VERION=$(git rev-parse HEAD)
FILENAME=arena-trainer-$VERION

cd cmd/arena-trainer && go build -o ../../$FILENAME -ldflags="-s -w" -v

echo "Generated $FILENAME"
