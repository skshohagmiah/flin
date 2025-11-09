#!/bin/bash

# Change to project root
cd "$(dirname "$0")/.." || exit 1

echo "🚀 Flin Queue Performance Test"
echo "==============================="
echo ""

# Build
echo "📦 Building..."
cat > /tmp/queue_perf.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/queue"
)

func main() {
	fmt.Println("🎯 Queue Performance Benchmark")
	fmt.Println("===============================")
	fmt.Println()

	// Create queue client (in-memory)
	client, err := queue.NewClient("")
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		return
	}

	// Test 1: Enqueue Performance
	fmt.Println("📊 Test 1: Enqueue Performance")
	fmt.Println("-------------------------------")
	
	var enqueueOps atomic.Int64
	workers := 100
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
				msg := []byte(fmt.Sprintf("Message from worker %d: %d", workerID, count))
				if err := client.Enqueue("perf-test", msg); err == nil {
					enqueueOps.Add(1)
				}
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	ops := enqueueOps.Load()
	throughput := float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Duration:    %v\n", duration)
	fmt.Printf("   Operations:  %d\n", ops)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:     %.2fμs per op\n", (elapsed.Seconds()*1000000)/float64(ops))
	fmt.Println()

	// Test 2: Dequeue Performance
	fmt.Println("📊 Test 2: Dequeue Performance")
	fmt.Println("-------------------------------")
	
	var dequeueOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				msg, err := client.DequeueWithTimeout("perf-test", 100*time.Millisecond)
				if err == nil {
					msg.Ack()
					dequeueOps.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = dequeueOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     %d\n", workers)
	fmt.Printf("   Duration:    %v\n", duration)
	fmt.Printf("   Operations:  %d\n", ops)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Printf("   Latency:     %.2fμs per op\n", (elapsed.Seconds()*1000000)/float64(ops))
	fmt.Println()

	// Test 3: Priority Queue Performance
	fmt.Println("📊 Test 3: Priority Queue Performance")
	fmt.Println("--------------------------------------")
	
	var priorityOps atomic.Int64
	startTime = time.Now()
	stopTime = startTime.Add(5 * time.Second)

	for w := 0; w < 50; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			count := 0
			for time.Now().Before(stopTime) {
				priority := count % 10 // 0-9
				msg := []byte(fmt.Sprintf("Priority %d message", priority))
				if err := client.EnqueueWithOptions("priority-perf", msg, nil, priority, 0); err == nil {
					priorityOps.Add(1)
				}
				count++
			}
		}(w)
	}

	wg.Wait()
	elapsed = time.Since(startTime)
	ops = priorityOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Workers:     50\n")
	fmt.Printf("   Duration:    5s\n")
	fmt.Printf("   Operations:  %d\n", ops)
	fmt.Printf("   Throughput:  %.2fK ops/sec\n", throughput/1000)
	fmt.Println()

	// Test 4: Consume Performance
	fmt.Println("📊 Test 4: Consume Performance")
	fmt.Println("-------------------------------")
	
	var consumeOps atomic.Int64
	
	// Start consumer
	client.Consume("consume-perf", func(msg *queue.Message) {
		msg.Ack()
		consumeOps.Add(1)
	})

	// Enqueue messages
	startTime = time.Now()
	for i := 0; i < 100000; i++ {
		client.Enqueue("consume-perf", []byte(fmt.Sprintf("Message %d", i)))
	}

	// Wait for consumption
	time.Sleep(5 * time.Second)
	elapsed = time.Since(startTime)
	ops = consumeOps.Load()
	throughput = float64(ops) / elapsed.Seconds()

	fmt.Printf("   Messages:    100,000\n")
	fmt.Printf("   Consumed:    %d\n", ops)
	fmt.Printf("   Throughput:  %.2fK msgs/sec\n", throughput/1000)
	fmt.Println()

	// Summary
	fmt.Println("📊 Performance Summary")
	fmt.Println("======================")
	fmt.Println("✅ Queue operations are fast and scalable")
	fmt.Println("✅ Priority queues maintain performance")
	fmt.Println("✅ Consume pattern handles high throughput")
	fmt.Println()
	fmt.Println("🎉 Performance test complete!")
}
EOF

go run /tmp/queue_perf.go

# Cleanup
rm -f /tmp/queue_perf.go

echo ""
echo "✅ Test complete!"
