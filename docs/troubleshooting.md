# meinDENKWERK Troubleshooting Guide

This guide helps diagnose and resolve common issues with meinDENKWERK.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Service Issues](#service-issues)
3. [Connection Issues](#connection-issues)
4. [LLM/Ollama Issues](#llmollama-issues)
5. [RAG/Search Issues](#ragsearch-issues)
6. [Container Issues](#container-issues)
7. [Build Issues](#build-issues)

---

## Quick Diagnostics

### Check All Services

```bash
# CLI status check
./bin/mdw status

# API health check
curl http://localhost:8080/api/v1/health

# Container status
podman-compose ps
```

### Expected Output

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "services": {
    "kant": "healthy",
    "turing": "healthy",
    "hypatia": "healthy",
    "leibniz": "healthy",
    "babbage": "healthy"
  }
}
```

### View Logs

```bash
# All service logs
podman-compose logs -f

# Specific service
podman-compose logs -f turing

# Local logs
./bin/mdw serve kant 2>&1 | tee kant.log
```

---

## Service Issues

### Service Won't Start

**Symptom:** Service exits immediately or won't bind to port.

**Check:**
```bash
# Check if port is in use
lsof -i :8080   # Kant
lsof -i :9100   # Russell
lsof -i :9200   # Turing
```

**Solutions:**
1. Kill the process using the port:
   ```bash
   kill $(lsof -t -i :8080)
   ```
2. Use a different port in config
3. Check for zombie containers:
   ```bash
   podman ps -a | grep mdw
   podman rm -f $(podman ps -aq --filter name=mdw)
   ```

### Service Crashes with Panic

**Symptom:** `panic:` in logs.

**Common Causes:**
1. **Nil pointer** - Missing configuration
2. **Port conflict** - Port already in use
3. **Missing dependency** - Service started before dependency

**Solution:**
```bash
# Check config file exists
cat configs/config.toml

# Start services in correct order
./bin/mdw serve russell &
sleep 2
./bin/mdw serve turing &
sleep 2
./bin/mdw serve kant
```

### Service Disconnected

**Symptom:** Health check shows "disconnected" for a service.

**Check:**
```bash
# Verify service is running
podman ps | grep mdw-turing

# Test gRPC connection
grpcurl -plaintext localhost:9200 list
```

**Solutions:**
1. Restart the service:
   ```bash
   podman-compose restart turing
   ```
2. Check network connectivity:
   ```bash
   podman network inspect mdw-network
   ```

---

## Connection Issues

### "Connection Refused"

**Symptom:** `dial tcp: connection refused`

**Causes:**
1. Service not running
2. Wrong port
3. Firewall blocking

**Solutions:**
```bash
# 1. Check service is running
podman-compose ps

# 2. Verify port in config matches service
grep grpc_port configs/config.toml

# 3. Check firewall (macOS)
sudo pfctl -s rules
```

### "Context Deadline Exceeded"

**Symptom:** Requests timeout.

**Causes:**
1. Service overloaded
2. Network latency
3. LLM processing too slow

**Solutions:**
```bash
# Increase timeout
export MDW_TIMEOUT=120s

# Check Ollama is responsive
curl http://localhost:11434/api/tags

# Check system resources
top -l 1 | head -10
```

### gRPC Connection Pool Errors

**Symptom:** `too many open files` or connection pool exhausted.

**Solutions:**
```bash
# Increase file descriptor limit
ulimit -n 65535

# Check connection status
./bin/mdw status --verbose
```

---

## LLM/Ollama Issues

### "Ollama Not Available"

**Symptom:** Chat/generate fails with Ollama connection error.

**Check:**
```bash
# Is Ollama running?
pgrep ollama

# Can we reach it?
curl http://localhost:11434/api/tags
```

**Solutions:**
```bash
# Start Ollama
ollama serve

# In containers, check host access
# Add to podman-compose.yml:
extra_hosts:
  - "host.containers.internal:host-gateway"
```

### "Model Not Found"

**Symptom:** `model 'xyz' not found`

**Solutions:**
```bash
# List available models
ollama list

# Pull missing model
ollama pull llama3.2
ollama pull nomic-embed-text
```

### Slow LLM Responses

**Symptom:** Chat takes very long time.

**Causes:**
1. Large model on slow hardware
2. First request (model loading)
3. Long context

**Solutions:**
```bash
# Use smaller model
ollama pull llama3.2:1b  # 1B parameter version

# Pre-load model
curl -X POST http://localhost:11434/api/generate \
  -d '{"model": "llama3.2", "prompt": ""}'

# Reduce max_tokens in config
```

### Out of Memory

**Symptom:** `OOM killed` or system slowdown.

**Solutions:**
1. Use quantized models:
   ```bash
   ollama pull llama3.2:7b-q4_0  # 4-bit quantization
   ```
2. Limit parallel requests
3. Add swap space

---

## RAG/Search Issues

### "No Results Found"

**Symptom:** Search returns empty results.

**Check:**
```bash
# Are documents indexed?
curl -X GET http://localhost:8080/api/v1/collections

# Check vector store
ls -la data/vectors/
```

**Solutions:**
```bash
# Ingest documents first
curl -X POST http://localhost:8080/api/v1/ingest \
  -H "Content-Type: application/json" \
  -d '{"content": "Your document text", "collection": "default"}'

# Lower min_score threshold
curl -X POST http://localhost:8080/api/v1/search \
  -d '{"query": "test", "min_score": 0.5}'
```

### Poor Search Quality

**Symptom:** Results not relevant.

**Solutions:**
1. Adjust chunking settings:
   ```toml
   [services.hypatia]
   chunk_size = 512      # Smaller for precise matches
   chunk_overlap = 128   # Higher for context preservation
   ```
2. Use hybrid search:
   ```bash
   curl -X POST http://localhost:8080/api/v1/search \
     -d '{"query": "test", "hybrid": true}'
   ```
3. Check embedding model:
   ```bash
   ollama pull nomic-embed-text  # Ensure correct model
   ```

### Vector Store Corruption

**Symptom:** Database errors, search failures.

**Solutions:**
```bash
# Backup current data
cp data/vectors/vectors.db data/vectors/vectors.db.bak

# Vacuum the database
sqlite3 data/vectors/vectors.db "VACUUM;"

# Or recreate from scratch
rm data/vectors/vectors.db
# Re-ingest documents
```

---

## Container Issues

### Container Won't Build

**Symptom:** `podman build` fails.

**Common Errors:**

1. **"go: module requires Go 1.24"**
   ```bash
   # Update Go in Containerfile
   FROM golang:1.24-alpine AS builder
   ```

2. **"protoc: command not found"**
   ```bash
   # Install protoc first
   make proto-install
   make proto
   ```

3. **Network issues during build**
   ```bash
   # Use host network for build
   podman build --network=host ...
   ```

### Container Networking Issues

**Symptom:** Services can't communicate in containers.

**Check:**
```bash
# Inspect network
podman network inspect mdw-network

# Test DNS resolution
podman exec mdw-kant ping turing
```

**Solutions:**
```bash
# Recreate network
podman-compose down
podman network rm mdw-network
podman-compose up -d
```

### Volume Permission Denied

**Symptom:** `permission denied` accessing mounted volumes.

**Solutions:**
```bash
# Fix ownership
sudo chown -R $(id -u):$(id -g) data/

# Or use :Z flag (SELinux)
volumes:
  - ./data:/app/data:Z
```

---

## Build Issues

### Proto Generation Fails

**Symptom:** `make proto` fails.

**Solutions:**
```bash
# Install protoc (macOS)
brew install protobuf

# Install protoc (Linux)
apt-get install protobuf-compiler

# Install Go plugins
make proto-install

# Verify installation
protoc --version
which protoc-gen-go
which protoc-gen-go-grpc
```

### Module Not Found

**Symptom:** `cannot find module` errors.

**Solutions:**
```bash
# Update dependencies
go mod tidy

# Clear module cache
go clean -modcache

# Verify go.mod
cat go.mod | head
```

### CGO Errors (sqlite)

**Symptom:** `cgo: C compiler not found`

**Solutions:**
```bash
# macOS - install Xcode tools
xcode-select --install

# Linux
apt-get install gcc

# Or use pure Go SQLite
# (already using mattn/go-sqlite3 which requires CGO)
```

---

## Performance Issues

### High Memory Usage

**Check:**
```bash
# Container memory
podman stats

# Process memory
ps aux | grep mdw
```

**Solutions:**
1. Reduce embedding cache size:
   ```toml
   [cache]
   max_embeddings = 5000  # Reduce from 10000
   ```
2. Use smaller models
3. Limit concurrent requests

### High CPU Usage

**Check:**
```bash
# Find CPU-intensive process
top -o %CPU
```

**Solutions:**
1. Profile the application:
   ```bash
   go tool pprof http://localhost:8080/debug/pprof/profile
   ```
2. Reduce vector search dimensions
3. Use batch operations instead of individual requests

### Slow Startup

**Causes:**
1. Large model loading
2. Database initialization
3. Network timeouts

**Solutions:**
1. Use health check with startup probe
2. Pre-warm the cache:
   ```bash
   ./bin/mdw warmup
   ```

---

## Getting Help

### Collect Diagnostic Info

```bash
# System info
uname -a
go version
ollama --version

# Service versions
./bin/mdw version

# Logs (last 100 lines)
podman-compose logs --tail 100 > diagnostic.log

# Configuration (sanitized)
cat configs/config.toml | grep -v "key\|secret\|password"
```

### Report Issues

When reporting issues, include:
1. Error message (full stack trace)
2. Steps to reproduce
3. Configuration (sanitized)
4. System info (OS, Go version, Docker/Podman version)
5. Relevant logs

### Common Log Locations

| Component | Location |
|-----------|----------|
| Service Logs | stdout/stderr or `data/logs/` |
| Container Logs | `podman-compose logs` |
| Ollama Logs | `~/.ollama/logs/` |
