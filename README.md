# Sekin

## Overview

Sekin is a complete infrastructure stack for running KIRA blockchain nodes with integrated indexing, cross-chain interaction services, and comprehensive monitoring. It provides a production-ready deployment environment using Docker Compose with automated CI/CD pipelines for seamless updates.

## Architecture

Sekin orchestrates multiple microservices in a containerized environment:

```
┌─────────────────────────────────────────────────────────────┐
│                     Sekin Infrastructure                     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────────┐     │
│  │  Sekai   │  │  Shidai  │  │   Interx Manager      │     │
│  │ (Cosmos) │  │  (Infra) │  │   (P2P/HTTP Server)   │     │
│  └────┬─────┘  └────┬─────┘  └───────────┬───────────┘     │
│       │             │                     │                  │
│  ┌────┴─────────────┴─────────────────────┴───────────┐     │
│  │              Centralized Logging (Syslog-ng)       │     │
│  └────────────────────────────────────────────────────┘     │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Storage    │  │    Proxy     │  │   MongoDB    │      │
│  │  (MongoDB)   │  │  (HTTP/REST) │  │  (Database)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
│  ┌──────────────────┐  ┌──────────────────────────┐         │
│  │ Cosmos Services  │  │  Ethereum Services       │         │
│  │ - Indexer        │  │  - Indexer               │         │
│  │ - Interaction    │  │  - Interaction           │         │
│  └──────────────────┘  └──────────────────────────┘         │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Components

### Core Services

**Sekai (v0.4.13)**
- KIRA blockchain node built on Cosmos SDK
- Provides consensus, state management, and transaction processing
- Ports: 26657 (RPC), 26656 (P2P), 9090 (gRPC), 1317 (REST API)

**Shidai (v0.15.2)**
- Infrastructure management and orchestration service
- Monitors blockchain status and manages container lifecycle
- Provides API endpoint on port 8282
- Built from source in `src/shidai/`

**Syslog-ng (v0.15.2)**
- Centralized logging server for all services
- Collects logs via UDP/TCP on port 514
- Configured with log rotation and retention policies

### Interx Microservices (v0.7.0)

**Manager**
- P2P load balancer and HTTP server
- Handles peer discovery and request routing
- Ports: 8080 (HTTP), 9000 (UDP P2P)

**Proxy**
- Legacy HTTP request converter
- Translates between different API formats
- Port: 11000

**Storage**
- MongoDB-backed storage service
- Provides data persistence layer
- Port: 8880

**Cosmos Indexer**
- Indexes KIRA/Cosmos blockchain data
- Real-time block and transaction indexing
- Port: 8883

**Cosmos Interaction**
- Creates and publishes Cosmos transactions
- Handles transaction signing and broadcasting
- Port: 8884

**Ethereum Indexer**
- Indexes Ethereum blockchain data
- Monitors smart contract events
- Port: 8881

**Ethereum Interaction**
- Creates and publishes Ethereum transactions
- Smart contract interaction layer
- Port: 8882

**MongoDB**
- Database backend for storage service
- Version: 7.0

## Network Topology

All services run on a custom bridge network `kiranet` (10.1.0.0/16):

| Service              | IP Address   | Hostname                    |
|----------------------|--------------|-----------------------------|
| Gateway              | 10.1.0.1     | -                           |
| Syslog-ng            | 10.1.0.2     | syslog-ng.local             |
| Sekai                | 10.1.0.3     | sekai.local                 |
| Manager              | 10.1.0.4     | manager.local               |
| Shidai               | 10.1.0.5     | shidai.local                |
| Proxy                | 10.1.0.10    | proxy.local                 |
| Storage              | 10.1.0.11    | storage.local               |
| Cosmos Indexer       | 10.1.0.12    | cosmos-indexer.local        |
| Cosmos Interaction   | 10.1.0.13    | cosmos-interaction.local    |
| Ethereum Indexer     | 10.1.0.14    | ethereum-indexer.local      |
| Ethereum Interaction | 10.1.0.15    | ethereum-interaction.local  |
| MongoDB              | 10.1.0.16    | mongo.local                 |

## Prerequisites

- Ubuntu 20.04 or later
- Minimum 8GB RAM, 16GB recommended
- 100GB+ available disk space
- Root or sudo access
- Docker Engine 20.10+
- Docker Compose v2.0+

## Installation

### 1. Prepare the Host

Clone the repository and make scripts executable:

```bash
git clone https://github.com/KiraCore/sekin.git
cd sekin
chmod +x ./scripts/*
```

### 2. Bootstrap the Environment

Run the bootstrap script to install all dependencies (Docker, Docker Compose, etc.):

```bash
sudo ./scripts/bootstrap.sh
```

This script will:
- Update system packages
- Install Docker and Docker Compose
- Configure Docker daemon
- Set up required users and permissions

### 3. Deploy the Stack

**Production Deployment:**

```bash
docker compose -f compose.yml up -d
```

**Development Deployment:**

```bash
docker compose -f dev-compose.yml up -d
```

The main differences:
- `compose.yml`: Uses pre-built images from GHCR
- `dev-compose.yml`: Builds images locally from Dockerfiles

### 4. Verify Services

Check that all services are running:

```bash
docker compose ps
```

View logs from all services:

```bash
docker compose logs -f
```

View logs from a specific service:

```bash
docker compose logs -f sekai
# or check the centralized logs
tail -f ./syslog-data/sekai.log
```

## Building Images Independently

Apart from using Docker Compose, you can build Sekai and Interx independently.

### Build Sekai

```bash
./scripts/docker-sekaid-build.sh v0.4.13
```

### Build Interx

```bash
./scripts/docker-interxd-build.sh v0.7.0
```

### Run Containers

**Run Sekai:**

```bash
./scripts/docker-sekaid-run.sh v0.4.13
```

**Run Interx:**

```bash
./scripts/docker-interxd-run.sh v0.7.0
```

## Port Mappings

### External Access (0.0.0.0)

| Port  | Service | Description                    |
|-------|---------|--------------------------------|
| 26657 | Sekai   | RPC (Tendermint)              |
| 26656 | Sekai   | P2P (Tendermint)              |
| 9000  | Manager | P2P UDP                        |
| 11000 | Proxy   | HTTP Proxy                     |
| 8880  | Storage | Storage API                    |
| 8881  | ETH Idx | Ethereum Indexer API          |
| 8882  | ETH Int | Ethereum Interaction API      |
| 8883  | Cos Idx | Cosmos Indexer API            |
| 8884  | Cos Int | Cosmos Interaction API        |

### Localhost Only (127.0.0.1)

| Port  | Service    | Description                 |
|-------|------------|-----------------------------|
| 26658 | Sekai      | ABCI                        |
| 26660 | Sekai      | Prometheus Metrics          |
| 8181  | Sekai      | RPC sCaller                 |
| 1317  | Sekai      | REST API                    |
| 9090  | Sekai      | gRPC                        |
| 8080  | Manager    | HTTP Server                 |
| 8282  | Shidai     | Infrastructure Manager API  |
| 514   | Syslog-ng  | Syslog Server (UDP/TCP)     |

## Configuration

Configuration files for each service are located in their respective directories:

- **Manager**: `./manager/config.yml`
- **Proxy**: `./proxy/config.yml`
- **Storage**: `./worker/sai-storage-mongo/config.yml`
- **Cosmos Indexer**: `./worker/cosmos/sai-cosmos-indexer/config.yml`
- **Cosmos Interaction**: `./worker/cosmos/sai-cosmos-interaction/config.yml`
- **Ethereum Indexer**: `./worker/ethereum/sai-ethereum-indexer/config.json`
- **Ethereum Interaction**: `./worker/ethereum/sai-ethereum-contract-interaction/config.yml`
- **Syslog**: `./config/syslog-ng.conf`, `./config/logrotate.conf`

Sekai and Interx configurations are stored in `./sekai/` and `./interx/` directories (gitignored).

## CI/CD Automation

Sekin uses GitHub Actions for automated image building, signing, and deployment:

### Workflows

**`ci.yml` - Build and Release**
- Triggers on PR merge to `main` branch
- Creates semantic version tags
- Builds and pushes Shidai and Syslog-ng images to GHCR
- Signs images with Cosign (Sigstore)
- Updates `compose.yml` with new versions

**`hook.yml` - Repository Dispatch**
- Triggered by external repositories (Sekai/Interx releases)
- Builds and publishes Docker images
- Signs images with Cosign
- Updates compose file automatically

**`interx-update.yml` - Interx Services Update**
- Triggered by Interx repository releases
- Updates all Interx microservice versions in `compose.yml`
- Maintains version consistency across services

### Image Signing

All images are signed using Cosign for supply chain security. Verify signatures:

```bash
cosign verify --key cosign.pub ghcr.io/kiracore/sekin/shidai:v0.15.2
```

## Monitoring and Maintenance

### View Service Status

```bash
# Check Shidai status
curl http://localhost:8282/status

# Check blockchain status via RPC
curl http://localhost:26657/status
```

### Access Logs

Logs are centralized in `./syslog-data/`:

```bash
# View all logs
ls -lh ./syslog-data/

# Tail specific service logs
tail -f ./syslog-data/sekai.log
tail -f ./syslog-data/manager.log
```

### Backup Data

```bash
# Backup blockchain data
tar -czf sekai-backup.tar.gz ./sekai/

# Backup MongoDB data
docker compose exec mongo mongodump --out /backup
```

### Update Services

To update to newer versions, modify image tags in `compose.yml` and restart:

```bash
docker compose pull
docker compose up -d
```

## Development

### Source Code Structure

```
sekin/
├── src/
│   ├── shidai/          # Infrastructure manager
│   ├── sCaller/         # Sekai command executor
│   ├── iCaller/         # Interx command executor
│   ├── exporter/        # Metrics exporter
│   └── updater/         # Upgrade manager
├── manager/             # Interx Manager (P2P/HTTP)
├── proxy/               # Interx Proxy
├── worker/              # Interx Worker services
│   ├── cosmos/          # Cosmos indexer & interaction
│   ├── ethereum/        # Ethereum indexer & interaction
│   └── sai-storage-mongo/ # Storage service
├── scripts/             # Utility scripts
├── config/              # Configuration files
└── compose.yml          # Production compose file
```

### Building from Source

Each component can be built independently using its respective Dockerfile:

```bash
# Build Shidai
docker build -f shidai.Dockerfile -t sekin/shidai:custom .

# Build Syslog-ng
docker build -f syslog-ng.Dockerfile -t sekin/syslog-ng:custom .
```

## Troubleshooting

### Services Not Starting

Check logs for errors:

```bash
docker compose logs <service-name>
```

### Network Issues

Ensure the kiranet network exists:

```bash
docker network ls | grep kiranet
docker network inspect kiranet
```

### Port Conflicts

Check for existing processes using required ports:

```bash
sudo netstat -tulpn | grep -E '26657|26656|8080|8282'
```

### Disk Space

Monitor disk usage:

```bash
df -h
docker system df
```

Clean up unused Docker resources:

```bash
docker system prune -a --volumes
```

## Security Considerations

- RPC endpoint (26657) is exposed to 0.0.0.0 by default. Change to 127.0.0.1 in production
- Configure firewall rules to restrict access to sensitive ports
- Regularly update all services to latest versions
- Monitor logs for suspicious activity
- Use strong passwords for MongoDB
- Rotate signing keys periodically
- Enable prometheus API to connect monitoring

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request to `main`

## License

CC BY-NC-SA 4.0

## Resources

- [KIRA Network](https://kira.network)
- [GitHub Repository](https://github.com/KiraCore/sekin)
- [Docker Documentation](https://docs.docker.com)
- [Cosmos SDK Documentation](https://docs.cosmos.network)
