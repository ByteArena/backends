#!/usr/bin/env sh

bash "heroku-buildpack-$1/bin/compile" $2 $3 1>&2 > log
