{

    set -x

    # ID=$(cat /proc/sys/kernel/random/uuid)
    ID=$(cat cat /sys/class/net/eth0/address)

    exec /usr/bin/arena-server --port "${PORT}" --mqhost "${MQHOST}" --apiurl "${APIURL}" --id "$ID" --timeout "${GAME_TIMEOUT}" --registryAddr "${REGISTRY_ADDR}" --arenaAddr "${ARENA_ADDR}"

} | tee -a /dev/ttyS0
