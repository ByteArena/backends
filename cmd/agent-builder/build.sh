#!/bin/bash

NAME="$1"
SOURCE="$2"

docker build -t $NAME $SOURCE
