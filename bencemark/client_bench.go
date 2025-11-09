package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

const (
	testDuration = 10 * time.Second
	valueSize    = 1024 // 1KB
)

type BenchResult struct {
	Operation    string
	TotalOps     int64
	Duration     time.Duration
	OpsPerSecond float64
	AvgLatencyUs float64
	Concurrency  int
}

func main() {
	// Get server address from environment or use default
	serverAddr := os.Getenv("FLIN_SERVER")
	if serverAddr == "" {
		serverAddr = "localhost:6380"
	}

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Flin Client Benchmark (Network Performance)              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n🌐 Server: %s\n", serverAddr)
	fmt.Printf("⏱  Test Duration: %v per operation\n", testDuration)
	fmt.Printf("📦 Value Size: %d bytes\n\n", valueSize)

	// Test connection
	fmt.Printf("📡 Testing connection to server...\n")
	testClient, err := client.New(serverAddr)
	if err != nil {
		fmt.Printf("❌ Failed to connect to server: %v\n", err)
		fmt.Printf("\n💡 Make sure Flin server is running:\n")
		fmt.Printf("   docker compose up -d flin-server\n")
		fmt.Printf("   OR\n")
		fmt.Printf("   ./kvserver\n\n")
		os.Exit(1)
	}

	// Test with a simple SET operation
	testErr := testClient.Set("__test__", []byte("test"))
	testClient.Delete("__test__")
	testClient.Close()

	if testErr != nil {
		fmt.Printf("❌ Server is not responding: %v\n", testErr)
		os.Exit(1)
	}
	fmt.Printf("✅ Connected successfully\n\n")

	// Test different concurrency levels
	concurrencyLevels := []int{1, 4, 8, 16}

	for _, concurrency := range concurrencyLevels {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("🔧 Concurrency: %d connections\n", concurrency)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Benchmark SET
		setResult := benchmarkSet(serverAddr, concurrency)
		printResult(setResult)

		// Benchmark GET
		getResult := benchmarkGet(serverAddr, concurrency)
		printResult(getResult)

		// Benchmark Mixed
		mixedResult := benchmarkMixed(serverAddr, concurrency)
		printResult(mixedResult)

		fmt.Println()
		time.Sleep(1 * time.Second)
	}

	fmt.Println("✅ Benchmark completed!")
	fmt.Println("\n📊 Network Performance Notes:")
	fmt.Println("   - Network adds ~100-500μs latency vs embedded mode")
	fmt.Println("   - Connection pooling improves throughput significantly")
	fmt.Println("   - Results depend on network conditions and Docker overhead")
}

func benchmarkSet(serverAddr string, concurrency int) BenchResult {
	var totalOps atomic.Int64
	var totalLatency atomic.Int64
	var wg sync.WaitGroup

	value := make([]byte, valueSize)
	for i := range value {
		value[i] = byte(i % 256)
	}

	startTime := time.Now()
	stopTime := startTime.Add(testDuration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker gets its own client
			c, err := client.New(serverAddr)
			if err != nil {
				fmt.Printf("Worker %d failed to connect: %v\n", workerID, err)
				return
			}
			defer c.Close()

			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("bench_set_%d_%d", workerID, ops)

				opStart := time.Now()
				err := c.Set(key, value)
				opLatency := time.Since(opStart)

				if err != nil {
					continue
				}

				totalLatency.Add(opLatency.Microseconds())
				ops++
			}
			totalOps.Add(ops)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	total := totalOps.Load()
	avgLatency := float64(totalLatency.Load()) / float64(total)

	return BenchResult{
		Operation:    "SET",
		TotalOps:     total,
		Duration:     duration,
		OpsPerSecond: float64(total) / duration.Seconds(),
		AvgLatencyUs: avgLatency,
		Concurrency:  concurrency,
	}
}

func benchmarkGet(serverAddr string, concurrency int) BenchResult {
	// Pre-populate keys
	setupClient, _ := client.New(serverAddr)
	value := make([]byte, valueSize)
	numKeys := 10000

	fmt.Printf("   Populating %d keys...\n", numKeys)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("bench_get_%d", i)
		setupClient.Set(key, value)
	}
	setupClient.Close()

	var totalOps atomic.Int64
	var totalLatency atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()
	stopTime := startTime.Add(testDuration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			c, err := client.New(serverAddr)
			if err != nil {
				return
			}
			defer c.Close()

			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("bench_get_%d", ops%int64(numKeys))

				opStart := time.Now()
				_, err := c.Get(key)
				opLatency := time.Since(opStart)

				if err != nil {
					continue
				}

				totalLatency.Add(opLatency.Microseconds())
				ops++
			}
			totalOps.Add(ops)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	total := totalOps.Load()
	avgLatency := float64(totalLatency.Load()) / float64(total)

	return BenchResult{
		Operation:    "GET",
		TotalOps:     total,
		Duration:     duration,
		OpsPerSecond: float64(total) / duration.Seconds(),
		AvgLatencyUs: avgLatency,
		Concurrency:  concurrency,
	}
}

func benchmarkMixed(serverAddr string, concurrency int) BenchResult {
	// Pre-populate keys
	setupClient, _ := client.New(serverAddr)
	value := make([]byte, valueSize)
	numKeys := 10000

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("bench_mixed_%d", i)
		setupClient.Set(key, value)
	}
	setupClient.Close()

	var totalOps atomic.Int64
	var totalLatency atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()
	stopTime := startTime.Add(testDuration)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			c, err := client.New(serverAddr)
			if err != nil {
				return
			}
			defer c.Close()

			ops := int64(0)
			for time.Now().Before(stopTime) {
				key := fmt.Sprintf("bench_mixed_%d", ops%int64(numKeys))

				opStart := time.Now()

				// 70% GET, 30% SET
				if ops%10 < 7 {
					_, err = c.Get(key)
				} else {
					err = c.Set(key, value)
				}

				opLatency := time.Since(opStart)

				if err != nil {
					continue
				}

				totalLatency.Add(opLatency.Microseconds())
				ops++
			}
			totalOps.Add(ops)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	total := totalOps.Load()
	avgLatency := float64(totalLatency.Load()) / float64(total)

	return BenchResult{
		Operation:    "MIXED (70% GET, 30% SET)",
		TotalOps:     total,
		Duration:     duration,
		OpsPerSecond: float64(total) / duration.Seconds(),
		AvgLatencyUs: avgLatency,
		Concurrency:  concurrency,
	}
}

func printResult(result BenchResult) {
	var icon string
	switch result.Operation {
	case "SET":
		icon = "✍️ "
	case "GET":
		icon = "📖"
	default:
		icon = "🔀"
	}

	fmt.Printf("\n%s %s\n", icon, result.Operation)
	fmt.Println("  ┌─────────────────────────────────────────────────────────┐")
	fmt.Printf("  │ Operations:  %-43s │\n", formatNumber(result.TotalOps))
	fmt.Printf("  │ Throughput:  %-43s │\n", formatOpsPerSec(result.OpsPerSecond))
	fmt.Printf("  │ Avg Latency: %-43s │\n", formatLatency(result.AvgLatencyUs))
	fmt.Printf("  │ Duration:    %-43s │\n", formatDuration(result.Duration))
	fmt.Println("  └─────────────────────────────────────────────────────────┘")
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.2fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.2fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatOpsPerSec(ops float64) string {
	if ops >= 1000000 {
		return fmt.Sprintf("%.2fM ops/sec", ops/1000000)
	} else if ops >= 1000 {
		return fmt.Sprintf("%.2fK ops/sec", ops/1000)
	}
	return fmt.Sprintf("%.2f ops/sec", ops)
}

func formatLatency(us float64) string {
	if us >= 1000 {
		return fmt.Sprintf("%.2f ms", us/1000)
	}
	return fmt.Sprintf("%.0f μs", us)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2f sec", d.Seconds())
}
