#!/bin/bash

echo "🚀 Flin Single Node Test"
echo "========================"
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Clean up any existing cluster
echo "🧹 Cleaning up existing cluster..."
(cd examples/10-node-cluster 2>/dev/null && docker compose down 2>/dev/null) || true

# Kill any existing flin-server processes
pkill -f flin-server 2>/dev/null || true
sleep 1

# Build the server
echo "📦 Building Flin..."
mkdir -p bin
go build -o bin/flin-server ./cmd/kvserver

# Start the server in background
echo "🔧 Starting single node..."
./bin/flin-server \
  -node-id=node-1 \
  -http=localhost:7080 \
  -raft=localhost:7090 \
  -port=:7380 \
  -data=./data/node1 \
  -partitions=64 &

SERVER_PID=$!
echo "   Server PID: $SERVER_PID"

# Wait for server to start
echo "⏳ Waiting for server to start..."
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    exit 1
fi

echo "✅ Server is running"
echo ""

# Simple benchmark
echo "📊 Running benchmark..."
echo "   Concurrency: 128 workers"
echo "   Duration: 10 seconds"
echo "   Value size: 1KB"
echo ""

# Create benchmark program
cat > /tmp/flin_single_bench.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

func main() {
	concurrency := 128
	duration := 10 * time.Second
	
	var totalOps atomic.Int64
	var wg sync.WaitGroup
	
	value := make([]byte, 1024)
	startTime := time.Now()
	stopTime := startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			c, err := client.NewTCP("localhost:7380")
			if err != nil {
				return
			}
			defer c.Close()
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("key_%d_%d", workerID, ops)
				if err := c.Set(key, value); err == nil {
					ops++
				}
			}
			totalOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	elapsed := time.Since(startTime)
	total := totalOps.Load()
	throughput := float64(total) / elapsed.Seconds()
	
	// Format throughput
	var throughputStr string
	if throughput >= 1000000 {
		throughputStr = fmt.Sprintf("%.2fM", throughput/1000000)
	} else if throughput >= 1000 {
		throughputStr = fmt.Sprintf("%.2fK", throughput/1000)
	} else {
		throughputStr = fmt.Sprintf("%.0f", throughput)
	}
	
	// Format total ops
	var totalStr string
	if total >= 1000000 {
		totalStr = fmt.Sprintf("%.2fM", float64(total)/1000000)
	} else if total >= 1000 {
		totalStr = fmt.Sprintf("%.2fK", float64(total)/1000)
	} else {
		totalStr = fmt.Sprintf("%d", total)
	}
	
	fmt.Printf("\n📊 Results:\n")
	fmt.Printf("   Total operations: %s\n", totalStr)
	fmt.Printf("   Duration: %.2fs\n", elapsed.Seconds())
	fmt.Printf("   Throughput: %s ops/sec\n", throughputStr)
	fmt.Printf("   Latency: %.2fμs per op\n", (elapsed.Seconds()*1000000)/float64(total))
}
EOF

go run /tmp/flin_single_bench.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null
rm -f /tmp/flin_single_bench.go
rm -rf ./data/node1

echo "✅ Test complete!"
