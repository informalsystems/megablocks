#!/usr/bin/env bash


set -eux

MINID_BIN=$(which minid)
CHAIN_ID="sdk-app-2"

# Node IP address
NODE_IP="127.0.0.1"


# Validator moniker
MONIKER="minid"

# Validator directory
if [ -z ${1+x} ] ; then
  NODE_DIR=${HOME}/.${MONIKER}
else
NODE_DIR=$1
fi

echo Home is ${NODE_DIR}

# sdk-app key
NODE_KEY=alice

APP_NAME="minid"
APP_BIN=$(which minid)
# BIN_DIR="${HOME}/Development/Informal/cosmos/megablocks/app/sdk-chain-a/cmd/minid"
# APP_BIN=${BIN_DIR}/${APP_NAME}

# Clean start
pkill -f ${APP_NAME} &> /dev/null || true
rm -rf ${NODE_DIR}

## configure


# configure minid
${APP_BIN}  config set client chain-id ${CHAIN_ID} --home ${NODE_DIR}
${APP_BIN}  config set client keyring-backend test --home ${NODE_DIR}

${APP_BIN} keys add ${NODE_KEY} --home ${NODE_DIR}


${APP_BIN} init ${MONIKER} --chain-id ${CHAIN_ID} --home ${NODE_DIR}


# update genesis
${APP_BIN} genesis add-genesis-account ${NODE_KEY} 10000000stake --keyring-backend test --home ${NODE_DIR}

# create default validator
${APP_BIN} genesis gentx ${NODE_KEY} 1000000stake --chain-id ${CHAIN_ID} --home ${NODE_DIR}
${APP_BIN} genesis collect-gentxs --gentx-dir ${NODE_DIR}/config/gentx/ --home ${NODE_DIR}

# Fix SDK issue generating initial height with wrong json data-type
sed -i -e 's/"initial_height": 1/"initial_height": "1"/g' ${NODE_DIR}/config/genesis.json

NODE_GRPC_PORT=9091
NODE_RPC_PORT=26657
NODE_PORT=26655
NODE_P2P_PORT=26656

NODE_IP="127.0.0.1"

RPC_ADDRESS="tcp://${NODE_IP}:${NODE_RPC_PORT}"
GRPC_ADDRESS="${NODE_IP}:${NODE_GRPC_PORT}"
LISTEN_ADDRESS="unix:///tmp/mind.sock"
#LISTEN_ADDRESS="tcp://${NODE_IP}:${NODE_PORT}"
P2P_ADDRESS="tcp://${NODE_IP}:${NODE_P2P_PORT}"
LOG_LEVEL="trace" # switch to trace to see panic messages and rich and all debug msgs
#LOG_LEVEL="--log_level info"
ENABLE_WEBGRPC="false"

# Without integrated cometBFT
WITH_COMET_MOCK="false"

${APP_BIN} start \
    --home ${NODE_DIR} \
    --rpc.laddr ${RPC_ADDRESS} \
    --grpc.address ${GRPC_ADDRESS} \
    --address ${LISTEN_ADDRESS} \
    --p2p.laddr ${P2P_ADDRESS} \
    --minimum-gas-prices="0stake" \
    --grpc-web.enable=${ENABLE_WEBGRPC} \
    --log_level=${LOG_LEVEL} \
    --with-comet=${WITH_COMET_MOCK} | tee ${NODE_DIR}/sdk-chain-a.log
