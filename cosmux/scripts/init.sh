#!/usr/bin/env bash

# Home directory
if [ -z ${1+x} ] ; then
  COSMUX_HOME="${HOME}/.cosmux"
else
  COSMUX_HOME=$1
fi

rm -r ${COSMUX_HOME} || true
go run github.com/cometbft/cometbft/cmd/cometbft@v0.38.5 init --home ${COSMUX_HOME}