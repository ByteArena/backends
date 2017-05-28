echo "# Starting ssh service"
service ssh start

echo "# Wait for mysql"
while ! nc -z $MYSQL_HOST 3306; do sleep 2; done

echo "# Starting git-daemon"
su git -c "/usr/bin/git daemon --verbose --base-path=/home/git/repositories --export-all" &

echo "# Starting dotgit-api"
su git -c "/usr/bin/dotgit-api"
