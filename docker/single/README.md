# Flin Single Node

Single node deployment for development and testing.

## Quick Start

```bash
./run.sh
```

## Manual Usage

```bash
# Start
docker compose up -d

# Stop
docker compose down -v

# View logs
docker compose logs -f
```

## Access

- **KV API**: `http://localhost:6380`
- **HTTP API**: `http://localhost:8080`

## Test It

```bash
# Set a value
curl -X POST http://localhost:6380/kv/test \
  -H "Content-Type: application/json" \
  -d '{"value":"hello"}'

# Get the value
curl http://localhost:6380/kv/test

# Check health
curl http://localhost:8080/health
```
