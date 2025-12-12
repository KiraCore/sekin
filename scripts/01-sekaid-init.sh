#!/bin/bash
# Initialize a new genesis validator node
# Usage: ./01-sekaid-init.sh [CHAIN_ID] [MONIKER]
#
# Optional:
#   CHAIN_ID    Chain identifier (default: testnet-1)
#   MONIKER     Node moniker (default: KIRA TEST LOCAL VALIDATOR NODE)

set -e

CHAIN_ID="${1:-testnet-1}"
MONIKER="${2:-KIRA TEST LOCAL VALIDATOR NODE}"
HOME_DIR="${HOME_DIR:-/sekai}"

echo "Initializing sekaid with chain-id: ${CHAIN_ID}, moniker: ${MONIKER}"

docker compose exec sekai sekaid init "${MONIKER}" \
    --chain-id "${CHAIN_ID}" \
    --home "${HOME_DIR}" \
    --overwrite
