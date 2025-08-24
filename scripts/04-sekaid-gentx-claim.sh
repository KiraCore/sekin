#!/bin/env bash
curl -X POST http://localhost:8181/api/execute \
-H "Content-Type: application/json" \
-d '{
    "command": "gentx-claim",
    "args": {
        "address": "genesis",
        "keyring-backend": "test",
        "moniker": "GENESIS VALIDATOR",
        "home": "/sekai"
    }
}'
