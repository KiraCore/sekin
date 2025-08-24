#!/bin/env bash

curl -X POST http://localhost:8181/api/execute -H "Content-Type: application/json" \
-d '{
    "command": "add-genesis-account",
    "args": {
        "address": "genesis",
        "coins": ["300000000000000ukex"],
        "keyring-backend": "test",
        "home": "/sekai",
        "log_format": "",  
        "log_level": "",   
        "trace": false     
    }
}'

