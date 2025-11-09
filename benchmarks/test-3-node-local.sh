#!/bin/bash

echo "🚀 Flin 3-Node Local Cluster Test"
echo "=================================="
echo ""

# Save script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Clean up any existing processes
echo "🧹 Cleaning up existing processes..."
pkill -f "kvserver.*node-" 2>/dev/null || true
rm -rf ./cluster-test-data
sleep 2

# Build the server
echo "📦 Building Flin..."
go build -o bin/flin-server ./cmd/kvserver

# Start Node 1 (bootstrap)
echo "🔧 Starting Node 1 (bootstrap)..."
./bin/flin-server \
  -node-id=node-1 \
  -http=localhost:8080 \
  -raft=localhost:9080 \
  -port=:6380 \
  -data=./cluster-test-data/node1 \
  -workers=256 > /tmp/node1.log 2>&1 &
NODE1_PID=$!
echo "   Node 1 PID: $NODE1_PID"

sleep 3

# Start Node 2 (join)
echo "🔧 Starting Node 2 (join)..."
./bin/flin-server \
  -node-id=node-2 \
  -http=localhost:8081 \
  -raft=localhost:9081 \
  -port=:6381 \
  -data=./cluster-test-data/node2 \
  -join=localhost:8080 \
  -workers=256 > /tmp/node2.log 2>&1 &
NODE2_PID=$!
echo "   Node 2 PID: $NODE2_PID"

sleep 3

# Start Node 3 (join)
echo "🔧 Starting Node 3 (join)..."
./bin/flin-server \
  -node-id=node-3 \
  -http=localhost:8082 \
  -raft=localhost:9082 \
  -port=:6382 \
  -data=./cluster-test-data/node3 \
  -join=localhost:8080 \
  -workers=256 > /tmp/node3.log 2>&1 &
NODE3_PID=$!
echo "   Node 3 PID: $NODE3_PID"

# Wait for cluster to stabilize
echo ""
echo "⏳ Waiting for cluster to stabilize..."
sleep 10

# Check cluster status
echo ""
echo "📊 Cluster Status:"
curl -s http://localhost:8080/cluster 2>/dev/null | jq '{
  nodes: .cluster.nodes | length,
  partitions: .cluster.partition_map.partitions | length,
  replication: .cluster.config.replication_factor
}' 2>/dev/null || echo "   Cluster API not available yet"

echo ""
echo "📊 Running benchmark..."
echo "   Concurrency: 256 workers per node (768 total)"
echo "   Duration: 10 seconds per node"
echo "   Value size: 1KB"
echo ""

# Create benchmark program using pool client
cat > /tmp/flin_3node_bench.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

func runWriteTest(poolClient *client.PoolClient, concurrency int, duration time.Duration) (int64, time.Duration) {
	value := make([]byte, 1024)
	var totalOps atomic.Int64
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
				if err := poolClient.Set(key, value); err == nil {
					ops++
				}
			}
			totalOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	return totalOps.Load(), time.Since(startTime)
}

func runReadTest(poolClient *client.PoolClient, concurrency int, duration time.Duration) (int64, time.Duration) {
	var totalOps atomic.Int64
	var wg sync.WaitGroup
	
	startTime := time.Now()
	stopTime := startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("key_%d_%d", workerID, ops%1000)
				if _, err := poolClient.Get(key); err == nil {
					ops++
				}
			}
			totalOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	return totalOps.Load(), time.Since(startTime)
}

func main() {
	// Use pool client connecting to all 3 nodes
	config := &client.PoolConfig{
		NodeAddrs:      []string{"localhost:6380", "localhost:6381", "localhost:6382"},
		PartitionCount: 256,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   10 * time.Second,
		ReconnectDelay: 1 * time.Second,
	}
	
	poolClient, err := client.NewPoolClient(config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer poolClient.Close()
	
	// Wait for connections
	time.Sleep(3 * time.Second)
	
	stats := poolClient.Stats()
	fmt.Printf("Connected: %d/%d connections\n\n", stats["connected"], stats["total_connections"])
	
	concurrency := 256
	duration := 10 * time.Second
	
	// Write test
	fmt.Println("🔴 WRITE Test (SET operations)")
	fmt.Println("--------------------------------")
	writeOps, writeElapsed := runWriteTest(poolClient, concurrency, duration)
	writeThroughput := float64(writeOps) / writeElapsed.Seconds()
	writeLatency := (writeElapsed.Seconds() * 1000000) / float64(writeOps)
	
	fmt.Printf("   Operations:  %.2fM\n", float64(writeOps)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", writeThroughput/1000)
	fmt.Printf("   Latency:     %.2fμs\n\n", writeLatency)
	
	// Wait a bit
	time.Sleep(2 * time.Second)
	
	// Read test
	fmt.Println("🟢 READ Test (GET operations)")
	fmt.Println("--------------------------------")
	readOps, readElapsed := runReadTest(poolClient, concurrency, duration)
	readThroughput := float64(readOps) / readElapsed.Seconds()
	readLatency := (readElapsed.Seconds() * 1000000) / float64(readOps)
	
	fmt.Printf("   Operations:  %.2fM\n", float64(readOps)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", readThroughput/1000)
	fmt.Printf("   Latency:     %.2fμs\n\n", readLatency)
	
	// Summary
	fmt.Println("📊 3-Node Cluster Summary")
	fmt.Println("=========================")
	fmt.Printf("   WRITE:  %.2fK ops/sec (%.2fμs latency)\n", writeThroughput/1000, writeLatency)
	fmt.Printf("   READ:   %.2fK ops/sec (%.2fμs latency)\n", readThroughput/1000, readLatency)
	fmt.Printf("   Ratio:  %.2fx (reads vs writes)\n", readThroughput/writeThroughput)
	
	// Comparison with single node
	singleNodeWrite := 144.83 // From single node test
	singleNodeRead := 205.10
	
	fmt.Println("\n🎯 vs Single Node:")
	fmt.Println("=========================")
	fmt.Printf("   Single node WRITE: %.2fK ops/sec\n", singleNodeWrite)
	fmt.Printf("   Cluster WRITE:     %.2fK ops/sec (%.1fx)\n", writeThroughput/1000, (writeThroughput/1000)/singleNodeWrite)
	fmt.Printf("   Single node READ:  %.2fK ops/sec\n", singleNodeRead)
	fmt.Printf("   Cluster READ:      %.2fK ops/sec (%.1fx)\n", readThroughput/1000, (readThroughput/1000)/singleNodeRead)
	
	if readThroughput/1000 > singleNodeRead {
		improvement := ((readThroughput/1000) - singleNodeRead) / singleNodeRead * 100
		fmt.Printf("\n🎉 Cluster reads are %.1f%% faster! Distributed reads FTW! 🚀\n", improvement)
	}
}
EOF

go run /tmp/flin_3node_bench.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null
wait $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null
rm -rf ./cluster-test-data
rm -f /tmp/flin_3node_bench.go /tmp/node*.log

echo ""
echo "✅ Test complete!"
