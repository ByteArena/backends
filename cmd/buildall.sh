#!/usr/bin/env bash

BUILDFLAGS=""
COMMANDS=$(find * -maxdepth 0 -type d)

for i in $COMMANDS
do
   : 
   printf "# Building ${i}"

   (cd "$i" && go build $BUILDFLAGS && cd ..)

   if [[ "$?" -eq 0 ]]
   then
       echo " OK"
   fi

done
