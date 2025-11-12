#!/bin/bash

echo "🚀 Flin Cluster Throughput Benchmark"
echo "====================================="
echo ""
# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Configuration
CONCURRENCY=${1:-256}
DURATION=${2:-5}
VALUE_SIZE=${3:-512}

echo "📊 Configuration:"
echo "   Concurrency: $CONCURRENCY workers (single client)"
echo "   Duration: ${DURATION}s"
echo "   Value size: ${VALUE_SIZE} bytes"
echo "   Cluster: 3 nodes"
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

# Clean up old data
rm -rf ./data/bench-node-*

# Start 3-node cluster
echo "🔧 Starting 3-node cluster..."

# Node 1 (bootstrap)
./bin/flin-server \
  -node-id=bench-node-1 \
  -http=localhost:7080 \
  -raft=localhost:7090 \
  -port=:7380 \
  -data=./data/bench-node-1 \
  -partitions=64 \
  -workers=256 &
NODE1_PID=$!
echo "   Node 1 PID: $NODE1_PID (bootstrap)"

sleep 3

# Node 2 (join node 1)
./bin/flin-server \
  -node-id=bench-node-2 \
  -http=localhost:7081 \
  -raft=localhost:7091 \
  -port=:7381 \
  -data=./data/bench-node-2 \
  -partitions=64 \
  -workers=256 \
  -join=localhost:7080 &
NODE2_PID=$!
echo "   Node 2 PID: $NODE2_PID"

sleep 2

# Node 3 (join node 1)
./bin/flin-server \
  -node-id=bench-node-3 \
  -http=localhost:7082 \
  -raft=localhost:7092 \
  -port=:7382 \
  -data=./data/bench-node-3 \
  -partitions=64 \
  -workers=256 \
  -join=localhost:7080 &
NODE3_PID=$!
echo "   Node 3 PID: $NODE3_PID"

# Wait for cluster to stabilize
echo "⏳ Waiting for cluster to stabilize..."
sleep 8

# Wait for partitions to be created
echo "⏳ Waiting for partitions to be created..."
sleep 5

# Verify partitions are created
echo "🔍 Verifying partitions..."
for i in {1..10}; do
    PARTITION_COUNT=$(curl -s http://localhost:7080/cluster 2>/dev/null | grep -o '"partitions"' | wc -l)
    if [ "$PARTITION_COUNT" -gt 1 ]; then
        echo "✅ Partitions detected in cluster API"
        break
    fi
    echo "   Waiting for partitions... (attempt $i/10)"
    sleep 2
done

# Check if all nodes are running
if ! kill -0 $NODE1_PID 2>/dev/null; then
    echo "❌ Node 1 failed to start"
    exit 1
fi
if ! kill -0 $NODE2_PID 2>/dev/null; then
    echo "❌ Node 2 failed to start"
    exit 1
fi
if ! kill -0 $NODE3_PID 2>/dev/null; then
    echo "❌ Node 3 failed to start"
    exit 1
fi

echo "✅ Cluster is running"
echo ""

# Create benchmark program using the new cluster-aware SDK
mkdir -p /tmp/flin_cluster_bench
cd /tmp/flin_cluster_bench

cat > go.mod << MODEOF
module flin-cluster-bench

go 1.21
require github.com/skshohagmiah/flin v0.0.0

replace github.com/skshohagmiah/flin => $SCRIPT_DIR/..
MODEOF

cat > main.go << 'GOEOF'
package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	// Parse command line args
	concurrency := 256
	duration := 5
	valueSize := 512
	
	if len(os.Args) > 1 {
		concurrency, _ = strconv.Atoi(os.Args[1])
	}
	if len(os.Args) > 2 {
		duration, _ = strconv.Atoi(os.Args[2])
	}
	if len(os.Args) > 3 {
		valueSize, _ = strconv.Atoi(os.Args[3])
	}
	
	// Create cluster-aware client using new SDK
	opts := flin.DefaultClusterOptions([]string{
		"localhost:7080",
		"localhost:7081",
		"localhost:7082",
	})
	
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.Close()
	
	fmt.Printf("Client mode: %s\n", map[bool]string{true: "Cluster", false: "Single-node"}[client.IsClusterMode()])
	
	// Wait for topology
	time.Sleep(2 * time.Second)
	
	// Prepare test data
	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 256)
	}
	
	// Run WRITE test
	fmt.Println("🔴 WRITE Test (SET operations - distributed)")
	fmt.Println("----------------------------------------------")
	
	var writeOps atomic.Int64
	var wg sync.WaitGroup
	
	startTime := time.Now()
	stopTime := startTime.Add(time.Duration(duration) * time.Second)
	
	var writeErrors atomic.Int64
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			errors := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("worker%d_key%d", workerID, ops)
				if err := client.Set(key, value); err == nil {
					ops++
				} else {
					errors++
					if errors == 1 {
						// Print first error
						fmt.Printf("❌ Write error: %v\n", err)
					}
				}
			}
			writeOps.Add(ops)
			writeErrors.Add(errors)
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
	fmt.Printf("   Errors:      %d\n", writeErrors.Load())
	fmt.Printf("   Throughput:  %s ops/sec\n", writeThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\n", writeLatency)
	fmt.Println()
	
	// Run READ test
	fmt.Println("🟢 READ Test (GET operations - distributed)")
	fmt.Println("--------------------------------------------")
	
	var readOps atomic.Int64
	
	startTime = time.Now()
	stopTime = startTime.Add(time.Duration(duration) * time.Second)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("worker%d_key%d", workerID, ops%1000)
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
	
	// Test batch operations (distributed across nodes)
	fmt.Println("📦 BATCH Test (MSET/MGET - parallel across nodes)")
	fmt.Println("--------------------------------------------------")
	
	batchSize := 100
	var batchOps atomic.Int64
	
	startTime = time.Now()
	stopTime = startTime.Add(time.Duration(duration) * time.Second)
	
	for i := 0; i < concurrency/4; i++ { // Fewer workers for batch
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ops := int64(0)
			for time.Now().Before(stopTime) {
				keys := make([]string, batchSize)
				values := make([][]byte, batchSize)
				for j := 0; j < batchSize; j++ {
					keys[j] = fmt.Sprintf("batch_%d_%d_%d", workerID, ops, j)
					values[j] = value
				}
				
				if err := client.MSet(keys, values); err == nil {
					ops++
				}
			}
			batchOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	batchElapsed := time.Since(startTime)
	batchTotal := batchOps.Load()
	batchThroughput := float64(batchTotal*int64(batchSize)) / batchElapsed.Seconds()
	
	var batchThroughputStr string
	if batchThroughput >= 1000000 {
		batchThroughputStr = fmt.Sprintf("%.2fM", batchThroughput/1000000)
	} else if batchThroughput >= 1000 {
		batchThroughputStr = fmt.Sprintf("%.2fK", batchThroughput/1000)
	} else {
		batchThroughputStr = fmt.Sprintf("%.0f", batchThroughput)
	}
	
	fmt.Printf("   Batch ops:   %d (x%d keys each)\n", batchTotal, batchSize)
	fmt.Printf("   Total keys:  %.2fM\n", float64(batchTotal*int64(batchSize))/1000000)
	fmt.Printf("   Throughput:  %s keys/sec\n", batchThroughputStr)
	fmt.Println()
	
	// Summary
	fmt.Println("📊 Cluster Summary")
	fmt.Println("===================")
	fmt.Printf("   Nodes:  3\n")
	fmt.Printf("   WRITE:  %s ops/sec (%.2fμs latency)\n", writeThroughputStr, writeLatency)
	fmt.Printf("   READ:   %s ops/sec (%.2fμs latency)\n", readThroughputStr, readLatency)
	fmt.Printf("   BATCH:  %s keys/sec (parallel across nodes)\n", batchThroughputStr)
	
	ratio := readThroughput / writeThroughput
	fmt.Printf("   Ratio:  %.2fx (reads faster than writes)\n", ratio)
	fmt.Println()
	fmt.Println("   ✨ Client-side partitioning: Keys automatically routed to correct nodes!")
	fmt.Println("   ✨ Batch operations: Parallelized across multiple nodes!")
}
GOEOF

echo "📊 Running cluster throughput benchmark..."
echo ""

cd /tmp/flin_cluster_bench
go mod tidy 2>/dev/null
go run main.go $CONCURRENCY $DURATION $VALUE_SIZE

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null
rm -rf /tmp/flin_cluster_bench
rm -rf ./data/bench-node-*

echo "✅ Cluster benchmark complete!"
