#!/bin/env bash
curl -X POST http://localhost:8181/api/execute \
-H "Content-Type: application/json" \
-d '{
    "command": "start",
    "args": {
        "home": "/sekai"
    }
}'
