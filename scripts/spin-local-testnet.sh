#!/bin/bash
# Test script: Setup a local genesis network
# Usage: ./spin-local-testnet.sh
#
# This script runs all steps to create a local test network:
# 1. Initialize sekaid
# 2. Add genesis key
# 3. Add genesis account
# 4. Claim validator role
# 5. Start the node

set -e

CONTAINER="${CONTAINER:-sekin-sekai-1}"
CHAIN_ID="${CHAIN_ID:-testnet-1}"
KEY_NAME="${KEY_NAME:-genesis}"
MONIKER="${MONIKER:-Genesis}"
COINS="${COINS:-300000000000000ukex}"

echo "=== Setting up local genesis network ==="
echo "Container: ${CONTAINER}"
echo "Chain ID: ${CHAIN_ID}"
echo "Key Name: ${KEY_NAME}"
echo "Moniker: ${MONIKER}"
echo ""

echo ">>> Step 1: Initialize sekaid"
docker exec ${CONTAINER} /scaller init --chain-id "${CHAIN_ID}" --moniker "${MONIKER}"
echo ""

echo ">>> Step 2: Add genesis key"
docker exec ${CONTAINER} /scaller keys-add --name "${KEY_NAME}"
echo ""

echo ">>> Step 3: Add genesis account"
docker exec ${CONTAINER} /scaller add-genesis-account --name "${KEY_NAME}" --coins "${COINS}"
echo ""

echo ">>> Step 4: Claim validator role"
docker exec ${CONTAINER} /scaller gentx-claim --name "${KEY_NAME}" --moniker "${MONIKER}"
echo ""

echo ">>> Step 5: Start sekaid"
docker exec ${CONTAINER} /scaller start
