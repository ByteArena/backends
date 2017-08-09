#!/bin/bash
set -x

ID=$(cat /proc/sys/kernel/random/uuid)

function teardown {
    /usr/bin/mq-cli -mqhost="${MQHOST}" --publish "game:stopped" --data "{\"id\": \"${ID}\"}"
    echo teardown
}

trap teardown EXIT

exec $(
    /usr/bin/arena-server --port "${PORT}" --mqhost "${MQHOST}" --apiurl "${APIURL}" --id "$ID" --timeout "${GAME_TIMEOUT}" --registryAddr "${REGISTRY_ADDR}" --arenaAddr "${ARENA_ADDR}";
    teardown
)
