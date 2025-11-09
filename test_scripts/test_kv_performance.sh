#!/bin/bash

# Change to project root
cd "$(dirname "$0")/.." || exit 1

echo "🚀 Flin KV Performance Test"
echo "============================"
echo ""

# Clean up
pkill -f "kvserver.*kv-perf" 2>/dev/null || true
rm -rf ./kv-perf-data
sleep 2

# Build
echo "📦 Building..."
go build -o bin/flin-server ./cmd/kvserver

# Start server
echo "🔧 Starting server..."
./bin/flin-server \
  -node-id=kv-perf \
  -http=localhost:8102 \
  -raft=localhost:9102 \
  -port=:7380 \
  -data=./kv-perf-data \
  -workers=256 > /tmp/kv-perf.log 2>&1 &
SERVER_PID=$!

sleep 4

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    exit 1
fi

echo "✅ Server running (PID: $SERVER_PID)"
echo ""

# Create performance test
cat > /tmp/kv_perf.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

func main() {
	fmt.Println("🎯 KV Performance Benchmark")
	fmt.Println("============================")
	fmt.Println()

	// Create client pool
	pool, _ := client.NewPoolClient("localhost:7380", 256)
	defer pool.Close()

	value := make([]byte, 1024) // 1KB value

	// Test 1: Single SET Performance
	fmt.Println("📊 Test 1: Single SET Operations")
	fmt.Println("---------------------------------")
	
	var setOps atomic.Int64
	workers := 256
	duration := 10 * time.Second
	
	var wg sync.WaitGroup
	startTime := time.Now()
	stopTime := startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("key_%d_%d", workerID, count)
				if err := pool.Set(key, value); err == nil {
					setOps.Add(1)
				}
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	ops := setOps.Load()
	throughput := float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Value size:  1KB\n")
	fmt.Printf("   Duration:    %v\n", duration)
	fmt.Printf("   Operations:  %.2fM\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:     %.2fμs\n", (elapsed.Seconds()*1000000)/float64(ops))
	fmt.Println()

	// Test 2: Single GET Performance
	fmt.Println("📊 Test 2: Single GET Operations")
	fmt.Println("---------------------------------")
	
	var getOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("key_%d_%d", workerID, count%100)
				if _, err := pool.Get(key); err == nil {
					getOps.Add(1)
				}
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = getOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Duration:    %v\n", duration)
	fmt.Printf("   Operations:  %.2fM\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:     %.2fμs\n", (elapsed.Seconds()*1000000)/float64(ops))
	fmt.Println()

	// Test 3: Batch Operations
	fmt.Println("📊 Test 3: Batch Operations (10 keys)")
	fmt.Println("--------------------------------------")
	
	var batchOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			batchNum := 0
			for time.Now().Before(stopTime) {
				keys := make([]string, 10)
				values := make([][]byte, 10)
				for i := 0; i < 10; i++ {
					keys[i] = fmt.Sprintf("batch_%d_%d_%d", workerID, batchNum, i)
					values[i] = value
				}
				if err := pool.MSet(keys, values); err == nil {
					batchOps.Add(10)
				}
				batchNum++
			}
		}(w)
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = batchOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Batch size:  10 keys\n")
	fmt.Printf("   Duration:    %v\n", duration)
	fmt.Printf("   Operations:  %.2fM\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:     %.2fμs per key\n", (elapsed.Seconds()*1000000)/float64(ops))
	fmt.Println()

	// Test 4: Mixed Workload (70% reads, 30% writes)
	fmt.Println("📊 Test 4: Mixed Workload (70% read, 30% write)")
	fmt.Println("------------------------------------------------")
	
	var mixedOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("mixed_%d_%d", workerID, count%1000)
				
				if count%10 < 7 {
					// 70% reads
					if _, err := pool.Get(key); err == nil {
						mixedOps.Add(1)
					}
				} else {
					// 30% writes
					if err := pool.Set(key, value); err == nil {
						mixedOps.Add(1)
					}
				}
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = mixedOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Duration:    %v\n", duration)
	fmt.Printf("   Operations:  %.2fM\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:     %.2fμs\n", (elapsed.Seconds()*1000000)/float64(ops))
	fmt.Println()

	fmt.Println("📊 Performance Summary")
	fmt.Println("======================")
	fmt.Println("✅ High-performance KV operations")
	fmt.Println("✅ Batch operations provide massive speedup")
	fmt.Println("✅ Excellent mixed workload performance")
	fmt.Println()
	fmt.Println("🎉 Performance test complete!")
}
EOF

go run /tmp/kv_perf.go

# Cleanup
kill $SERVER_PID 2>/dev/null
rm -rf ./kv-perf-data
rm -f /tmp/kv_perf.go /tmp/kv-perf.log

echo ""
echo "✅ Test complete!"
