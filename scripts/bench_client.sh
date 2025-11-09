#!/bin/bash

# Benchmark Flin client against server

set -e

echo "🚀 Flin Client Benchmark"
echo "========================"
echo ""

cd "$(dirname "$0")/.."

# Check if server is running
if ! nc -z localhost 6380 2>/dev/null; then
    echo "❌ Flin server is not running on localhost:6380"
    echo ""
    echo "Please start the server first:"
    echo "  Option 1: cd examples/web-app && docker compose up -d flin-server"
    echo "  Option 2: ./kvserver"
    echo ""
    exit 1
fi

echo "✅ Server detected on localhost:6380"
echo ""

# Build and run benchmark
echo "Building benchmark..."
go build -o /tmp/benchmark_client ./bencemark/client_bench.go

echo "Running benchmark..."
echo ""

/tmp/benchmark_client

# Cleanup
rm -f /tmp/benchmark_client

echo ""
echo "✅ Benchmark complete!"
