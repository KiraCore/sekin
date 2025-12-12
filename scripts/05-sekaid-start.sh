#!/bin/bash
# Start the sekaid node
# Usage: ./05-sekaid-start.sh

set -e

HOME_DIR="${HOME_DIR:-/sekai}"

echo "Starting sekaid..."

docker compose exec sekai scaller start --home "${HOME_DIR}"
