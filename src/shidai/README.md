# Shidai

## Build and run

```
docker compose   -f ./sekin-compose.yml build && docker compose   -f ./sekin-compose.yml up
```

## Join

`ip`, `interxPort`, `rpcPort`. `p2pPort` are values from node you joining to

`mnemonic` is your master mnemonic

`sekaiAddress`, `interxAddress` are hostnames of the sekaid and interx containers (can be container name or `hostname` value from docker compose file)

`enableInterx` is a check if false, initializing node without interx

```
curl -X POST http://localhost:8282/api/execute -H "Content-Type: application/json" -d '{
  "command": "join",
  "args": {
    "ip": "ip of the node to join",
    "interxPort": 11000,
    "rpcPort": 26657,
    "p2pPort": 26656,
    "sekaiAddress": "sekai.local",
    "interxAddress": "interx.local",
    "enableInterx": true,
    "mnemonic": "bargain erosion electric skill extend aunt unfold cricket spice sudden insane shock purpose trumpet holiday tornado fiction check pony acoustic strike side gold resemble"
  }
}'

```

shidai initializing node with default values (interx=11000,rpc=26657,p2p=26656, etc...)