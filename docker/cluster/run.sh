#!/bin/bash

set -e -o pipefail

echo "🚀 Starting Flin 3-Node Cluster"
echo "================================"
echo ""

# Start cluster
docker compose up -d

echo ""
echo "⏳ Waiting for cluster to be ready..."
sleep 15

# Check cluster health with retries
echo ""
echo "📊 Checking cluster health..."

# Temporarily disable exit on error for health checks
set +e

HEALTHY_NODES=0
MAX_RETRIES=10
RETRY=0

while [ $HEALTHY_NODES -lt 3 ] && [ $RETRY -lt $MAX_RETRIES ]; do
    HEALTHY_NODES=0
    for port in 8080 8081 8082; do
        if curl -sf http://localhost:${port}/health > /dev/null 2>&1; then
            ((HEALTHY_NODES++))
        fi
    done
    
    if [ $HEALTHY_NODES -lt 3 ]; then
        echo "  ⏳ $HEALTHY_NODES/3 nodes ready, waiting..."
        sleep 2
        ((RETRY++))
    fi
done

# Re-enable exit on error
set -e

if [ $HEALTHY_NODES -eq 3 ]; then
    echo "  ✓ All 3 nodes are healthy"
else
    echo ""
    echo "❌ Only $HEALTHY_NODES/3 nodes are healthy after ${MAX_RETRIES} retries"
    echo "Check logs with: docker compose logs"
    exit 1
fi

echo ""
echo "📊 Running Cluster Performance Benchmark (using Go client)"
echo ""

# Run Go benchmark from project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.." && go run docker/cluster/benchmark.go
cd "$SCRIPT_DIR"

echo ""
echo "========================================="
echo "✅ Cluster running with performance metrics!"
echo ""
echo "Access points:"
echo "  Node 1 - KV: http://localhost:6380, HTTP: http://localhost:8080"
echo "  Node 2 - KV: http://localhost:6381, HTTP: http://localhost:8081"
echo "  Node 3 - KV: http://localhost:6382, HTTP: http://localhost:8082"
echo ""
echo "Test replication:"
echo "  # Write to node 1"
echo "  curl -X POST http://localhost:6380/kv/test -H 'Content-Type: application/json' -d '{\"value\":\"hello\"}'"
echo ""
echo "  # Read from node 2 (replicated!)"
echo "  curl http://localhost:6381/kv/test"
echo ""
echo "  # Check cluster status"
echo "  curl http://localhost:8080/nodes | jq"
echo ""
echo "Stop with: docker compose down -v"
