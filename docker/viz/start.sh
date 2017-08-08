#!/bin/bash
set -x

exec /usr/bin/viz-server --port "${PORT}" --mqhost "${MQHOST}" --apiurl "${APIURL}" --record-dir "${RECORD_DIR}"
