echo "start rsyslog"

# /usr/bin/rsyslog &
service rsyslog start

service ssh start

echo "starting git-daemon"

/usr/bin/git daemon --base-path=/home/git/repositories --detach --export-all &

echo "starting gandalf-server"

su git -c "/usr/bin/gandalf-server"
