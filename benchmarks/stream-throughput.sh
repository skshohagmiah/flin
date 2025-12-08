#!/bin/bash

echo "🚀 Flin Stream Throughput Benchmark"
echo "==================================="
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Configuration
CONCURRENCY=${1:-16}  # Reduced to 16 for safety
DURATION=${2:-5}
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
  -workers=256 > server.log 2>&1 &

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
mkdir -p /tmp/flin_stream_bench
cd /tmp/flin_stream_bench

cat > go.mod << MODEOF
module flin-stream-bench

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

	fmt.Printf("Unified server (KV + Queue + Stream): localhost:7380\\n")
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

	if client.Stream == nil {
		fmt.Println("❌ Stream client is nil!")
		return
	}

	// Prepare test data
	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 256)
	}

	// Create 1 topic with concurrency-based partitions for maximum parallelism
	// (each worker gets dedicated partition to eliminate lock contention)
	numPartitions := concurrency
	fmt.Println("📝 Creating topics...")
	topicName := "bench_topic_0"
	err = client.Stream.CreateTopic(topicName, numPartitions, 0)
	if err != nil {
		fmt.Printf("Warning: Failed to create topic: %v\\n", err)
	}
	time.Sleep(500 * time.Millisecond) // Wait for topic creation

	// Run PUBLISH test
	fmt.Println("🔴 PUBLISH Test (Append operations)")
	fmt.Println("----------------------------------")

	var pubOps atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()
	stopTime := startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker gets its own partition to eliminate lock contention
			topicName := "bench_topic_0"
			partition := workerID % numPartitions
			ops := int64(0)
			
			// Batch configuration
			batchSize := 50
			batch := make([]*flin.BatchMessage, 0, batchSize)
			
			for time.Now().Before(stopTime) {
				batch = append(batch, &flin.BatchMessage{
					Partition: partition,
					Key:       "",
					Value:     value,
				})
				
				if len(batch) >= batchSize {
					if err := client.PublishBatch(topicName, batch); err == nil {
						ops += int64(len(batch))
					}
					batch = batch[:0]
				}
			}
			pubOps.Add(ops)
		}(i)
	}

	wg.Wait()
	pubElapsed := time.Since(startTime)
	pubTotal := pubOps.Load()
	pubThroughput := float64(pubTotal) / pubElapsed.Seconds()
	pubLatency := (pubElapsed.Seconds() * 1000000) / float64(pubTotal)

	// Format publish results
	var pubThroughputStr string
	if pubThroughput >= 1000000 {
		pubThroughputStr = fmt.Sprintf("%.2fM", pubThroughput/1000000)
	} else if pubThroughput >= 1000 {
		pubThroughputStr = fmt.Sprintf("%.2fK", pubThroughput/1000)
	} else {
		pubThroughputStr = fmt.Sprintf("%.0f", pubThroughput)
	}

	var pubTotalStr string
	if pubTotal >= 1000000 {
		pubTotalStr = fmt.Sprintf("%.2fM", float64(pubTotal)/1000000)
	} else if pubTotal >= 1000 {
		pubTotalStr = fmt.Sprintf("%.2fK", float64(pubTotal)/1000)
	} else {
		pubTotalStr = fmt.Sprintf("%d", pubTotal)
	}

	fmt.Printf("   Operations:  %s\\n", pubTotalStr)
	fmt.Printf("   Throughput:  %s ops/sec\\n", pubThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\\n", pubLatency)
	fmt.Println()

	// Run CONSUME test
	fmt.Println("🟢 CONSUME Test (Fetch operations)")
	fmt.Println("----------------------------------")

	var subOps atomic.Int64
	
	// Subscribe first (each worker subscribes to its assigned partition)
	for i := 0; i < concurrency; i++ {
		topicName := "bench_topic_0"
		partition := i % numPartitions
		groupName := fmt.Sprintf("bench_group_%d", partition)
		consumerName := fmt.Sprintf("consumer_%d", i)
		client.Stream.Subscribe(topicName, groupName, consumerName)
	}

	startTime = time.Now()
	stopTime = startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker consumes from its assigned partition
			topicName := "bench_topic_0"
			partition := workerID % numPartitions
			groupName := fmt.Sprintf("bench_group_%d", partition)
			consumerName := fmt.Sprintf("consumer_%d", workerID)
			
			ops := int64(0)
			for time.Now().Before(stopTime) {
				msgs, err := client.Stream.Consume(topicName, groupName, consumerName, 10)
				if err == nil {
					ops += int64(len(msgs))
					if len(msgs) == 0 {
						// If empty, publish some more or sleep briefly
						// time.Sleep(1 * time.Millisecond)
					}
				} else {
					fmt.Printf("Error consuming: %v\\n", err)
				}
			}
			subOps.Add(ops)
		}(i)
	}

	wg.Wait()
	subElapsed := time.Since(startTime)
	subTotal := subOps.Load()
	subThroughput := float64(subTotal) / subElapsed.Seconds()
	subLatency := (subElapsed.Seconds() * 1000000) / float64(subTotal)

	// Format consume results
	var subThroughputStr string
	if subThroughput >= 1000000 {
		subThroughputStr = fmt.Sprintf("%.2fM", subThroughput/1000000)
	} else if subThroughput >= 1000 {
		subThroughputStr = fmt.Sprintf("%.2fK", subThroughput/1000)
	} else {
		subThroughputStr = fmt.Sprintf("%.0f", subThroughput)
	}

	var subTotalStr string
	if subTotal >= 1000000 {
		subTotalStr = fmt.Sprintf("%.2fM", float64(subTotal)/1000000)
	} else if subTotal >= 1000 {
		subTotalStr = fmt.Sprintf("%.2fK", float64(subTotal)/1000)
	} else {
		subTotalStr = fmt.Sprintf("%d", subTotal)
	}

	fmt.Printf("   Operations:  %s\\n", subTotalStr)
	fmt.Printf("   Throughput:  %s ops/sec\\n", subThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\\n", subLatency)
	fmt.Println()

	// Summary
	fmt.Println("📊 Summary")
	fmt.Println("===================")
	fmt.Printf("   PUBLISH: %s ops/sec (%.2fμs latency)\\n", pubThroughputStr, pubLatency)
	fmt.Printf("   CONSUME: %s ops/sec (%.2fμs latency)\\n", subThroughputStr, subLatency)

	avgThroughput := (pubThroughput + subThroughput) / 2
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

echo "📊 Running stream throughput benchmark..."
echo ""

cd /tmp/flin_stream_bench
go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $SERVER_PID 2>/dev/null
rm -rf /tmp/flin_stream_bench
rm -rf ./data/bench

echo "✅ Benchmark complete!"
