package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/skshohagmiah/flin/internal/kv"
)

const (
	version = "1.0.0"
	banner  = `
╔═══════════════════════════════════════╗
║   Flin - High-Performance KV Store   ║
║   Version: %s                    ║
╚═══════════════════════════════════════╝
`
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		fmt.Printf(banner, version)

	case "benchmark", "bench":
		runBenchmark()

	case "set":
		if len(os.Args) < 4 {
			fmt.Println("Usage: flin set <key> <value> [ttl_seconds]")
			os.Exit(1)
		}
		runSet(os.Args[2], os.Args[3], os.Args[4:])

	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: flin get <key>")
			os.Exit(1)
		}
		runGet(os.Args[2])

	case "delete", "del":
		if len(os.Args) < 3 {
			fmt.Println("Usage: flin delete <key>")
			os.Exit(1)
		}
		runDelete(os.Args[2])

	case "exists":
		if len(os.Args) < 3 {
			fmt.Println("Usage: flin exists <key>")
			os.Exit(1)
		}
		runExists(os.Args[2])

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(banner, version)
	fmt.Println(`Usage: flin <command> [arguments]

Commands:
  set <key> <value> [ttl]   Set a key-value pair (optional TTL in seconds)
  get <key>                 Get value by key
  delete <key>              Delete a key
  exists <key>              Check if key exists
  benchmark                 Run performance benchmark
  version                   Show version information
  help                      Show this help message

Examples:
  flin set mykey "hello world"
  flin set session:123 "data" 3600
  flin get mykey
  flin delete mykey
  flin exists mykey
  flin benchmark

Environment Variables:
  FLIN_DATA_DIR            Data directory (default: ./data)
  FLIN_BENCHMARK_DURATION  Benchmark duration (default: 10s)`)
}

func getStore() (*kv.KVStore, error) {
	dataDir := os.Getenv("FLIN_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return kv.New(dataDir)
}

func runSet(key, value string, args []string) {
	store, err := getStore()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	var ttl time.Duration
	if len(args) > 0 {
		seconds, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Error: invalid TTL value: %v\n", err)
			os.Exit(1)
		}
		ttl = time.Duration(seconds) * time.Second
	}

	if err := store.Set(key, []byte(value), ttl); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if ttl > 0 {
		fmt.Printf("✅ Set '%s' with TTL %v\n", key, ttl)
	} else {
		fmt.Printf("✅ Set '%s'\n", key)
	}
}

func runGet(key string) {
	store, err := getStore()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	value, err := store.Get(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", value)
}

func runDelete(key string) {
	store, err := getStore()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Delete(key); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Deleted '%s'\n", key)
}

func runExists(key string) {
	store, err := getStore()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	exists, err := store.Exists(key)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if exists {
		fmt.Printf("✅ Key '%s' exists\n", key)
	} else {
		fmt.Printf("❌ Key '%s' does not exist\n", key)
	}
}

func runBenchmark() {
	fmt.Printf(banner, version)
	fmt.Println("🚀 Running benchmark...")
	fmt.Println()

	// Create temporary directory for benchmark
	tmpDir, err := os.MkdirTemp("", "flin-bench-*")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	store, err := kv.New(tmpDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	duration := 10 * time.Second
	if envDuration := os.Getenv("FLIN_BENCHMARK_DURATION"); envDuration != "" {
		if d, err := time.ParseDuration(envDuration); err == nil {
			duration = d
		}
	}

	value := make([]byte, 1024) // 1KB
	for i := range value {
		value[i] = byte(i % 256)
	}

	// Benchmark SET
	fmt.Printf("📝 Benchmarking SET operations (%v)...\n", duration)
	setOps := benchmarkOperation(store, duration, func(i int) error {
		key := fmt.Sprintf("bench_key_%d", i)
		return store.Set(key, value, 0)
	})
	fmt.Printf("   Throughput: %.2fK ops/sec\n\n", float64(setOps)/duration.Seconds()/1000)

	// Pre-populate for GET benchmark
	fmt.Printf("📖 Benchmarking GET operations (%v)...\n", duration)
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("get_key_%d", i)
		store.Set(key, value, 0)
	}

	getOps := benchmarkOperation(store, duration, func(i int) error {
		key := fmt.Sprintf("get_key_%d", i%numKeys)
		_, err := store.Get(key)
		return err
	})
	fmt.Printf("   Throughput: %.2fK ops/sec\n\n", float64(getOps)/duration.Seconds()/1000)
	
	fmt.Println("✅ Benchmark completed!")
}

func benchmarkOperation(store *kv.KVStore, duration time.Duration, op func(int) error) int {
	start := time.Now()
	stop := start.Add(duration)
	ops := 0

	for time.Now().Before(stop) {
		if err := op(ops); err != nil {
			continue
		}
		ops++
	}

	return ops
}
