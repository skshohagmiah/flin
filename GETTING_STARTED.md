# 🚀 Getting Started with Flin KV Store

## What is Flin?

Flin is a **high-performance, embedded key-value store** built in Go that achieves **Redis-level performance** without network overhead.

### Key Features

- ✅ **103K SET ops/sec** (4 workers)
- ✅ **787K GET ops/sec** (4 workers)
- ✅ **Sub-40μs latency** for writes
- ✅ **Persistent storage** with BadgerDB
- ✅ **Docker ready** with 41MB image
- ✅ **CLI tool** for easy management

## Quick Start (3 Options)

### Option 1: Docker (Easiest)

```bash
# Test the CLI
docker run --rm flin-kv:latest ./flin version

# Set a key
docker run --rm -v flin-data:/data flin-kv:latest ./flin set mykey "hello"

# Get a key
docker run --rm -v flin-data:/data flin-kv:latest ./flin get mykey

# Run benchmark
docker run --rm flin-kv:latest ./flin benchmark
```

### Option 2: Using Make

```bash
# Build everything
make build

# Use CLI
./flin set mykey "hello world"
./flin get mykey
./flin benchmark

# Docker
make docker-build
make docker-run
make docker-cli ARGS='set key value'
```

### Option 3: Go Library

```go
package main

import (
    "fmt"
    "time"
    "github.com/skshohagmiah/flin/internal/kv"
)

func main() {
    // Create store
    store, _ := kv.New("./data")
    defer store.Close()
    
    // Set a key
    store.Set("user:1", []byte("John Doe"), 0)
    
    // Set with TTL (expires in 1 hour)
    store.Set("session:abc", []byte("data"), 1*time.Hour)
    
    // Get a key
    value, _ := store.Get("user:1")
    fmt.Println(string(value)) // "John Doe"
    
    // Check existence
    exists, _ := store.Exists("user:1")
    fmt.Println(exists) // true
    
    // Delete
    store.Delete("user:1")
    
    // Batch operations (5-10x faster!)
    batch := map[string][]byte{
        "key1": []byte("value1"),
        "key2": []byte("value2"),
        "key3": []byte("value3"),
    }
    store.BatchSet(batch)
    
    results, _ := store.BatchGet([]string{"key1", "key2", "key3"})
    fmt.Println(results)
}
```

## Installation

### Prerequisites

- Go 1.24+ (for building from source)
- Docker (for containerized deployment)
- Make (optional, for convenience)

### From Source

```bash
# Clone repository
git clone https://github.com/skshohagmiah/flin.git
cd flin

# Build CLI
make build-cli

# Build server
make build-server

# Or build both
make build
```

### Docker

```bash
# Pull image (when published)
docker pull flin-kv:latest

# Or build locally
make docker-build
```

## CLI Usage

### Basic Commands

```bash
# Set a key-value pair
./flin set mykey "hello world"

# Set with TTL (expires in 3600 seconds)
./flin set session:123 "user data" 3600

# Get a value
./flin get mykey

# Delete a key
./flin delete mykey

# Check if key exists
./flin exists mykey

# Run benchmark
./flin benchmark

# Show version
./flin version

# Show help
./flin help
```

### Environment Variables

```bash
# Change data directory
export FLIN_DATA_DIR=/path/to/data
./flin set mykey "value"

# Change benchmark duration
export FLIN_BENCHMARK_DURATION=30s
./flin benchmark
```

## Docker Usage

### Run CLI Commands

```bash
# Using docker run
docker run --rm -v flin-data:/data flin-kv:latest ./flin set mykey "value"
docker run --rm -v flin-data:/data flin-kv:latest ./flin get mykey

# Using make (easier)
make docker-cli ARGS='set mykey "value"'
make docker-cli ARGS='get mykey'
```

### Run Server

```bash
# Using docker-compose
make docker-run

# Or manually
docker run -d \
  --name flin-server \
  -p 6380:6380 \
  -v flin-data:/data \
  flin-kv:latest ./kvserver
```

### Run Benchmark

```bash
# Quick benchmark
make docker-benchmark

# Custom duration
docker run --rm \
  -e FLIN_BENCHMARK_DURATION=60s \
  flin-kv:latest ./flin benchmark
```

## Performance Tips

### 1. Use Batch Operations

```go
// Slow: 103K ops/sec
for i := 0; i < 1000; i++ {
    store.Set(key, value, 0)
}

// Fast: 500K+ ops/sec
batch := make(map[string][]byte)
for i := 0; i < 1000; i++ {
    batch[key] = value
}
store.BatchSet(batch)
```

### 2. Optimal Worker Count

```go
// Use 4 workers for best performance
// More workers cause lock contention!
```

### 3. Use TTL for Temporary Data

```go
// Automatic cleanup
store.Set("cache:"+key, data, 1*time.Hour)
```

### 4. Pre-allocate Maps

```go
// Faster
results := make(map[string][]byte, len(keys))
store.BatchGet(keys)
```

## Common Use Cases

### Session Store

```go
// Store user sessions
sessionID := "sess_" + uuid.New().String()
store.Set(sessionID, sessionData, 24*time.Hour)

// Retrieve session
data, err := store.Get(sessionID)
if err != nil {
    // Session expired or not found
}
```

### Cache Layer

```go
// Cache expensive computations
cacheKey := "cache:" + computeKey(params)
if value, err := store.Get(cacheKey); err == nil {
    return value // Cache hit
}

// Cache miss - compute and store
result := expensiveComputation(params)
store.Set(cacheKey, result, 1*time.Hour)
```

### Event Store

```go
// Store events with timestamps
eventID := fmt.Sprintf("event:%d:%s", time.Now().Unix(), uuid.New())
store.Set(eventID, eventData, 0) // No expiration
```

### Rate Limiting

```go
// Simple rate limiter
key := "rate:" + userID
count, err := store.Incr(key)
if err != nil {
    store.Set(key, []byte("1"), 60*time.Second)
    count = 1
}

if count > 100 {
    return errors.New("rate limit exceeded")
}
```

## Benchmarking

### Run Benchmarks

```bash
# CLI benchmark
./flin benchmark

# Go benchmark
cd scripts
./run_throughput_test.sh

# Docker benchmark
make docker-benchmark
```

### Expected Results

```
📝 SET operations: 103K ops/sec (38μs latency)
📖 GET operations: 787K ops/sec (4μs latency)
🔀 MIXED operations: 140K ops/sec (28μs latency)
```

### Interpreting Results

- **Throughput**: Operations per second
- **Latency**: Average time per operation
- **4 workers optimal**: More workers degrade performance
- **Docker overhead**: ~20% slower than native

## Troubleshooting

### CLI Issues

```bash
# Permission denied
chmod +x ./flin

# Data directory error
mkdir -p ./data
export FLIN_DATA_DIR=./data

# Build errors
go mod tidy
make build
```

### Docker Issues

```bash
# Container won't start
docker logs flin-server

# Port already in use
docker ps
lsof -i :6380

# Volume issues
docker volume rm flin_flin-data
make docker-run
```

### Performance Issues

```bash
# Check if using SSD
df -h ./data

# Increase cache size (edit internal/storage/kv.go)
opts.BlockCacheSize = 1024 << 20  // 1GB

# Use batch operations
# See "Performance Tips" above
```

## Next Steps

1. **Read the docs**
   - [BENCHMARKS.md](./BENCHMARKS.md) - Performance analysis
   - [DOCKER.md](./DOCKER.md) - Docker guide
   - [performance.md](./performance.md) - Tuning guide

2. **Try examples**
   - Run `./scripts/test_cli.sh`
   - Check `./bencemark/kv_test.go`

3. **Integrate into your app**
   - Use as library (see "Go Library" above)
   - Or run server and connect via TCP

4. **Optimize for your workload**
   - Tune cache sizes
   - Use batch operations
   - Adjust worker count

## Support

- **Issues**: Open a GitHub issue
- **Questions**: Check documentation
- **Contributions**: PRs welcome!

---

**You're ready to use Flin!** 🚀

Start with: `./flin set mykey "hello world"`
