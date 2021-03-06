#!/bin/bash
set -x

echo "# Starting ssh service"
service ssh start

echo "# Starting git-daemon"
su git -c "/usr/bin/git daemon --verbose --base-path=/home/git/repositories --export-all" &

echo "# Starting dotgit-mq-consumer"
su git -c "/usr/bin/dotgit-mq-consumer"
