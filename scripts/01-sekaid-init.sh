#!/bin/env bash
curl -X POST http://localhost:8181/api/execute -H "Content-Type: application/json" \
-d '{
    "command": "init",
    "args": {
        "chain-id": "testnet-1",
        "moniker": "KIRA TEST LOCAL VALIDATOR NODE",
        "home": "/sekai",
        "log_format": "",  
        "log_level": "",   
        "trace": false,
        "overwrite": true
    }
}'

