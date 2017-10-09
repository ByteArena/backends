{

    set -x

    ID=$(cat cat /sys/class/net/eth0/address)

    export AGENT_LOGS_PATH=/tmp/agent-logs

    exec /usr/bin/arena-server --port "${PORT}" --mqhost "${MQHOST}" --apiurl "${APIURL}" --id "$ID" --timeout "${GAME_TIMEOUT}" --registryAddr "${REGISTRY_ADDR}" --arenaAddr "${ARENA_ADDR}"

} | tee -a /dev/ttyS0
