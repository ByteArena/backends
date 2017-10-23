#!/bin/sh -x

./arena-master 2>&1 | tee -a /var/log/arenamaster.log
