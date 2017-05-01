# TODO(sven): check for keys

GIT_ADMIN_KEY_PRIVATE=$(cat "$PWD/.keys/git_admin_key" | base64) \
GIT_ADMIN_KEY_PUBLIC=$(cat "$PWD/.keys/git_admin_key.pub" | base64) \
\
docker-compose -f docker-compose.dev.yml "$@"
