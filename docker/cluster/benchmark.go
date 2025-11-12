package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	// Create cluster-aware client that discovers topology
	httpAddrs := []string{
		"localhost:8080",
		"localhost:8081",
		"localhost:8082",
	}

	opts := flin.DefaultClusterOptions(httpAddrs)
	client, err := flin.NewClient(opts)
	if err != nil {
		fmt.Printf("Failed to create cluster client: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("📊 Cluster client initialized")
	fmt.Printf("   Cluster mode: %v\n", client.IsClusterMode())
	topology := client.GetTopology()
	fmt.Printf("   Nodes discovered: %d\n", len(topology.Nodes))
	fmt.Printf("   Partitions: %d\n", len(topology.PartitionMap))
	fmt.Println()

	concurrency := 128
	duration := 30 * time.Second

	fmt.Println("📊 Running Cluster Performance Benchmark")
	fmt.Println("=========================================")
	fmt.Println()

	// Write benchmark - distributed across all 3 nodes
	fmt.Printf("⚡ Write Performance (%d workers, %v duration, distributed across 3 nodes)...\n", concurrency, duration)
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
				// Client automatically routes to correct primary node
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
	fmt.Printf("  ✓ Per node: ~%s ops/sec\n", formatNumber(int64(writeThroughput/3)))

	// Wait for replication
	fmt.Println()
	fmt.Println("⏳ Waiting for replication...")
	time.Sleep(2 * time.Second)

	// Read benchmark - distributed across all 3 nodes
	fmt.Println()
	fmt.Printf("⚡ Read Performance (%d workers, %v duration, distributed across 3 nodes)...\n", concurrency, duration)
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
				// Client automatically routes to correct node (primary or replica)
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
	fmt.Printf("  ✓ Per node: ~%s ops/sec\n", formatNumber(int64(readThroughput/3)))
	fmt.Println()
	fmt.Println("=========================================")
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.2fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
