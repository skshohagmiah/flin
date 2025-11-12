# Flin 3-Node Cluster

Production-ready 3-node cluster with Raft consensus and replication.

## Quick Start

```bash
./run.sh
```

## Manual Usage

```bash
# Start cluster
docker compose up -d

# Stop cluster
docker compose down -v

# View logs
docker compose logs -f

# View specific node logs
docker compose logs -f flin-node1
```

## Access

- **Node 1** - KV: `http://localhost:6380`, HTTP: `http://localhost:8080`
- **Node 2** - KV: `http://localhost:6381`, HTTP: `http://localhost:8081`
- **Node 3** - KV: `http://localhost:6382`, HTTP: `http://localhost:8082`

## Test Replication

```bash
# Write to node 1
curl -X POST http://localhost:6380/kv/test \
  -H "Content-Type: application/json" \
  -d '{"value":"hello"}'

# Read from node 2 (replicated!)
curl http://localhost:6381/kv/test

# Check cluster status
curl http://localhost:8080/nodes | jq

# Check partitions
curl http://localhost:8080/partitions | jq
```

## Run Tests

To run comprehensive tests on this cluster:

```bash
cd ../test-runner
./run.sh
```
