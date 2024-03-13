#!/usr/bin/env bash
set -eux

# Home directory
if [ -z ${1+x} ] ; then
  KV_HOME="${HOME}/.kvstore"
else
  KV_HOME=$1
fi
if [[ -d ${KV_HOME} ]]; then
    rm -r ${KV_HOME} || true
else
    mkdir ${KV_HOME}
fi

LISTEN_ADDRESS=unix:///tmp/kvapp.sock
APP_BIN=$(which kvstore)
echo Running KV store
${APP_BIN} -kv-home ${KV_HOME} -v=3 -socket-addr ${LISTEN_ADDRESS}

