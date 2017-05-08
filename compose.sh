#!/usr/bin/env sh
# TODO(sven): check for keys
export $(cat .secrets/params | xargs)

GIT_ADMIN_KEY_PRIVATE=$(cat "$PWD/.secrets/git_admin_key" | base64) \
GIT_ADMIN_KEY_PUBLIC=$(cat "$PWD/.secrets/git_admin_key.pub" | base64) \
docker-compose -f docker-compose.dev.yml "$@"
