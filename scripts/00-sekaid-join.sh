#!/bin/bash
# Join an existing KIRA network as a new node
# Usage: ./00-sekaid-join.sh <RPC_NODE_IP> [OPTIONS]
#
# Required:
#   RPC_NODE_IP    IP address of the trusted RPC node to join from
#
# Optional environment variables:
#   MNEMONIC       24-word BIP39 mnemonic (will prompt if not set)
#   MONIKER        Node moniker (default: node)
#   STATESYNC      Enable state sync (default: false)
#   PRUNE          Pruning mode: default|nothing|everything|custom (default: nothing)

set -e

RPC_NODE="${1:?Error: RPC node IP address required. Usage: $0 <RPC_NODE_IP>}"
RPC_PORT="${RPC_PORT:-26657}"
MONIKER="${MONIKER:-node}"
STATESYNC="${STATESYNC:-false}"
PRUNE="${PRUNE:-nothing}"
HOME_DIR="${HOME_DIR:-/sekai}"

# Build scaller flags
FLAGS="--rpc-node ${RPC_NODE}:${RPC_PORT} --moniker ${MONIKER} --home ${HOME_DIR} --prune ${PRUNE}"

if [ "${STATESYNC}" = "true" ]; then
    FLAGS="${FLAGS} --statesync"
fi

# Read mnemonic from env or prompt
if [ -z "${MNEMONIC}" ]; then
    echo "Enter your 24-word mnemonic (input hidden):"
    read -rs MNEMONIC
fi

echo "Joining network via RPC node: ${RPC_NODE}:${RPC_PORT}"
echo "${MNEMONIC}" | docker compose exec -T sekai scaller join ${FLAGS}
