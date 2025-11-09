#!/bin/bash

echo "🚀 In-Memory vs Disk Performance Comparison"
echo "============================================"
echo ""

# Build
echo "📦 Building Flin..."
go build -o bin/flin-server ./cmd/kvserver

# Test 1: Disk-based storage
echo ""
echo "=========================================="
echo "Test 1: Disk-Based Storage (BadgerDB)"
echo "=========================================="
echo ""

pkill -f "kvserver.*disk-test" 2>/dev/null || true
rm -rf ./disk-test-data
sleep 2

./bin/flin-server \
  -node-id=disk-test \
  -http=localhost:8097 \
  -raft=localhost:9097 \
  -port=:7380 \
  -data=./disk-test-data \
  -workers=256 > /tmp/disk-server.log 2>&1 &
DISK_PID=$!

sleep 4

if ! kill -0 $DISK_PID 2>/dev/null; then
    echo "❌ Disk server failed to start"
    exit 1
fi

echo "✅ Disk-based server running (PID: $DISK_PID)"
echo ""

# Run benchmark
cat > /tmp/disk_bench.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

func main() {
	pool, _ := client.NewBinaryPoolClient("localhost:7380", 256)
	defer pool.Close()

	value := make([]byte, 1024)
	var totalOps atomic.Int64
	var wg sync.WaitGroup

	duration := 10 * time.Second
	startTime := time.Now()
	stopTime := startTime.Add(duration)

	for w := 0; w < 256; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			batchNum := 0
			for time.Now().Before(stopTime) {
				keys := make([]string, 10)
				values := make([][]byte, 10)
				for i := 0; i < 10; i++ {
					keys[i] = fmt.Sprintf("k_%d_%d_%d", workerID, batchNum, i)
					values[i] = value
				}
				if err := pool.MSet(keys, values); err == nil {
					totalOps.Add(10)
				}
				batchNum++
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	ops := totalOps.Load()
	throughput := float64(ops) / elapsed.Seconds()

	fmt.Printf("   Throughput: %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:    %.2fμs\n", (elapsed.Seconds()*1000000)/float64(ops))
}
EOF

go run /tmp/disk_bench.go

kill $DISK_PID 2>/dev/null
rm -rf ./disk-test-data
sleep 2

# Test 2: In-memory storage
echo ""
echo "=========================================="
echo "Test 2: In-Memory Storage (Like Redis)"
echo "=========================================="
echo ""

pkill -f "kvserver.*memory-test" 2>/dev/null || true
sleep 2

./bin/flin-server \
  -node-id=memory-test \
  -http=localhost:8098 \
  -raft=localhost:9098 \
  -port=:7380 \
  -workers=256 \
  -memory > /tmp/memory-server.log 2>&1 &
MEMORY_PID=$!

sleep 4

if ! kill -0 $MEMORY_PID 2>/dev/null; then
    echo "❌ Memory server failed to start"
    exit 1
fi

echo "✅ In-memory server running (PID: $MEMORY_PID)"
echo ""

# Run same benchmark
go run /tmp/disk_bench.go

kill $MEMORY_PID 2>/dev/null
sleep 2

# Cleanup
rm -f /tmp/disk_bench.go /tmp/disk-server.log /tmp/memory-server.log

echo ""
echo "📊 Summary"
echo "=========================================="
echo "Both modes tested with:"
echo "  - 256 workers"
echo "  - 256 connections"
echo "  - 10-key batches"
echo "  - 1KB values"
echo ""
echo "Key differences:"
echo "  Disk:   Durable, larger capacity"
echo "  Memory: Faster, volatile"
echo ""
echo "✅ Test complete!"
