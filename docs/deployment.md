# meinDENKWERK Deployment Guide

This guide covers deploying meinDENKWERK (mDW) in different environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Local Development](#local-development)
4. [Container Deployment (Podman/Docker)](#container-deployment)
5. [Service Configuration](#service-configuration)
6. [Networking](#networking)
7. [Data Persistence](#data-persistence)

---

## Prerequisites

### Required

- **Go 1.24+** - For building from source
- **Ollama** - LLM backend (https://ollama.com)
- **Podman** or **Docker** - For containerized deployment

### Optional

- **Qdrant** - Vector database (for production-scale RAG)
- **PostgreSQL** - For persistent logging/metrics

### Install Ollama

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Start Ollama
ollama serve

# Pull a model
ollama pull llama3.2
ollama pull nomic-embed-text  # For embeddings
```

---

## Quick Start

### Option 1: Local Binary

```bash
# Clone repository
git clone https://github.com/msto63/mDW.git
cd mDW

# Build
make build

# Run all services
make run-all

# Or run specific service
make run SERVICE=kant
```

### Option 2: Container

```bash
# Clone repository
git clone https://github.com/msto63/mDW.git
cd mDW

# Start with Podman
make podman-up

# Or with Docker
docker-compose up -d
```

---

## Local Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build the CLI
make build

# Binary is created at bin/mdw
./bin/mdw --help
```

### Running Individual Services

```bash
# Start Russell (Service Discovery) - required first
./bin/mdw serve russell

# Start other services in separate terminals
./bin/mdw serve turing   # LLM
./bin/mdw serve hypatia  # RAG
./bin/mdw serve babbage  # NLP
./bin/mdw serve leibniz  # Agent
./bin/mdw serve bayes    # Logging
./bin/mdw serve kant     # API Gateway
```

### Using the CLI

```bash
# Chat with LLM
./bin/mdw chat "Hello, how are you?"

# Search documents
./bin/mdw search "What is machine learning?"

# Analyze text
./bin/mdw analyze "This is a great product!"

# List models
./bin/mdw models list

# Check service status
./bin/mdw status
```

### Hot Reload Development

```bash
# Install air (hot reload tool)
go install github.com/air-verse/air@latest

# Run with hot reload
make dev
```

---

## Container Deployment

### Architecture

```
                    ┌──────────────────┐
                    │    Client        │
                    └────────┬─────────┘
                             │ :8080
                    ┌────────▼─────────┐
                    │      KANT        │ API Gateway
                    │   (HTTP/REST)    │
                    └────────┬─────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────▼────┐         ┌────▼────┐         ┌────▼────┐
   │ TURING  │         │ HYPATIA │         │ LEIBNIZ │
   │  :9200  │         │  :9220  │         │  :9140  │
   │  (LLM)  │         │  (RAG)  │         │ (Agent) │
   └────┬────┘         └────┬────┘         └────┬────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
            ┌───────────────┼───────────────┐
            │               │               │
       ┌────▼────┐     ┌────▼────┐     ┌────▼────┐
       │ RUSSELL │     │ BABBAGE │     │  BAYES  │
       │  :9100  │     │  :9150  │     │  :9120  │
       │(Discov.)│     │  (NLP)  │     │(Logging)│
       └─────────┘     └─────────┘     └─────────┘
```

### Start All Services

```bash
# With Podman
podman-compose up -d

# With Docker
docker-compose up -d

# View logs
podman-compose logs -f

# Stop
podman-compose down
```

### Start Individual Services

```bash
# Build all images
podman-compose build

# Start specific service
podman-compose up -d russell turing kant

# Scale a service
podman-compose up -d --scale turing=2
```

### Build Images Manually

```bash
# Build all
make podman-build

# Build specific service
podman build -f containers/kant/Containerfile -t mdw-kant .
```

### With Qdrant (Vector DB)

```bash
# Start with Qdrant profile
podman-compose --profile qdrant up -d
```

---

## Service Configuration

### Configuration File

The main configuration is in `configs/config.toml`:

```toml
# Application
[app]
name = "meinDENKWERK"
version = "0.1.0"
debug = true

# Services
[services]
[services.kant]
http_port = 8080

[services.russell]
grpc_port = 9100

[services.turing]
grpc_port = 9200
default_model = "llama3.2"
embedding_model = "nomic-embed-text"

[services.hypatia]
grpc_port = 9220
chunk_size = 512
chunk_overlap = 128

[services.leibniz]
grpc_port = 9140
max_iterations = 10

[services.babbage]
grpc_port = 9150

[services.bayes]
grpc_port = 9120

# Ollama
[ollama]
host = "http://localhost:11434"
timeout = "120s"
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MDW_CONFIG` | Path to config file | `./configs/config.toml` |
| `MDW_SERVICE` | Service to start | `kant` |
| `MDW_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `OLLAMA_HOST` | Ollama API endpoint | `http://localhost:11434` |

### Container-Specific Configuration

In containerized environments, use these settings in `podman-compose.yml`:

```yaml
services:
  turing:
    environment:
      - OLLAMA_HOST=host.containers.internal:11434
    extra_hosts:
      - "host.containers.internal:host-gateway"
```

---

## Networking

### Port Mapping

| Service | gRPC Port | HTTP Port | Purpose |
|---------|-----------|-----------|---------|
| Kant | - | 8080 | API Gateway |
| Russell | 9100 | 9101 | Service Discovery |
| Turing | 9200 | 9201 | LLM Management |
| Hypatia | 9220 | 9221 | RAG |
| Leibniz | 9140 | 9141 | Agentic AI |
| Babbage | 9150 | 9151 | NLP |
| Bayes | 9120 | 9121 | Logging |

### Internal Network

All containers communicate via `mdw-network`:

```yaml
networks:
  mdw-network:
    driver: bridge
```

Services can reach each other by container name:
- `http://turing:9200`
- `http://hypatia:9220`
- etc.

### External Access

Only Kant (8080) is exposed externally by default. To expose other ports, modify `podman-compose.yml`:

```yaml
services:
  turing:
    ports:
      - "9200:9200"  # Expose Turing gRPC
```

---

## Data Persistence

### Volume Mounts

```yaml
volumes:
  - ./configs:/app/configs:ro      # Configuration (read-only)
  - ./data:/app/data               # Application data
  - ./data/vectors:/app/data/vectors  # Vector store
  - ./data/logs:/app/data/logs     # Logs
```

### Directory Structure

```
data/
├── vectors/     # SQLite vector database (Hypatia)
├── logs/        # Log files (Bayes)
└── qdrant/      # Qdrant data (optional)
```

### Backup

```bash
# Backup data directory
tar -czf mdw-backup-$(date +%Y%m%d).tar.gz data/

# Backup vector database specifically
cp data/vectors/vectors.db data/vectors/vectors.db.backup
```

---

## Production Recommendations

### 1. Use External Databases

For production, consider:
- **Qdrant** for vector storage (instead of SQLite)
- **PostgreSQL** for logging/metrics

```yaml
# Add Qdrant
podman-compose --profile qdrant up -d
```

### 2. Configure Resource Limits

```yaml
services:
  turing:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

### 3. Enable Health Checks

```yaml
services:
  kant:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### 4. Use Secrets Management

```yaml
services:
  turing:
    secrets:
      - openai_api_key

secrets:
  openai_api_key:
    file: ./secrets/openai_api_key.txt
```

### 5. Logging Configuration

```yaml
services:
  kant:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

---

## Troubleshooting

See [Troubleshooting Guide](troubleshooting.md) for common issues and solutions.

### Quick Checks

```bash
# Check service status
./bin/mdw status

# Check container status
podman-compose ps

# View logs
podman-compose logs -f kant

# Test Ollama connection
curl http://localhost:11434/api/tags

# Test API Gateway
curl http://localhost:8080/api/v1/health
```
