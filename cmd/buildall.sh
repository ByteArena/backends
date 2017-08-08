#!/bin/sh
#set -x

BUILDFLAGS="-v"
COMMANDS=$(find * -maxdepth 0 -type d)

for i in $COMMANDS
do
   : 
   # do whatever on $i
   echo "############################################################"
   echo "# Building ${i}"
   echo "############################################################"
   echo ""
   cd "$i" && go build $BUILDFLAGS && cd ..
   echo ""
done

