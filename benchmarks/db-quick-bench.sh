#!/bin/bash

echo "🚀 Quick Document Store Throughput Test"
echo "========================================"
echo ""

# Configuration
CONCURRENCY=32
DURATION=5
DOC_SIZE=256

echo "📊 Configuration:"
echo "   Concurrency: $CONCURRENCY workers"
echo "   Duration: ${DURATION}s"
echo "   Document size: ${DOC_SIZE} bytes"
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Kill any existing processes
pkill -f flin-server 2>/dev/null || true
sleep 1

# Start server
echo "🔧 Starting Flin server..."
./bin/flin-server \
  -node-id=bench-node \
  -http=localhost:7080 \
  -raft=localhost:7090 \
  -port=:7380 \
  -data=./data/bench \
  -partitions=32 \
  -workers=128 > /dev/null 2>&1 &

SERVER_PID=$!
echo "   Server PID: $SERVER_PID"

# Wait for server
echo "⏳ Waiting for server to start..."
sleep 4

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    exit 1
fi

echo "✅ Server is running"
echo ""

# Create benchmark
mkdir -p /tmp/flin_quick_bench
cd /tmp/flin_quick_bench

cat > go.mod <<MODEOF
module flin-quick-bench

go 1.21

require github.com/skshohagmiah/flin v0.0.0

replace github.com/skshohagmiah/flin => $SCRIPT_DIR/..
MODEOF

cat > main.go <<'EOF'
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	concurrency := CONCURRENCY_PLACEHOLDER
	duration := DURATION_PLACEHOLDER * time.Second
	docSize := DOC_SIZE_PLACEHOLDER
	
	client, err := flin.NewDocClient("localhost:7380")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.Close()
	
	fmt.Println("📝 Quick Throughput Test")
	fmt.Println("========================")
	fmt.Println()
	
	var createOps atomic.Int64
	var wg sync.WaitGroup
	
	startTime := time.Now()
	stopTime := startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				doc := map[string]interface{}{
					"worker":  workerID,
					"counter": ops,
					"data":    randString(docSize),
				}
				_, err := client.Collection("docs").Create().Data(doc).Exec()
				if err == nil {
					ops++
				}
			}
			createOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	elapsed := time.Since(startTime)
	total := createOps.Load()
	throughput := float64(total) / elapsed.Seconds()
	latency := (elapsed.Seconds() * 1000000) / float64(total)
	
	fmt.Printf("✅ Results:\n")
	fmt.Printf("   Operations:  %d\n", total)
	fmt.Printf("   Duration:    %v\n", elapsed)
	fmt.Printf("   Throughput:  %.0f docs/sec\n", throughput)
	fmt.Printf("   Latency:     %.2fμs\n", latency)
}

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
EOF

# Replace placeholders
sed -i "s/CONCURRENCY_PLACEHOLDER/$CONCURRENCY/g" main.go
sed -i "s/DURATION_PLACEHOLDER/$DURATION/g" main.go
sed -i "s/DOC_SIZE_PLACEHOLDER/$DOC_SIZE/g" main.go

echo "📊 Running benchmark..."
echo ""

go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $SERVER_PID 2>/dev/null
rm -rf /tmp/flin_quick_bench
rm -rf ./data/bench

echo "✅ Benchmark complete!"
