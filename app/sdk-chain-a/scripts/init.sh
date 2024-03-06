#!/usr/bin/env bash

# Home directory
if [ -z ${1+x} ] ; then
  NODE_DIR=${HOME}/.${MONIKER}
else
NODE_DIR=$1
fi

rm -r ${NODE_DIR} || true
MINID_BIN=$(which minid)
CHAIN_ID="sdk-app-2"
MONIKER="minid"

# configure minid
$MINID_BIN config set client chain-id ${CHAIN_ID} --home ${NODE_DIR}
$MINID_BIN config set client keyring-backend test --home ${NODE_DIR}
$MINID_BIN keys add alice --home ${NODE_DIR}
$MINID_BIN keys add bob --home ${NODE_DIR}
$MINID_BIN init ${MONIKER} --chain-id ${CHAIN_ID} --default-denom stake --home ${NODE_DIR}
# update genesis
$MINID_BIN genesis add-genesis-account alice 10000000stake --keyring-backend test --home ${NODE_DIR}
$MINID_BIN genesis add-genesis-account bob 1000stake --keyring-backend test --home ${NODE_DIR}
# create default validator
$MINID_BIN genesis gentx alice 1000000stake --chain-id ${CHAIN_ID} --home ${NODE_DIR}
$MINID_BIN genesis collect-gentxs --home ${NODE_DIR}

# Fix SDK issue generating initial height with wrong json data-type
sed -i -e 's/"initial_height": 1/"initial_height": "1"/g' ${NODE_DIR}/config/genesis.json
