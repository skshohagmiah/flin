#!/bin/bash

echo "🚀 Flin Pure KV Benchmark (No Cluster Overhead)"
echo "================================================"
echo ""

# Configuration
CONCURRENCY=${1:-256}
DURATION=${2:-30}
VALUE_SIZE=${3:-1024}

echo "📊 Configuration:"
echo "   Concurrency: $CONCURRENCY workers"
echo "   Duration: ${DURATION}s"
echo "   Value size: ${VALUE_SIZE} bytes"
echo "   Mode: Pure KV (no cluster/raft overhead)"
echo ""

# Save current directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Create benchmark program
mkdir -p /tmp/flin_pure_bench
cd /tmp/flin_pure_bench

cat > go.mod << 'MODEOF'
module flin-pure-bench

go 1.21

require github.com/skshohagmiah/flin v0.0.0

replace github.com/skshohagmiah/flin => REPLACE_PATH
MODEOF

# Replace the path
sed -i "s|REPLACE_PATH|$SCRIPT_DIR/..|g" go.mod

cat > main.go << 'EOF'
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/internal/kv"
)

func main() {
	concurrency := CONCURRENCY_VAL
	duration := DURATION_VAL * time.Second
	valueSize := VALUE_SIZE_VAL
	
	fmt.Println("🔧 Creating pure KV store (disk-based)...")
	
	// Create pure KV store without cluster overhead
	store, err := kv.New("/tmp/flin_pure_bench_data")
	if err != nil {
		fmt.Printf("Failed to create store: %v\n", err)
		return
	}
	defer store.Close()
	
	fmt.Println("✅ Store created")
	fmt.Println()
	
	// Prepare test data
	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 256)
	}
	
	// Warmup
	fmt.Println("🔥 Warming up...")
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("warmup_%d", i)
		store.Set(key, value, 0)
	}
	fmt.Println()
	
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
				if err := store.Set(key, value, 0); err == nil {
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
				if _, err := store.Get(key); err == nil {
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
	
	fmt.Println()
	fmt.Println("💡 This is PURE KV performance (no network, no cluster overhead)")
	fmt.Println("   For networked performance, expect 60-70% of these numbers")
}
EOF

# Replace placeholders
sed -i "s/CONCURRENCY_VAL/$CONCURRENCY/g" main.go
sed -i "s/DURATION_VAL/$DURATION/g" main.go
sed -i "s/VALUE_SIZE_VAL/$VALUE_SIZE/g" main.go

echo "📊 Running pure KV benchmark..."
echo ""

go mod tidy 2>/dev/null
go run main.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
cd "$SCRIPT_DIR/.."
rm -rf /tmp/flin_pure_bench
rm -rf /tmp/flin_pure_bench_data

echo "✅ Benchmark complete!"
