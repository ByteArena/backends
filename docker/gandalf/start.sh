echo "start rsyslog"

service rsyslog start

service ssh start

echo "starting gandalf-server"

su git -c "/usr/bin/gandalf-server" &

echo "starting git-daemon"

su git -c "/usr/bin/git daemon --verbose --base-path=/home/git/repositories --export-all"