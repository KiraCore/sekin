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

# Production Setup (WireGuard)

Secure remote signing across separate machines using WireGuard VPN tunnel.

## Architecture (Production)

```
┌─────────────────────────────────────────────────────────────────┐
│ VALIDATOR MACHINE                                               │
│                                                                 │
│   compose.yml (sekai, interx services, syslog-ng)              │
│   └── sekai: priv_validator_laddr = "tcp://0.0.0.0:26659"      │
│                                                                 │
│   wireguard-compose.yml (WireGuard container)                  │
│   └── connects to kiranet, routes 10.200.0.0/24                │
│                                                                 │
│   WireGuard IP: 10.200.0.1                                     │
│                                                                 │
└───────────────────────────┬─────────────────────────────────────┘
                            │ WireGuard UDP :51820
                            │ Encrypted tunnel
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ SIGNER MACHINE                                                  │
│                                                                 │
│   signer-compose.yml (WireGuard + TMKMS)                       │
│   ├── wireguard: routes to 10.200.0.1                          │
│   └── tmkms: connects to tcp://10.200.0.1:26659                │
│                                                                 │
│   WireGuard IP: 10.200.0.2                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
sekin/
├── compose.yml                          # Main services (uncomment port 26659)
├── tmkms/
│   ├── wireguard-compose.yml            # WireGuard for validator machine
│   ├── signer-compose.yml               # TMKMS + WireGuard for signer machine
│   ├── tmkms.toml.example               # TMKMS config template
│   ├── wireguard/
│   │   ├── validator/
│   │   │   └── wg0.conf.example         # Validator WireGuard config
│   │   └── signer/
│   │       └── wg0.conf.example         # Signer WireGuard config
│   ├── secrets/                         # Keys (gitignored)
│   │   ├── priv_validator_key.json
│   │   └── kms-identity.key
│   └── state/                           # State files (gitignored)
```

## Step-by-Step Setup

### On Validator Machine

#### 1. Generate WireGuard Keys

```bash
# Generate validator private key
wg genkey > validator_private.key

# Derive public key
cat validator_private.key | wg pubkey > validator_public.key

# Save these securely - you'll need them for both configs
cat validator_private.key
cat validator_public.key
```

#### 2. Configure WireGuard

```bash
cd sekin/tmkms
cp wireguard/validator/wg0.conf.example wireguard/validator/wg0.conf
```

Edit `wireguard/validator/wg0.conf`:
```ini
[Interface]
PrivateKey = <VALIDATOR_PRIVATE_KEY>
Address = 10.200.0.1/24
ListenPort = 51820

[Peer]
PublicKey = <SIGNER_PUBLIC_KEY>
AllowedIPs = 10.200.0.2/32
Endpoint = <SIGNER_PUBLIC_IP>:51820
PersistentKeepalive = 25
```

#### 3. Enable Remote Signer Port

Uncomment port 26659 in `compose.yml`:

```yaml
ports:
  # ... other ports ...
  - "127.0.0.1:26659:26659"         # TMKMS remote signer (uncomment for external TMKMS)
```

#### 4. Configure sekaid for Remote Signer

```bash
docker run --rm \
  -v $(pwd)/sekai:/sekai \
  alpine sed -i 's/priv_validator_laddr = ""/priv_validator_laddr = "tcp:\/\/0.0.0.0:26659"/' \
  /sekai/config/config.toml
```

#### 5. Start Services

```bash
# Start main services
docker compose up -d

# Start WireGuard
docker compose -f tmkms/wireguard-compose.yml up -d
```

### On Signer Machine

#### 1. Generate WireGuard Keys

```bash
# Generate signer private key
wg genkey > signer_private.key

# Derive public key
cat signer_private.key | wg pubkey > signer_public.key

# Save these securely
cat signer_private.key
cat signer_public.key
```

#### 2. Clone Repository and Setup

```bash
git clone https://github.com/kiracore/sekin.git
cd sekin/tmkms
```

#### 3. Configure WireGuard

```bash
cp wireguard/signer/wg0.conf.example wireguard/signer/wg0.conf
```

Edit `wireguard/signer/wg0.conf`:
```ini
[Interface]
PrivateKey = <SIGNER_PRIVATE_KEY>
Address = 10.200.0.2/24
ListenPort = 51820

[Peer]
PublicKey = <VALIDATOR_PUBLIC_KEY>
AllowedIPs = 10.200.0.1/32
Endpoint = <VALIDATOR_PUBLIC_IP>:51820
PersistentKeepalive = 25
```

#### 4. Setup TMKMS Keys

Import validator key (copy from validator machine first):

```bash
mkdir -p secrets state

# Copy priv_validator_key.json from validator to signer machine first
docker compose -f signer-compose.yml build tmkms

docker run --rm \
  -v $(pwd)/secrets:/tmkms \
  sekin-tmkms softsign import \
  /path/to/original/priv_validator_key.json \
  /tmkms/priv_validator_key.json
```

Generate KMS identity key:

```bash
docker run --rm \
  -v $(pwd)/secrets:/out \
  sekin-tmkms init /out -n kira

docker run --rm \
  -v $(pwd)/secrets:/out \
  alpine mv /out/secrets/kms-identity.key /out/kms-identity.key

docker run --rm \
  -v $(pwd)/secrets:/out \
  alpine sh -c "rm -rf /out/secrets /out/schema /out/state /out/tmkms.toml"
```

#### 5. Configure TMKMS

```bash
cp tmkms.toml.example tmkms.toml
```

Edit `tmkms.toml` - update chain ID and address:

```toml
[[chain]]
id = "your-chain-id"
key_format = { type = "bech32", account_key_prefix = "kirapub", consensus_key_prefix = "kiravalconspub" }
state_file = "/tmkms/state/your-chain-id-consensus.json"

[[providers.softsign]]
chain_ids = ["your-chain-id"]
key_type = "consensus"
path = "/tmkms/secrets/priv_validator_key.json"

[[validator]]
chain_id = "your-chain-id"
addr = "tcp://10.200.0.1:26659"   # Validator via WireGuard tunnel
secret_key = "/tmkms/secrets/kms-identity.key"
protocol_version = "v0.38"
reconnect = true
```

#### 6. Start TMKMS

```bash
docker compose -f signer-compose.yml up -d
```

### Verify Connection

On signer machine, check TMKMS logs:

```bash
docker compose -f signer-compose.yml logs -f tmkms
```

Expected output:
```
INFO tmkms::session: [chain-id@tcp://10.200.0.1:26659] signed Proposal:...
INFO tmkms::session: [chain-id@tcp://10.200.0.1:26659] signed Prevote:...
INFO tmkms::session: [chain-id@tcp://10.200.0.1:26659] signed Precommit:...
```

## Firewall Rules

### Validator Machine

```bash
# Allow WireGuard
ufw allow 51820/udp

# Allow P2P
ufw allow 26656/tcp

# Block direct access to remote signer port
# (only accessible via WireGuard tunnel)
ufw deny 26659/tcp
```

### Signer Machine

```bash
# Allow WireGuard only
ufw allow 51820/udp

# Deny everything else from internet
ufw default deny incoming
```

## Security Considerations

- Signer machine should be air-gapped or heavily firewalled
- Only allow WireGuard traffic between validator and signer
- Backup `priv_validator_key.json` securely (encrypted, offline)
- Never expose port 26659 to public internet
- Use strong WireGuard keys (256-bit)
- Keep WireGuard and TMKMS updated
- Monitor TMKMS logs for unauthorized connection attempts

---

# TODO - Production WireGuard Setup

## Status: WIP - Not Fully Working

### What Works
- [x] Local development setup (TMKMS + sekaid on same Docker network via kiranet)
- [x] Terraform config for deploying validator/signer VMs (`tmkms/terraform/`)
- [x] WireGuard tunnel establishment (handshakes successful, ping works 10.200.0.1 <-> 10.200.0.2)
- [x] TMKMS key import from sekaid format
- [x] WireGuard keys generation

### What Doesn't Work
- [ ] TMKMS connecting to sekaid via WireGuard tunnel

## Root Cause

Docker bridge networking isolates containers from the host's WireGuard interface (wg0). Traffic arriving on wg0:26659 cannot reach sekaid running in a Docker container with bridge networking.

### Tested Approaches (Failed)

1. **wireguard-compose.yml (Docker WireGuard on kiranet)**
   - WireGuard container joined kiranet bridge
   - Could ping between WireGuard containers
   - Could NOT route wg0 traffic to sekai container port 26659
   - Result: "Connection refused"

2. **iptables NAT forwarding in WireGuard container**
   ```bash
   iptables -t nat -A PREROUTING -i wg0 -p tcp --dport 26659 -j DNAT --to-destination 172.17.0.1:26659
   iptables -t nat -A POSTROUTING -j MASQUERADE
   ```
   - Traffic reached sekaid but got "protocol error: I/O error"
   - NAT may be corrupting the privval protocol

3. **Host WireGuard + Docker port proxy**
   - WireGuard on host (not Docker)
   - Docker port proxy listens on 0.0.0.0:26659
   - BUT doesn't accept connections arriving on wg0 interface
   - Result: "Connection refused" even locally on validator

## Solution Options

### Option A: Host Networking for Sekai (Recommended)

Sekai container uses `network_mode: host`, listens directly on all host interfaces including wg0.

```yaml
services:
  sekai:
    network_mode: host
    # NO ports: section needed
    volumes:
      - ./sekai:/sekai
```

**Pros:** Simple, no routing needed
**Cons:** Sekai loses Docker network isolation

### Option B: Proper iptables Forwarding

Add iptables rules to WireGuard container for full traffic forwarding:

```ini
# In wg0.conf PostUp
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth+ -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth+ -j MASQUERADE
```

Also need `AllowedIPs = 0.0.0.0/0` in peer config.

**Note:** Known issue - can't reach Docker host from inside VPN.

## TODO Items

### High Priority

1. [ ] **Fix protocol error** - Investigate if NAT breaks privval protocol or if Tendermint version mismatch (sekaid uses 0.37.2, TMKMS supports v0.34/v0.38)

2. [ ] **Test host networking for sekai** - Create compose variant with `network_mode: host`

3. [ ] **Update sekaidCaller** - Verify works with host networking

### Medium Priority

4. [ ] **Update compose files** - Add TMKMS-ready compose variant

5. [ ] **Security review** - Host networking reduces isolation, document implications

## Corrected Architecture

```
VALIDATOR MACHINE
├── WireGuard on HOST (not Docker)
│   └── wg0: 10.200.0.1/24
├── sekaid with network_mode: host
│   └── Listens on 0.0.0.0:26659 (includes wg0)
└── Other services on kiranet (bridge)

SIGNER MACHINE
├── signer-compose.yml
│   ├── wireguard container
│   │   └── wg0: 10.200.0.2/24
│   └── tmkms (network_mode: service:wireguard)
│       └── Connects to tcp://10.200.0.1:26659
```

## Testing Notes (2025-12-12)

- Validator: AWS eu-central-1, Signer: AWS eu-west-1
- WireGuard tunnel: **WORKING** (20ms latency)
- sekaid local signing: **WORKING** (900+ blocks)
- TMKMS via WireGuard: **FAILED** ("protocol error: I/O error")
