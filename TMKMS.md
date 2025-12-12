# TMKMS Integration Guide

TMKMS (Tendermint Key Management System) enables remote validator signing. The validator private key never exists on the sekaid node - it lives only in the TMKMS container.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ SEKAID CONTAINER                                                │
│                                                                 │
│   sekaid                                                        │
│   └── config.toml: priv_validator_laddr = "tcp://0.0.0.0:26659" │
│                                                                 │
│   NO priv_validator_key.json (key never here)                   │
│                                                                 │
└───────────────────────────┬─────────────────────────────────────┘
                            │ TCP :26659
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ TMKMS CONTAINER                                                 │
│                                                                 │
│   tmkms                                                         │
│   ├── tmkms.toml (config)                                       │
│   ├── secrets/priv_validator_key.json (ed25519 consensus key)   │
│   ├── secrets/kms-identity.key (connection identity)            │
│   └── state/*.json (double-sign prevention)                     │
│                                                                 │
│   Connects TO sekaid at tcp://sekai.local:26659                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

# Development Setup

## Prerequisites

- Docker and Docker Compose
- sekin repository cloned

## Directory Structure

```
sekin/
├── tmkms.Dockerfile          # Builds TMKMS image
├── tmkms/
│   ├── tmkms.toml            # Active config (gitignored)
│   ├── tmkms.toml.example    # Config template
│   ├── secrets/              # Keys (gitignored)
│   │   ├── priv_validator_key.json
│   │   └── kms-identity.key
│   └── state/                # State files (gitignored)
│       └── testnet-1-consensus.json
└── dev-compose.yml           # TMKMS service definition
```

## Step-by-Step Setup

### 1. Build TMKMS Image

```bash
docker compose -f dev-compose.yml build tmkms
```

### 2. Start Containers

```bash
docker compose -f dev-compose.yml up -d syslog-ng sekai tmkms
```

### 3. Initialize Local Testnet (First Time Only)

Initialize the node without starting it:

```bash
CONTAINER=sekin-sekai-1

# Initialize sekaid
docker exec $CONTAINER /scaller init --chain-id testnet-1 --moniker Genesis

# Create genesis account key
docker exec $CONTAINER /scaller keys-add --name genesis

# Fund the account
docker exec $CONTAINER /scaller add-genesis-account --name genesis

# Claim validator role
docker exec $CONTAINER /scaller gentx-claim --name genesis --moniker Genesis
```

### 4. Import Validator Key to TMKMS

Convert sekaid's key format to TMKMS format:

```bash
docker run --rm \
  -v $(pwd)/sekai:/sekai \
  -v $(pwd)/tmkms/secrets:/tmkms \
  sekin-tmkms softsign import \
  /sekai/config/priv_validator_key.json \
  /tmkms/priv_validator_key.json
```

### 5. Generate KMS Identity Key

```bash
docker run --rm \
  -v $(pwd)/tmkms/secrets:/out \
  sekin-tmkms init /out -n cosmoshub
```

Then move the identity key:

```bash
docker run --rm \
  -v $(pwd)/tmkms/secrets:/out \
  alpine mv /out/secrets/kms-identity.key /out/kms-identity.key
```

Clean up extra files:

```bash
docker run --rm \
  -v $(pwd)/tmkms/secrets:/out \
  alpine sh -c "rm -rf /out/secrets /out/schema /out/state /out/tmkms.toml"
```

### 6. Create TMKMS Config

```bash
cp tmkms/tmkms.toml.example tmkms/tmkms.toml
```

Edit `tmkms/tmkms.toml` - update chain ID if needed:

```toml
# TMKMS Configuration for KIRA Network

## Chain Configuration
[[chain]]
id = "testnet-1"
key_format = { type = "bech32", account_key_prefix = "kirapub", consensus_key_prefix = "kiravalconspub" }
state_file = "/tmkms/state/testnet-1-consensus.json"

## Signing Provider Configuration - Software Signer
[[providers.softsign]]
chain_ids = ["testnet-1"]
key_type = "consensus"
path = "/tmkms/secrets/priv_validator_key.json"

## Validator Configuration
[[validator]]
chain_id = "testnet-1"
addr = "tcp://sekai.local:26659"
secret_key = "/tmkms/secrets/kms-identity.key"
protocol_version = "v0.38"
reconnect = true
```

### 7. Configure sekaid for Remote Signer

Set `priv_validator_laddr` in config.toml:

```bash
docker run --rm \
  -v $(pwd)/sekai:/sekai \
  alpine sed -i 's/priv_validator_laddr = ""/priv_validator_laddr = "tcp:\/\/0.0.0.0:26659"/' \
  /sekai/config/config.toml
```

Verify:

```bash
docker run --rm \
  -v $(pwd)/sekai:/sekai \
  alpine grep priv_validator_laddr /sekai/config/config.toml
```

Expected output:
```
priv_validator_laddr = "tcp://0.0.0.0:26659"
```

### 8. Start TMKMS and sekaid

```bash
# Restart TMKMS to pick up new config
docker compose -f dev-compose.yml restart tmkms

# Start sekai container
docker compose -f dev-compose.yml up -d sekai

# Start sekaid
docker exec sekin-sekai-1 /scaller start
```

### 9. Verify TMKMS is Signing

Check TMKMS logs:

```bash
docker logs sekin-tmkms-1 -f
```

Expected output showing signing:
```
INFO tmkms::session: [testnet-1@tcp://sekai.local:26659] signed Proposal:ABC123 at h/r/s 51/0/0 (0 ms)
INFO tmkms::session: [testnet-1@tcp://sekai.local:26659] signed Prevote:ABC123 at h/r/s 51/0/1 (0 ms)
INFO tmkms::session: [testnet-1@tcp://sekai.local:26659] signed Precommit:ABC123 at h/r/s 51/0/2 (0 ms)
```

## Configuration Reference

### tmkms.toml

| Section | Field | Description |
|---------|-------|-------------|
| `[[chain]]` | `id` | Chain ID (must match sekaid) |
| | `key_format` | Bech32 prefixes for kira |
| | `state_file` | Tracks last signed height/round/step |
| `[[providers.softsign]]` | `chain_ids` | Which chains this key signs for |
| | `key_type` | `consensus` for validator signing |
| | `path` | Path to private key file |
| `[[validator]]` | `chain_id` | Chain ID to connect to |
| | `addr` | sekaid's priv_validator_laddr |
| | `secret_key` | KMS identity key for connection |
| | `protocol_version` | Tendermint protocol (v0.38) |
| | `reconnect` | Auto-reconnect on disconnect |

### sekaid config.toml

| Field | Value | Description |
|-------|-------|-------------|
| `priv_validator_laddr` | `tcp://0.0.0.0:26659` | Listen for remote signer |

## Troubleshooting

### "Connection refused" in TMKMS logs

sekaid isn't running or not listening on port 26659.

```bash
# Check sekaid is running
docker exec sekin-sekai-1 ps aux

# Check priv_validator_laddr is set
docker run --rm -v $(pwd)/sekai:/sekai alpine grep priv_validator_laddr /sekai/config/config.toml
```

### "bad encoding" when loading key

Key wasn't imported properly. Re-run the import:

```bash
docker run --rm \
  -v $(pwd)/sekai:/sekai \
  -v $(pwd)/tmkms/secrets:/tmkms \
  sekin-tmkms softsign import \
  /sekai/config/priv_validator_key.json \
  /tmkms/priv_validator_key.json
```

### "endpoint connection timed out" on sekaid start

TMKMS isn't running or can't reach sekaid. Check:

```bash
# Is TMKMS running?
docker ps | grep tmkms

# Are they on the same network?
docker network inspect kiranet
```

### Chain ID mismatch

Ensure `id` in tmkms.toml matches the chain ID used in sekaid init:

```bash
# Check sekaid chain ID
docker run --rm -v $(pwd)/sekai:/sekai alpine cat /sekai/config/genesis.json | grep chain_id
```

## Files Reference

### tmkms/secrets/priv_validator_key.json

TMKMS format (base64 raw key, 44 bytes):
```
<base64-encoded-ed25519-private-key>
```

### tmkms/secrets/kms-identity.key

Connection identity key (base64 raw key, 44 bytes):
```
<base64-encoded-ed25519-key>
```

### tmkms/state/testnet-1-consensus.json

Double-sign prevention state:
```json
{
  "height": "57",
  "round": "0",
  "step": 3,
  "block_id": "..."
}
```

---

# Production Setup

*TODO: Add WireGuard tunnel configuration for secure remote signing across machines.*

## Architecture (Production)

```
┌─────────────────┐      WireGuard VPN      ┌─────────────────┐
│   VALIDATOR     │◄───────────────────────►│     SIGNER      │
│   (Machine A)   │                         │   (Machine B)   │
│                 │                         │                 │
│   sekaid        │                         │   tmkms         │
│   wireguard     │                         │   wireguard     │
└─────────────────┘                         └─────────────────┘
```

## WireGuard Setup

*Coming soon*

## Security Considerations

- TMKMS machine should be air-gapped or heavily firewalled
- Only allow WireGuard traffic between validator and signer
- Backup `priv_validator_key.json` securely (encrypted, offline)
- Never expose port 26659 to public internet
