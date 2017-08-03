#!/bin/bash
set -x

ID=$(cat /proc/sys/kernel/random/uuid)

function teardown {
    /usr/bin/mq-cli -mqhost="${MQHOST}" --publish "arena:stoped" --data "{\"id\": \"${ID}\"}"
    echo teardown
}

trap teardown EXIT

service docker start && sleep 5

# Make sure docker is running
docker ps

exec /usr/bin/arena-server --port "${PORT}" --mqhost "${MQHOST}" --apiurl "${APIURL}" --id "$ID"
