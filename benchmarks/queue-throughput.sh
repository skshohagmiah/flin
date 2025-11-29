#!/bin/bash

echo "🚀 Flin Queue Throughput Benchmark"
echo "==================================="
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Configuration
CONCURRENCY=${1:-256}
DURATION=${2:-10}
VALUE_SIZE=${3:-1024}

echo "📊 Configuration:"
echo "   Concurrency: $CONCURRENCY workers"
echo "   Duration: ${DURATION}s"
echo "   Value size: ${VALUE_SIZE} bytes"
echo ""

# Build the server if needed
if [ ! -f "bin/flin-server" ]; then
    echo "📦 Building Flin server..."
    mkdir -p bin
    go build -o bin/flin-server ./cmd/server
fi

# Kill any existing flin-server processes
pkill -f flin-server 2>/dev/null || true
sleep 1

# Start the server in background
echo "🔧 Starting Flin server..."
./bin/flin-server \
  -node-id=bench-node \
  -http=localhost:7080 \
  -raft=localhost:7090 \
  -port=:7380 \
  -data=./data/bench \
  -partitions=64 \
  -workers=256 &

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

# Create benchmark program
mkdir -p /tmp/flin_queue_bench
cd /tmp/flin_queue_bench

cat > go.mod << MODEOF
module flin-queue-bench

go 1.21

require github.com/skshohagmiah/flin v0.0.0

replace github.com/skshohagmiah/flin => $SCRIPT_DIR/..
MODEOF

cat > main.go << EOF
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	concurrency := $CONCURRENCY
	duration := $DURATION * time.Second
	valueSize := $VALUE_SIZE

	fmt.Printf("Unified server (KV + Queue): localhost:7380\\n")
	fmt.Println()

	// Create client using real SDK (supports connection pooling)
	opts := flin.DefaultOptions("localhost:7380")
	// Ensure pool is large enough for all workers
	opts.MaxConnections = concurrency + 10
	opts.MinConnections = concurrency / 2
	
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to create client: %v\\n", err)
		return
	}
	defer client.Close()

	// Prepare test data
	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 256)
	}

	// Run PUSH test
	fmt.Println("🔴 PUSH Test (Enqueue operations)")
	fmt.Println("----------------------------------")

	var pushOps atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()
	stopTime := startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			queueName := fmt.Sprintf("bench_queue_%d", workerID)
			ops := int64(0)
			for time.Now().Before(stopTime) {
				if err := client.Queue.Push(queueName, value); err == nil {
					ops++
				}
			}
			pushOps.Add(ops)
		}(i)
	}

	wg.Wait()
	pushElapsed := time.Since(startTime)
	pushTotal := pushOps.Load()
	pushThroughput := float64(pushTotal) / pushElapsed.Seconds()
	pushLatency := (pushElapsed.Seconds() * 1000000) / float64(pushTotal)

	// Format push results
	var pushThroughputStr string
	if pushThroughput >= 1000000 {
		pushThroughputStr = fmt.Sprintf("%.2fM", pushThroughput/1000000)
	} else if pushThroughput >= 1000 {
		pushThroughputStr = fmt.Sprintf("%.2fK", pushThroughput/1000)
	} else {
		pushThroughputStr = fmt.Sprintf("%.0f", pushThroughput)
	}

	var pushTotalStr string
	if pushTotal >= 1000000 {
		pushTotalStr = fmt.Sprintf("%.2fM", float64(pushTotal)/1000000)
	} else if pushTotal >= 1000 {
		pushTotalStr = fmt.Sprintf("%.2fK", float64(pushTotal)/1000)
	} else {
		pushTotalStr = fmt.Sprintf("%d", pushTotal)
	}

	fmt.Printf("   Operations:  %s\\n", pushTotalStr)
	fmt.Printf("   Throughput:  %s ops/sec\\n", pushThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\\n", pushLatency)
	fmt.Println()

	// Run POP test
	fmt.Println("🟢 POP Test (Dequeue operations)")
	fmt.Println("----------------------------------")

	var popOps atomic.Int64

	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			queueName := fmt.Sprintf("bench_queue_%d", workerID)
			ops := int64(0)
			for time.Now().Before(stopTime) {
				if _, err := client.Queue.Pop(queueName); err == nil {
					ops++
				} else {
					// Queue empty, push one item to keep going
					client.Queue.Push(queueName, value)
				}
			}
			popOps.Add(ops)
		}(i)
	}

	wg.Wait()
	popElapsed := time.Since(startTime)
	popTotal := popOps.Load()
	popThroughput := float64(popTotal) / popElapsed.Seconds()
	popLatency := (popElapsed.Seconds() * 1000000) / float64(popTotal)

	// Format pop results
	var popThroughputStr string
	if popThroughput >= 1000000 {
		popThroughputStr = fmt.Sprintf("%.2fM", popThroughput/1000000)
	} else if popThroughput >= 1000 {
		popThroughputStr = fmt.Sprintf("%.2fK", popThroughput/1000)
	} else {
		popThroughputStr = fmt.Sprintf("%.0f", popThroughput)
	}

	var popTotalStr string
	if popTotal >= 1000000 {
		popTotalStr = fmt.Sprintf("%.2fM", float64(popTotal)/1000000)
	} else if popTotal >= 1000 {
		popTotalStr = fmt.Sprintf("%.2fK", float64(popTotal)/1000)
	} else {
		popTotalStr = fmt.Sprintf("%d", popTotal)
	}

	fmt.Printf("   Operations:  %s\\n", popTotalStr)
	fmt.Printf("   Throughput:  %s ops/sec\\n", popThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\\n", popLatency)
	fmt.Println()

	// Summary
	fmt.Println("📊 Summary")
	fmt.Println("===================")
	fmt.Printf("   PUSH:  %s ops/sec (%.2fμs latency)\\n", pushThroughputStr, pushLatency)
	fmt.Printf("   POP:   %s ops/sec (%.2fμs latency)\\n", popThroughputStr, popLatency)

	avgThroughput := (pushThroughput + popThroughput) / 2
	var avgThroughputStr string
	if avgThroughput >= 1000000 {
		avgThroughputStr = fmt.Sprintf("%.2fM", avgThroughput/1000000)
	} else if avgThroughput >= 1000 {
		avgThroughputStr = fmt.Sprintf("%.2fK", avgThroughput/1000)
	} else {
		avgThroughputStr = fmt.Sprintf("%.0f", avgThroughput)
	}
	fmt.Printf("   Average: %s ops/sec 🚀\\n", avgThroughputStr)
}
EOF

echo "📊 Running queue throughput benchmark..."
echo ""

cd /tmp/flin_queue_bench
go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $SERVER_PID 2>/dev/null
rm -rf /tmp/flin_queue_bench
rm -rf ./data/bench

echo "✅ Benchmark complete!"
