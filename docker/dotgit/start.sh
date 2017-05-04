echo "starting ssh service"
service ssh start

echo "starting git-daemon"
su git -c "/usr/bin/git daemon --verbose --base-path=/home/git/repositories --export-all"
