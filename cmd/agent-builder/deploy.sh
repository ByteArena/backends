#!/bin/bash

NAME="$1"
REGISTRY="$2"
TAG="$3"

# Create tag to push to remote registry

docker tag $NAME $REGISTRY/$NAME:$TAG

# Push to remote registry

docker push $REGISTRY/$NAME:$TAG

