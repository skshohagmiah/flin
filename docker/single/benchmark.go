package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	// Connect to single node
	opts := flin.DefaultOptions("localhost:6380")
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client.Close()

	concurrency := 128
	duration := 30 * time.Second

	fmt.Println("📊 Running Performance Benchmark")
	fmt.Println("=================================")
	fmt.Println()

	// Write benchmark
	fmt.Printf("⚡ Write Performance (%d workers, %v duration)...\n", concurrency, duration)
	var writeOps int64
	var wg sync.WaitGroup
	startTime := time.Now()
	endTime := startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localOps := 0
			for time.Now().Before(endTime) {
				key := fmt.Sprintf("bench-%d-%d", workerID, localOps)
				if err := client.Set(key, []byte("test")); err == nil {
					localOps++
				}
			}
			atomic.AddInt64(&writeOps, int64(localOps))
		}(i)
	}

	wg.Wait()
	writeDuration := time.Since(startTime).Seconds()
	writeThroughput := float64(writeOps) / writeDuration

	fmt.Printf("  ✓ Writes: %s operations\n", formatNumber(writeOps))
	fmt.Printf("  ✓ Throughput: %s ops/sec\n", formatNumber(int64(writeThroughput)))

	// Read benchmark
	fmt.Println()
	fmt.Printf("⚡ Read Performance (%d workers, %v duration)...\n", concurrency, duration)
	var readOps int64
	startTime = time.Now()
	endTime = startTime.Add(duration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localOps := 0
			for time.Now().Before(endTime) {
				key := fmt.Sprintf("bench-%d-0", workerID)
				if _, err := client.Get(key); err == nil {
					localOps++
				}
			}
			atomic.AddInt64(&readOps, int64(localOps))
		}(i)
	}

	wg.Wait()
	readDuration := time.Since(startTime).Seconds()
	readThroughput := float64(readOps) / readDuration

	fmt.Printf("  ✓ Reads: %s operations\n", formatNumber(readOps))
	fmt.Printf("  ✓ Throughput: %s ops/sec\n", formatNumber(int64(readThroughput)))
	fmt.Println()
	fmt.Println("=================================")
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.2fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
