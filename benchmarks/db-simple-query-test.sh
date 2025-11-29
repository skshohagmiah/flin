#!/bin/bash

echo "🔬 Flin Document Store Query Performance Test"
echo "=============================================="
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Configuration
CONCURRENCY=${1:-16}
DURATION=${2:-5}

echo "📊 Configuration:"
echo "   Concurrency: $CONCURRENCY workers"
echo "   Duration: ${DURATION}s"
echo ""

# Build the server if needed
echo "📦 Building Flin server..."
mkdir -p bin
go build -o bin/flin-server ./cmd/server 2>/dev/null

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
mkdir -p /tmp/flin_query_test
cd /tmp/flin_query_test

cat > go.mod << MODEOF
module flin-query-test

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
	docSize := 512

	// Create client
	opts := flin.DefaultOptions("localhost:7380")
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("📝 Query Performance Test")
	fmt.Println("========================")
	fmt.Println()

	// Phase 1: Insert test data
	fmt.Println("📥 Phase 1: Inserting test documents...")
	insertCount := int64(0)
	for i := 0; i < 1000; i++ {
		doc := map[string]interface{}{
			"id":        i,
			"worker":    i % concurrency,
			"data":      randString(docSize),
			"timestamp": time.Now().Unix(),
		}
		_, err := client.DB.Insert("docs", doc)
		if err == nil {
			insertCount++
		}
	}
	fmt.Printf("✓ Inserted %d documents\n", insertCount)
	fmt.Println()

	// Phase 2: Test filtered queries (WITH WHERE clause)
	fmt.Println("🟢 Test 1: FILTERED Query (WITH WHERE clause)")
	fmt.Println("---------------------------------------------")
	testQueries(client, concurrency, duration, true)
	fmt.Println()

	// Phase 3: Test simple queries (WITHOUT WHERE clause)
	fmt.Println("🟡 Test 2: SIMPLE Query (WITHOUT WHERE clause)")
	fmt.Println("---------------------------------------------")
	testQueries(client, concurrency, duration, false)
	fmt.Println()

	// Phase 4: Test single query performance
	fmt.Println("🔵 Test 3: SINGLE Query Performance")
	fmt.Println("-----------------------------------")
	testSingleQueries(client, concurrency, 20)
}

func testQueries(client *flin.Client, concurrency int, duration time.Duration, useFilter bool) {
	var queryOps atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()
	stopTime := startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ops := int64(0)
			errors := int64(0)
			for time.Now().Before(stopTime) {
				var err error
				if useFilter {
					// Query with WHERE clause
					_, err = client.DB.Query("docs").
						Where("worker", flin.Eq, workerID).
						Skip(0).
						Take(10).
						Exec()
				} else {
					// Query all documents
					_, err = client.DB.Query("docs").
						Skip(0).
						Take(100).
						Exec()
				}
				if err == nil {
					ops++
				} else {
					errors++
				}
			}
			queryOps.Add(ops)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	total := queryOps.Load()
	throughput := float64(total) / elapsed.Seconds()
	latency := (elapsed.Seconds() * 1000000) / float64(total)

	fmt.Printf("   Operations:  %d\n", total)
	fmt.Printf("   Throughput:  %.2f queries/sec\n", throughput)
	fmt.Printf("   Latency:     %.2fμs\n", latency)
}

func testSingleQueries(client *flin.Client, numQueries int, numSamples int) {
	fmt.Println("   Testing individual query times (first 20 queries):")
	fmt.Println()

	for sample := 0; sample < numSamples; sample++ {
		startTime := time.Now()

		// Test filtered query
		_, err := client.DB.Query("docs").
			Where("worker", flin.Eq, 0).
			Skip(0).
			Take(10).
			Exec()

		elapsed := time.Since(startTime)

		if err != nil {
			fmt.Printf("   Sample %d: ERROR - %v\n", sample+1, err)
		} else {
			fmt.Printf("   Sample %2d: %.2fms (WHERE clause)\n", sample+1, elapsed.Seconds()*1000)
		}

		if sample == 4 {
			fmt.Println("   ...")
			break
		}
	}
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
sed -i "s/CONCURRENCY_PLACEHOLDER/$CONCURRENCY/g" main.go
sed -i "s/DURATION_PLACEHOLDER/$DURATION/g" main.go

echo "📊 Running query performance tests..."
echo ""

go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
kill $SERVER_PID 2>/dev/null
rm -rf /tmp/flin_query_test
rm -rf ./data/bench

echo "✅ Test complete!"
