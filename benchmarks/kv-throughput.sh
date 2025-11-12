#!/bin/bash

echo "🚀 Flin KV Throughput Benchmark"
echo "================================"
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Configuration
CONCURRENCY=${1:-128}
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
    go build -o bin/flin-server ./cmd/kvserver
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

# Create benchmark program using the new SDK
mkdir -p /tmp/flin_bench
cd /tmp/flin_bench

cat > go.mod << MODEOF
module flin-bench

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
	
	// Create client using new SDK
	opts := flin.DefaultOptions("localhost:7380")
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.Close()
	
	fmt.Printf("Client mode: %s\n", map[bool]string{true: "Cluster", false: "Single-node"}[client.IsClusterMode()])
	fmt.Println()
	
	// Prepare test data
	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 256)
	}
	
	// Run WRITE test
	fmt.Println("🔴 WRITE Test (SET operations)")
	fmt.Println("--------------------------------")
	
	var writeOps atomic.Int64
	var wg sync.WaitGroup
	
	startTime := time.Now()
	stopTime := startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("key_%d_%d", workerID, ops)
				if err := client.Set(key, value); err == nil {
					ops++
				}
			}
			writeOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	writeElapsed := time.Since(startTime)
	writeTotal := writeOps.Load()
	writeThroughput := float64(writeTotal) / writeElapsed.Seconds()
	writeLatency := (writeElapsed.Seconds() * 1000000) / float64(writeTotal)
	
	// Format write results
	var writeThroughputStr string
	if writeThroughput >= 1000000 {
		writeThroughputStr = fmt.Sprintf("%.2fM", writeThroughput/1000000)
	} else if writeThroughput >= 1000 {
		writeThroughputStr = fmt.Sprintf("%.2fK", writeThroughput/1000)
	} else {
		writeThroughputStr = fmt.Sprintf("%.0f", writeThroughput)
	}
	
	var writeTotalStr string
	if writeTotal >= 1000000 {
		writeTotalStr = fmt.Sprintf("%.2fM", float64(writeTotal)/1000000)
	} else if writeTotal >= 1000 {
		writeTotalStr = fmt.Sprintf("%.2fK", float64(writeTotal)/1000)
	} else {
		writeTotalStr = fmt.Sprintf("%d", writeTotal)
	}
	
	fmt.Printf("   Operations:  %s\n", writeTotalStr)
	fmt.Printf("   Throughput:  %s ops/sec\n", writeThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\n", writeLatency)
	fmt.Println()
	
	// Run READ test
	fmt.Println("🟢 READ Test (GET operations)")
	fmt.Println("--------------------------------")
	
	var readOps atomic.Int64
	
	startTime = time.Now()
	stopTime = startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("key_%d_%d", workerID, ops%1000)
				if _, err := client.Get(key); err == nil {
					ops++
				}
			}
			readOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	readElapsed := time.Since(startTime)
	readTotal := readOps.Load()
	readThroughput := float64(readTotal) / readElapsed.Seconds()
	readLatency := (readElapsed.Seconds() * 1000000) / float64(readTotal)
	
	// Format read results
	var readThroughputStr string
	if readThroughput >= 1000000 {
		readThroughputStr = fmt.Sprintf("%.2fM", readThroughput/1000000)
	} else if readThroughput >= 1000 {
		readThroughputStr = fmt.Sprintf("%.2fK", readThroughput/1000)
	} else {
		readThroughputStr = fmt.Sprintf("%.0f", readThroughput)
	}
	
	var readTotalStr string
	if readTotal >= 1000000 {
		readTotalStr = fmt.Sprintf("%.2fM", float64(readTotal)/1000000)
	} else if readTotal >= 1000 {
		readTotalStr = fmt.Sprintf("%.2fK", float64(readTotal)/1000)
	} else {
		readTotalStr = fmt.Sprintf("%d", readTotal)
	}
	
	fmt.Printf("   Operations:  %s\n", readTotalStr)
	fmt.Printf("   Throughput:  %s ops/sec\n", readThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\n", readLatency)
	fmt.Println()
	
	// Summary
	fmt.Println("📊 Summary")
	fmt.Println("===================")
	fmt.Printf("   WRITE:  %s ops/sec (%.2fμs latency)\n", writeThroughputStr, writeLatency)
	fmt.Printf("   READ:   %s ops/sec (%.2fμs latency)\n", readThroughputStr, readLatency)
	
	ratio := readThroughput / writeThroughput
	fmt.Printf("   Ratio:  %.2fx (reads faster than writes)\n", ratio)
	
	improvement := ((readThroughput - writeThroughput) / writeThroughput) * 100
	if improvement > 0 {
		fmt.Printf("   Reads are %.1f%% faster! 🚀\n", improvement)
	}
}
EOF

echo "📊 Running throughput benchmark..."
echo ""

cd /tmp/flin_bench
go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $SERVER_PID 2>/dev/null
rm -rf /tmp/flin_bench
rm -rf ./data/bench

echo "✅ Benchmark complete!"
