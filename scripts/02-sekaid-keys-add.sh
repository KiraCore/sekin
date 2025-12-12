#!/bin/bash
# Add a new key to the keyring
# Usage: ./02-sekaid-keys-add.sh [KEY_NAME]
#
# Optional:
#   KEY_NAME          Name for the key (default: genesis)
#   KEYRING_BACKEND   Keyring backend: test|file|os (default: test)

set -e

KEY_NAME="${1:-genesis}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
HOME_DIR="${HOME_DIR:-/sekai}"

echo "Adding key '${KEY_NAME}' with keyring backend '${KEYRING_BACKEND}'"

docker compose exec sekai sekaid keys add "${KEY_NAME}" \
    --keyring-backend "${KEYRING_BACKEND}" \
    --home "${HOME_DIR}"
