#!/bin/bash
# Add an account to genesis with initial token allocation
# Usage: ./03-sekaid-add-genesis-account.sh [KEY_NAME] [COINS]
#
# Optional:
#   KEY_NAME          Name of the key to add (default: genesis)
#   COINS             Initial coin allocation (default: 300000000000000ukex)
#   KEYRING_BACKEND   Keyring backend: test|file|os (default: test)

set -e

KEY_NAME="${1:-genesis}"
COINS="${2:-300000000000000ukex}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
HOME_DIR="${HOME_DIR:-/sekai}"

echo "Adding genesis account '${KEY_NAME}' with ${COINS}"

docker compose exec sekai sekaid add-genesis-account "${KEY_NAME}" "${COINS}" \
    --keyring-backend "${KEYRING_BACKEND}" \
    --home "${HOME_DIR}"
