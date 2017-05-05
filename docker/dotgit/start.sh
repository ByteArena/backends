echo "# Starting ssh service"
service ssh start

echo "# Starting git-daemon"
su git -c "/usr/bin/git daemon --verbose --base-path=/home/git/repositories --export-all" &

echo "# Starting dotgit-api"
su git -c "/usr/bin/dotgit-api"