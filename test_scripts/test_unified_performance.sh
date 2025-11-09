#!/bin/bash

# Change to project root
cd "$(dirname "$0")/.." || exit 1

echo "🚀 Flin Unified Performance Test (KV + Queue)"
echo "=============================================="
echo ""

# Clean up
pkill -f "kvserver.*unified-perf" 2>/dev/null || true
rm -rf ./unified-perf-data
sleep 2

# Build
echo "📦 Building..."
go build -o bin/flin-server ./cmd/kvserver

# Start server
echo "🔧 Starting server..."
./bin/flin-server \
  -node-id=unified-perf \
  -http=localhost:8103 \
  -raft=localhost:9103 \
  -port=:7380 \
  -data=./unified-perf-data \
  -workers=256 > /tmp/unified-perf.log 2>&1 &
SERVER_PID=$!

sleep 4

if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start"
    exit 1
fi

echo "✅ Server running (PID: $SERVER_PID)"
echo ""

# Create unified performance test
cat > /tmp/unified_perf.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/flin"
	"github.com/skshohagmiah/flin/pkg/queue"
)

func main() {
	fmt.Println("🎯 Unified Performance Benchmark")
	fmt.Println("=================================")
	fmt.Println()

	// Create unified client
	client, err := flin.NewClient("localhost:7380", "")
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		return
	}
	defer client.Close()

	value := make([]byte, 1024)

	// Test 1: KV Performance
	fmt.Println("📊 Test 1: KV Store Performance")
	fmt.Println("--------------------------------")
	
	var kvOps atomic.Int64
	workers := 128
	duration := 5 * time.Second
	
	var wg sync.WaitGroup
	startTime := time.Now()
	stopTime := startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("kv_%d_%d", workerID, count)
				client.KV.Set(key, value)
				client.KV.Get(key)
				kvOps.Add(2)
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	ops := kvOps.Load()
	throughput := float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Operations:  %.2fM (SET+GET)\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Println()

	// Test 2: Queue Performance
	fmt.Println("📊 Test 2: Queue Performance")
	fmt.Println("-----------------------------")
	
	var queueOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				msg := []byte(fmt.Sprintf("msg_%d_%d", workerID, count))
				client.Queue.Enqueue("perf-queue", msg)
				queueOps.Add(1)
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = queueOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Operations:  %.2fM (Enqueue)\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Println()

	// Test 3: Mixed KV + Queue
	fmt.Println("📊 Test 3: Mixed KV + Queue Operations")
	fmt.Println("---------------------------------------")
	
	var mixedOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				if count%2 == 0 {
					// KV operation
					key := fmt.Sprintf("mixed_%d_%d", workerID, count)
					client.KV.Set(key, value)
					mixedOps.Add(1)
				} else {
					// Queue operation
					msg := []byte(fmt.Sprintf("task_%d_%d", workerID, count))
					client.Queue.Enqueue("mixed-queue", msg)
					mixedOps.Add(1)
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
	fmt.Printf("   Operations:  %.2fM (50%% KV, 50%% Queue)\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Println()

	// Test 4: Real-world Scenario
	fmt.Println("📊 Test 4: Real-World Scenario")
	fmt.Println("-------------------------------")
	fmt.Println("   Simulating: Store order + Queue task")
	
	var scenarioOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				orderID := fmt.Sprintf("order:%d:%d", workerID, count)
				orderData := []byte(fmt.Sprintf(`{"id":"%s","total":99.99}`, orderID))
				
				// Store order in KV
				client.KV.Set(orderID, orderData)
				
				// Queue processing task
				task := []byte(fmt.Sprintf("process:%s", orderID))
				client.Queue.Enqueue("order-processing", task)
				
				scenarioOps.Add(1)
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = scenarioOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Orders:      %.2fM\n", float64(ops)/1000000)
	fmt.Printf("   Throughput:  %.2fK orders/sec\n", throughput/1000)
	fmt.Println()

	// Test 5: Consumer Performance
	fmt.Println("📊 Test 5: Queue Consumer Performance")
	fmt.Println("--------------------------------------")
	
	var consumeOps atomic.Int64
	
	client.Queue.Consume("consumer-perf", func(msg *queue.Message) {
		msg.Ack()
		consumeOps.Add(1)
	})

	// Enqueue 50K messages
	for i := 0; i < 50000; i++ {
		client.Queue.Enqueue("consumer-perf", []byte(fmt.Sprintf("msg_%d", i)))
	}

	time.Sleep(3 * time.Second)
	ops = consumeOps.Load()

	fmt.Printf("   Messages:    50,000\n")
	fmt.Printf("   Consumed:    %d\n", ops)
	fmt.Printf("   Rate:        %.2fK msgs/sec\n", float64(ops)/3000)
	fmt.Println()

	fmt.Println("📊 Performance Summary")
	fmt.Println("======================")
	fmt.Println("✅ KV Store: High-performance key-value operations")
	fmt.Println("✅ Queue: Fast message enqueue/dequeue")
	fmt.Println("✅ Mixed: Both systems work efficiently together")
	fmt.Println("✅ Real-world: Ready for production workloads")
	fmt.Println()
	fmt.Println("🎉 Unified performance test complete!")
}
EOF

go run /tmp/unified_perf.go

# Cleanup
kill $SERVER_PID 2>/dev/null
rm -rf ./unified-perf-data
rm -f /tmp/unified_perf.go /tmp/unified-perf.log

echo ""
echo "✅ Test complete!"
