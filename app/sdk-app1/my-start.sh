#! /bin/bash

set -eux

# User balance of stake tokens
USER_COINS="100000000000stake"
# Amount of stake tokens staked
STAKE="100000000stake"
# Node IP address
NODE_IP="127.0.0.1"

# Home directory
HOME_DIR="/Users/bernd"

# Validator moniker
MONIKER="sdk-app"

# Validator directory
NODE_DIR=${HOME_DIR}/.${MONIKER}

# sdk-app key
NODE_KEY=${MONIKER}-key

APP_NAME="sdk-app1"
BIN_DIR="${HOME_DIR}/Development/Informal/cosmos/megablocks/app/sdk-app1/cmd/sdk-appd"
APP_BIN=${BIN_DIR}/${APP_NAME}

# Clean start
pkill -f ${APP_NAME} &> /dev/null || true
rm -rf ${NODE_DIR}

# Build file and node directory structure
${APP_BIN} init $MONIKER --chain-id sdkapp --home ${NODE_DIR}
#jq ".app_state.gov.params.voting_period = \"20s\"  | .app_state.staking.params.unbonding_time = \"86400s\"" \
#   ${NODE_DIR}/config/genesis.json > \
#   ${NODE_DIR}/edited_genesis.json && mv ${NODE_DIR}/edited_genesis.json ${NODE_DIR}/config/genesis.json

sleep 1

# Create account keypair
${APP_BIN} keys add $NODE_KEY --home ${NODE_DIR} --keyring-backend test --output json > ${NODE_DIR}/${NODE_KEY}.json 2>&1
sleep 1

# Add stake to user
NODE_ACCOUNT_ADDR=$(jq -r '.address' ${NODE_DIR}/${NODE_KEY}.json)
${APP_BIN} genesis add-genesis-account $NODE_ACCOUNT_ADDR $USER_COINS --home ${NODE_DIR} --keyring-backend test
sleep 1


# Stake 1/1000 user's coins
${APP_BIN} genesis gentx $NODE_KEY $STAKE --chain-id provider --home ${NODE_DIR} --keyring-backend test --moniker $MONIKER
sleep 1

${APP_BIN} genesis collect-gentxs --home ${NODE_DIR} --gentx-dir ${NODE_DIR}/config/gentx/
sleep 1

NODE_GRPC_PORT=9091
NODE_RPC_PORT=26658
NODE_PORT=26655
NODE_P2P_PORT=26656

sed -i -r "/node =/ s/= .*/= \"tcp:\/\/${NODE_IP}:${NODE_RPC_PORT}\"/" ${NODE_DIR}/config/client.toml
sed -i -r 's/timeout_commit = "5s"/timeout_commit = "3s"/g' ${NODE_DIR}/config/config.toml
sed -i -r 's/timeout_propose = "3s"/timeout_propose = "1s"/g' ${NODE_DIR}/config/config.toml

# Start gaia
${APP_BIN} start \
    --home ${NODE_DIR} \
    --rpc.laddr tcp://${NODE_IP}:${NODE_RPC_PORT} \
    --grpc.address ${NODE_IP}:${NODE_GRPC_PORT} \
    --address tcp://${NODE_IP}:${NODE_PORT} \
    --p2p.laddr tcp://${NODE_IP}:${NODE_P2P_PORT} \
    --grpc-web.enable=false &> ${NODE_DIR}/logs &
