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

# sdk-app keys
KEY_NODE=carol
MNEMONIC_NODE="wagon angry enforce security average fat exclude stable control below law valley cable giggle spawn round dance absent comic snow urban clerk hobby sing"
KEY_ALICE=alice
MNEMONIC_ALICE="mammal accuse rapid blur fresh scissors attack wet one begin reduce arch winner noodle quick achieve quick hard olive must pattern tornado wise winter"
KEY_BOB=bob
MNEMONIC_BOB="adjust absurd witness inner differ click system option decline hurt fee supreme transfer diesel industry sniff use material sweet few multiply october pass eternal"

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


echo ${MNEMONIC_ALICE} | ${APP_BIN} keys add ${KEY_ALICE} --home ${NODE_DIR} --keyring-backend test --recover
echo ${MNEMONIC_BOB} | ${APP_BIN} keys add ${KEY_BOB} --home ${NODE_DIR} --keyring-backend test --recover
echo ${MNEMONIC_NODE} | ${APP_BIN} keys add ${KEY_NODE} --home ${NODE_DIR} --keyring-backend test --recover


${APP_BIN} init ${MONIKER} --chain-id ${CHAIN_ID} --home ${NODE_DIR}


# update genesis
${APP_BIN} genesis add-genesis-account ${KEY_NODE} 10000000stake --keyring-backend test --home ${NODE_DIR}  --chain-id ${CHAIN_ID}

# create default validator
${APP_BIN} genesis gentx ${KEY_NODE} 1000000stake --chain-id ${CHAIN_ID} --home ${NODE_DIR} --keyring-backend test
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
LOG_LEVEL="bank:trace,*:debug" # switch to trace to see panic messages and rich and all debug msgs
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
