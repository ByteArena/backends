{
    set -x

    ID=$(cat /sys/class/net/eth0/address)

    /usr/bin/arena-server \
        --mqhost "${MQHOST}" \
        --apiurl "${APIURL}" \
        --id "$ID" \
        --timeout "${GAME_TIMEOUT}" \
        --registryAddr "${REGISTRY_ADDR}" \
        --arenaAddr "${ARENA_ADDR}"

} | tee -a /dev/ttyS0
