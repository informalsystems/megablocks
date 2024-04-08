#!/usr/bin/env bash
set -eux

# Home directory
if [ -z ${1+x} ] ; then
  COSMUX_HOME="${HOME}/.cosmux"
else
  COSMUX_HOME=$1
fi

# init home
if [ ! -d "$COSMUX_HOME" ] ; then
  echo "Initializing Cosmux"
  go run github.com/cometbft/cometbft/cmd/cometbft@v0.38.5 init --home ${COSMUX_HOME}
else
  echo "Using existing home ${COSMUX_HOME}"
fi

MINID_BIN=$(which cosmux)

#${MINID_BIN} -v -cmt-home=${COSMUX_HOME} &> ${COSMUX_HOME}/logs &
${MINID_BIN}  -cmt-home=${COSMUX_HOME} 2>&1 | tee ${COSMUX_HOME}/cosmux.log
