#!/usr/bin/env bash
set -e

FILENAME=mq-cli
DIRECTORY=../../build/releases

BUILDS=(
    "GOARCH=amd64 GOOS=linux"
)

mkdir -p $DIRECTORY

cd cmd/mq-cli/


for i in "${BUILDS[@]}"
do
    echo $i
    eval $i

    FILE=$DIRECTORY/$FILENAME-$GOARCH-$GOOS

    env $i go build -o "$FILE" -ldflags="-s -w"
    upx -9 $FILE
done
