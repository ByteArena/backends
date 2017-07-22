#!/bin/bash


# function teardown {
#     echo teardown
# }

# trap teardown EXIT

/usr/bin/arena-server --port "${PORT}" --mqhost "${MQHOST}" --apiurl "${APIURL}"
