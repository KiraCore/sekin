
curl -X POST "http://localhost:8282/api/execute" \
     -H "Content-Type: application/json" \
     -d '{
            "command": "join",
            "args": {
                "ip": "0.0.0.0",
                "interx_port": 11000,
                "rpc_port": 26657,
                "p2p_port": 26656,
                "mnemonic": "bargain erosion electric skill extend aunt unfold cricket spice sudden insane shock purpose trumpet holiday tornado fiction check pony acoustic strike side gold resemble"
            }
         }'
