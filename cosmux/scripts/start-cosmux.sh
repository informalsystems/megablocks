#!/usr/bin/env bash
set -eux

# Home directory
if [ -z ${1+x} ] ; then
  COSMUX_HOME="${HOME}/.cosmux"
else
  COSMUX_HOME=$1
fi

MINID_BIN=$(which cosmux)

#${MINID_BIN} -v -cmt-home=${COSMUX_HOME} &> ${COSMUX_HOME}/logs &
${MINID_BIN} -v -cmt-home=${COSMUX_HOME} 2>&1 | tee ${COSMUX_HOME}/cosmux.log