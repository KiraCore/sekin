#!/bin/bash
# Generate a genesis transaction to claim validator role
# Usage: ./04-sekaid-gentx-claim.sh [KEY_NAME] [MONIKER]
#
# Optional:
#   KEY_NAME          Name of the validator key (default: genesis)
#   MONIKER           Validator moniker (default: GENESIS VALIDATOR)
#   KEYRING_BACKEND   Keyring backend: test|file|os (default: test)

set -e

KEY_NAME="${1:-genesis}"
MONIKER="${2:-GENESIS VALIDATOR}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
HOME_DIR="${HOME_DIR:-/sekai}"

echo "Generating gentx-claim for '${KEY_NAME}' as '${MONIKER}'"

docker compose exec sekai sekaid gentx-claim "${KEY_NAME}" \
    --keyring-backend "${KEYRING_BACKEND}" \
    --moniker "${MONIKER}" \
    --home "${HOME_DIR}"
