#!/bin/bash

echo "🚀 Flin Document Store Throughput Benchmark"
echo "==========================================="
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Configuration
CONCURRENCY=${1:-128}
DURATION=${2:-10}
DOC_SIZE=${3:-512}

echo "📊 Configuration:"
echo "   Concurrency: $CONCURRENCY workers"
echo "   Duration: ${DURATION}s"
echo "   Document size: ${DOC_SIZE} bytes"
echo ""

# Build the server if needed
# Build the server
echo "📦 Building Flin server..."
mkdir -p bin
go build -o bin/flin-server ./cmd/server

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

# Create benchmark program using the document SDK
mkdir -p /tmp/flin_db_bench
cd /tmp/flin_db_bench

cat > go.mod << MODEOF
module flin-db-bench

go 1.21

require github.com/skshohagmiah/flin v0.0.0

replace github.com/skshohagmiah/flin => $SCRIPT_DIR/..
MODEOF

cat > main.go << 'EOF'
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	concurrency := CONCURRENCY_PLACEHOLDER
	duration := DURATION_PLACEHOLDER * time.Second
	docSize := DOC_SIZE_PLACEHOLDER
	
	// Create client
	opts := flin.DefaultOptions("localhost:7380")
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.Close()
	
	fmt.Println("📝 Document Store Throughput Test")
	fmt.Println("==================================")
	fmt.Println()
	
	fmt.Println("📝 Document Store Throughput Test")
	fmt.Println("==================================")
	fmt.Println()
	
    // Register Schema (FQL) for optimization
    fmt.Println("📋 Registering FQL Schema...")
    err = client.DB.RegisterFQL(`
        CREATE TABLE docs (
            worker int,
            counter int,
            data text,
            timestamp int,
            PRIMARY KEY (worker, counter)
        );
    `)
    if err != nil {
        fmt.Printf("⚠️ Failed to register schema (might already exist): %v\n", err)
    } else {
        fmt.Println("✅ Schema registered: PRIMARY KEY (worker, counter)")
    }
    fmt.Println()

	// Run CREATE test
	fmt.Println("🟡 CREATE Test (document inserts)")
	fmt.Println("-----------------------------------")
	
	var createOps atomic.Int64
	var wg sync.WaitGroup
	
	startTime := time.Now()
	stopTime := startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			errors := int64(0)
			var lastErr error
			for time.Now().Before(stopTime) {
				doc := map[string]interface{}{
					"worker":  workerID,
					"counter": ops,
					"data":    randString(docSize),
					"timestamp": time.Now().Unix(),
				}
				_, err := client.DB.Insert("docs", doc)
				if err == nil {
					ops++
				} else {
					errors++
					if errors == 1 {
						lastErr = err
					}
				}
			}
			if errors > 0 && workerID == 0 {
				fmt.Printf("   [Worker %d] Errors: %d, Last error: %v\n", workerID, errors, lastErr)
			}
			createOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	createElapsed := time.Since(startTime)
	createTotal := createOps.Load()
	createThroughput := float64(createTotal) / createElapsed.Seconds()
	createLatency := (createElapsed.Seconds() * 1000000) / float64(createTotal)
	
	createThroughputStr := formatThroughput(createThroughput)
	createTotalStr := formatNumber(float64(createTotal))
	
	fmt.Printf("   Operations:  %s\n", createTotalStr)
	fmt.Printf("   Throughput:  %s docs/sec\n", createThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\n", createLatency)
	fmt.Println()
	
	// Create index on "worker" field for better query performance
	// NOTE: Index creation disabled due to timeout issue
	// fmt.Println("📇 Creating index on 'worker' field...")
	// err = client.DB.CreateIndex("docs", "worker")
	// if err != nil {
	// 	fmt.Printf("   ⚠️  Index creation error (non-critical): %v\n", err)
	// } else {
	// 	fmt.Println("   ✅ Index created successfully")
	// }
	// fmt.Println()
	
	// Run READ test
	fmt.Println("🟢 READ Test (document queries)")
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
				// Query documents by worker (reduced result set for better throughput)
				_, err := client.DB.Query("docs").
					Where("worker", flin.Eq, workerID).
					Skip(0).
					Take(5).
					Exec()
				if err == nil {
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
	
	readThroughputStr := formatThroughput(readThroughput)
	readTotalStr := formatNumber(float64(readTotal))
	
	fmt.Printf("   Operations:  %s\n", readTotalStr)
	fmt.Printf("   Throughput:  %s queries/sec\n", readThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\n", readLatency)
	fmt.Println()
	
	// Run UPDATE test
	fmt.Println("🔵 UPDATE Test (document updates)")
	fmt.Println("---------------------------------")
	
	var updateOps atomic.Int64
	
	startTime = time.Now()
	stopTime = startTime.Add(duration)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			ops := int64(0)
			counter := int64(0)
			for time.Now().Before(stopTime) {
				err := client.DB.Update("docs").
					Where("worker", flin.Eq, workerID).
					Set("counter", counter).
					Exec()
				if err == nil {
					ops++
					counter++
				}
			}
			updateOps.Add(ops)
		}(i)
	}
	
	wg.Wait()
	updateElapsed := time.Since(startTime)
	updateTotal := updateOps.Load()
	updateThroughput := float64(updateTotal) / updateElapsed.Seconds()
	updateLatency := (updateElapsed.Seconds() * 1000000) / float64(updateTotal)
	
	updateThroughputStr := formatThroughput(updateThroughput)
	updateTotalStr := formatNumber(float64(updateTotal))
	
	fmt.Printf("   Operations:  %s\n", updateTotalStr)
	fmt.Printf("   Throughput:  %s updates/sec\n", updateThroughputStr)
	fmt.Printf("   Latency:     %.2fμs\n", updateLatency)
	fmt.Println()
	
	// Summary
	fmt.Println("📊 Performance Summary")
	fmt.Println("=====================")
	fmt.Printf("   CREATE:  %s docs/sec (%.2fμs latency)\n", createThroughputStr, createLatency)
	fmt.Printf("   READ:    %s queries/sec (%.2fμs latency)\n", readThroughputStr, readLatency)
	fmt.Printf("   UPDATE:  %s updates/sec (%.2fμs latency)\n", updateThroughputStr, updateLatency)
	fmt.Println()
	
	// Compare operations
	readCreateRatio := readThroughput / createThroughput
	updateCreateRatio := updateThroughput / createThroughput
	
	fmt.Printf("   Reads:   %.2fx faster than creates 🔥\n", readCreateRatio)
	fmt.Printf("   Updates: %.2fx faster than creates ⚡\n", updateCreateRatio)
}

func formatThroughput(throughput float64) string {
	if throughput >= 1000000 {
		return fmt.Sprintf("%.2fM", throughput/1000000)
	} else if throughput >= 1000 {
		return fmt.Sprintf("%.2fK", throughput/1000)
	}
	return fmt.Sprintf("%.0f", throughput)
}

func formatNumber(num float64) string {
	if num >= 1000000 {
		return fmt.Sprintf("%.2fM", num/1000000)
	} else if num >= 1000 {
		return fmt.Sprintf("%.2fK", num/1000)
	}
	return fmt.Sprintf("%.0f", num)
}

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
EOF

# Replace placeholders
perl -pi -e "s/CONCURRENCY_PLACEHOLDER/$CONCURRENCY/g" main.go
perl -pi -e "s/DURATION_PLACEHOLDER/$DURATION/g" main.go
perl -pi -e "s/DOC_SIZE_PLACEHOLDER/$DOC_SIZE/g" main.go

echo "📊 Running document store throughput benchmark..."
echo ""

cd /tmp/flin_db_bench
go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $SERVER_PID 2>/dev/null
rm -rf /tmp/flin_db_bench
rm -rf ./data/bench

echo "✅ Benchmark complete!"
