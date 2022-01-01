#!/bin/bash

. /root/ssl_env_vars.sh
. /root/state_env_vars.sh

# restore haproxy state if it exists
yes | azcopy --source "${AZURE_STATE_STORAGE_PATH}" \
       --destination /etc/haproxy/haproxy.state \
       --source-key "${AZURE_STATE_ACCESS_KEY}"

# fetch ssl certs for haproxy fronting
yes | azcopy --source https://kitessl.blob.core.windows.net/kitessl/server.pem \
       --destination /etc/ssl/private/kite.pem \
       --source-key "${AZURE_SSL_ACCESS_KEY}"

sleep 2 && sudo service haproxy restart
