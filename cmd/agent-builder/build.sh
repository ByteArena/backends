#!/usr/bin/env sh

CONTAINER_ID=$(docker run --workdir /root -d ubuntu:16.04 /usr/bin/tail -f /dev/null)

exec_container() {
  echo "---> $1"
  docker exec $CONTAINER_ID $1
}

copy_container() {
  echo "---> $1:$2"
  docker cp $1 $CONTAINER_ID:$2
}

generate_entrypoint() {
  echo "---> entrypoint"

  sudo touch /root/entrypoint

  echo "#!/usr/bin/env sh" >> /root/entrypoint
  echo "source /root/.profile.d/*.sh" >> /root/entrypoint
  echo "npm start" >> /root/entrypoint
}

commit_and_tag_container() {
  echo "---> commit"

  BUILD_DATE=$(date +%Y%m%d)

  SHA256=$(docker commit --author "ByteArena whatever <a@b.fr>" --message "Builded at $BUILD_DATE" $CONTAINER_ID)
  NAME=$(echo $SHA256 | sed 's/sha256://')

  echo "---> tag $NAME"

  docker tag $NAME bytearena_foo
}

exec_container "apt-get update"
exec_container "apt-get install -y curl"

(
  set -e

  copy_container ./heroku-buildpack-$1/ /heroku-buildpack-$1/
  copy_container $2/. /root

  exec_container "ls /root"
  exec_container "/heroku-buildpack-$1/bin/compile /root /tmp"

  generate_entrypoint

  commit_and_tag_container

  docker stop $CONTAINER_ID
  docker rm $CONTAINER_ID
)

if [ $? = 1 ]
then
  echo "Exited with code 1"
  docker kill $CONTAINER_ID
  docker rm -f $CONTAINER_ID
fi
