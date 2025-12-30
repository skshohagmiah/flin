#!/bin/bash

echo "🚀 TaskFlow Setup"
echo "================="
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/../.."

# Build Flin server
echo "📦 Building Flin server..."
mkdir -p bin
go build -o bin/flin-server ./cmd/server

# Kill any existing flin-server processes
pkill -f flin-server 2>/dev/null || true
sleep 1

# Clean up old data
echo "🧹 Cleaning up old data..."
rm -rf ./data/taskflow

# Start Flin server
echo "🔧 Starting Flin server..."
./bin/flin-server \
  -node-id=taskflow-node \
  -http=localhost:7080 \
  -raft=localhost:7090 \
  -port=:7380 \
  -data=./data/taskflow \
  -partitions=64 \
  -workers=256 &

SERVER_PID=$!
echo "   Server PID: $SERVER_PID"

# Wait for server to start
echo "⏳ Waiting for server to start..."
sleep 4

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    exit 1
fi

echo "✅ Flin server is running"
echo ""
echo "🎯 TaskFlow setup complete!"
echo ""
echo "Next steps:"
echo "  1. Run: cd examples/taskflow"
echo "  2. Run: go run main.go"
echo ""
echo "To stop the server: pkill -f flin-server"
