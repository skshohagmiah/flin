#!/bin/bash

echo "🚀 Flin Read/Write Performance Test"
echo "===================================="
echo ""

# Clean up
pkill -f "kvserver.*node-1" 2>/dev/null || true
rm -rf ./data/node1
sleep 2

# Build
echo "📦 Building Flin..."
go build -o bin/flin-server ./cmd/kvserver

# Start server
echo "🔧 Starting server..."
./bin/flin-server \
  -node-id=node-1 \
  -http=localhost:7080 \
  -raft=localhost:7090 \
  -port=:7380 \
  -data=./data/node1 \
  -workers=256 > /tmp/server.log 2>&1 &
SERVER_PID=$!
echo "   Server PID: $SERVER_PID"

sleep 4

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    exit 1
fi

echo "✅ Server is running"
echo ""

# Create benchmark program
cat > /tmp/flin_rw_bench.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

func runWriteTest(concurrency int, duration time.Duration) (int64, time.Duration) {
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
	return totalOps.Load(), time.Since(startTime)
}

func runReadTest(concurrency int, duration time.Duration) (int64, time.Duration) {
	var totalOps atomic.Int64
	var wg sync.WaitGroup
	
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
				key := fmt.Sprintf("key_%d_%d", workerID, ops%1000)
				if _, err := c.Get(key); err == nil {
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
	concurrency := 256
	duration := 10 * time.Second
	
	fmt.Println("📊 Performance Test")
	fmt.Println("===================")
	fmt.Printf("   Workers: %d\n", concurrency)
	fmt.Printf("   Duration: %v per test\n", duration)
	fmt.Printf("   Value size: 1KB\n\n")
	
	// Write test
	fmt.Println("🔴 WRITE Test (SET operations)")
	fmt.Println("--------------------------------")
	writeOps, writeElapsed := runWriteTest(concurrency, duration)
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
	readOps, readElapsed := runReadTest(concurrency, duration)
	readThroughput := float64(readOps) / readElapsed.Seconds()
	readLatency := (readElapsed.Seconds() * 1000000) / float64(readOps)
	
	fmt.Printf("   Operations:  %.2fM\n", float64(readOps)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", readThroughput/1000)
	fmt.Printf("   Latency:     %.2fμs\n\n", readLatency)
	
	// Summary
	fmt.Println("📊 Summary")
	fmt.Println("===================")
	fmt.Printf("   WRITE:  %.2fK ops/sec (%.2fμs latency)\n", writeThroughput/1000, writeLatency)
	fmt.Printf("   READ:   %.2fK ops/sec (%.2fμs latency)\n", readThroughput/1000, readLatency)
	fmt.Printf("   Ratio:  %.2fx (reads faster than writes)\n", readThroughput/writeThroughput)
	
	if readThroughput > writeThroughput {
		improvement := ((readThroughput - writeThroughput) / writeThroughput) * 100
		fmt.Printf("   Reads are %.1f%% faster! 🚀\n", improvement)
	}
}
EOF

echo "📊 Running benchmark..."
echo "   Testing both READ and WRITE operations"
echo ""

go run /tmp/flin_rw_bench.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null
rm -f /tmp/flin_rw_bench.go /tmp/server.log
rm -rf ./data/node1

echo "✅ Test complete!"
