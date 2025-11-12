#!/bin/bash

set -e

echo "🚀 Starting Flin Single Node"
echo "============================="
echo ""

# Start single node
docker compose up -d

echo ""
echo "⏳ Waiting for node to be ready..."
sleep 5

# Check health
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Node is healthy"
else
    echo "❌ Node is not responding"
    exit 1
fi

echo ""
echo "📊 Running Performance Benchmark (using Go client)"
echo ""

# Run Go benchmark from project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.." && go run docker/single/benchmark.go
cd "$SCRIPT_DIR"

echo ""
echo "================================="
echo "✅ Single node running!"
echo ""
echo "Access points:"
echo "  KV API:   http://localhost:6380"
echo "  HTTP API: http://localhost:8080"
echo ""
echo "Test it:"
echo "  curl -X POST http://localhost:6380/kv/test -H 'Content-Type: application/json' -d '{\"value\":\"hello\"}'"
echo "  curl http://localhost:6380/kv/test"
echo ""
echo "Stop with: docker compose down -v"
