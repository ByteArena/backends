#!/bin/bash
set -x

echo -n "$PRIVATE_KEY" | base64 -d > /root/git_admin_key_private
cat /root/git_admin_key_private
chmod 600 /root/git_admin_key_private

/usr/bin/agent-builder
