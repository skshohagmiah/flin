# 🚀 Flin - High-Performance KV Store

Flin is a fast, distributed data engine built in Go, designed to handle key-value storage, message queues, and streaming workloads — all under one unified system.

**Current Status:** ✅ KV Store (Production Ready) | 🚧 Queue & Stream (Coming Soon)

## 🎯 Performance

**Optimized with NATS-style architecture:**
- ✅ **103K SET ops/sec** (4 workers)
- ✅ **787K GET ops/sec** (4 workers)
- ✅ **140K mixed ops/sec** (70% reads, 30% writes)
- ✅ **Sub-40μs latency** for writes
- ✅ **Sub-5μs latency** for reads

**Matches Redis performance for embedded use!** 🚀

See [BENCHMARKS.md](./BENCHMARKS.md) for detailed performance analysis.

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
# Build and start
make docker-build
make docker-run

# Use CLI
make docker-cli ARGS='set mykey "hello world"'
make docker-cli ARGS='get mykey'

# Run benchmark
make docker-benchmark
```

See [DOCKER.md](./DOCKER.md) for complete Docker guide.

### Using CLI

```bash
# Build
make build-cli

# Basic operations
./flin set mykey "hello world"
./flin set session:123 "data" 3600  # with TTL
./flin get mykey
./flin delete mykey
./flin exists mykey

# Run benchmark
./flin benchmark
```

### As an Embedded Library

```go
import "github.com/skshohagmiah/flin/internal/kv"

store, _ := kv.New("./data")
defer store.Close()

// Set with TTL
store.Set("session:123", []byte("data"), 3600*time.Second)

// Get
value, _ := store.Get("session:123")

// Batch operations (5-10x faster!)
batch := map[string][]byte{
    "key1": []byte("value1"),
    "key2": []byte("value2"),
}
store.BatchSet(batch)
```

### Using the Go SDK Client

```go
import "github.com/skshohagmiah/flin/pkg/client"

// Connect to Flin server
c, _ := client.New("localhost:6380")
defer c.Close()

// Simple operations
c.Set("mykey", []byte("hello"))
value, _ := c.Get("mykey")

// With connection pooling (recommended)
pc, _ := client.NewPooledClient(client.DefaultPoolConfig())
defer pc.Close()

pc.Set("key", []byte("value"))
```

See [pkg/client/README.md](./pkg/client/README.md) for full SDK documentation.

## 📦 Installation

### Docker

```bash
docker pull flin-kv:latest
docker run -d -p 6380:6380 -v flin-data:/data flin-kv:latest ./kvserver
```

### From Source

```bash
git clone https://github.com/skshohagmiah/flin.git
cd flin
make build
```

## 🧠 Architecture

Flin uses **NATS-style architecture** for extreme performance:

- **Per-connection goroutines** (not per-request!)
- **Inline processing** (no spawning overhead)
- **Buffered channels** for async I/O
- **Lock-free operations** where possible
- **Optimized BadgerDB** (512MB caches, async writes)
- **Optimal concurrency** (4 workers, not 100!)

## ⚡ Core Features

- 🚀 **High Performance** - NATS-level throughput (100K+ ops/sec)
- 💾 **Persistent Storage** - BadgerDB LSM tree with caching
- 🔄 **Buffer Pooling** - `sync.Pool` for zero allocations
- 📦 **Batch Operations** - 5-10x throughput improvement
- 🎯 **Stack Allocations** - No heap escape for small buffers
- ⏱️ **TTL Support** - Automatic expiration
- 🐳 **Docker Ready** - Production-ready containers
- 🛠️ **CLI Tool** - Easy command-line interface

## 📊 Benchmark Results

| Operation | Throughput | Latency | Notes |
|-----------|------------|---------|-------|
| **SET** | 103K ops/sec | 38 μs | 4 workers optimal |
| **GET** | 787K ops/sec | 4 μs | Cache-optimized |
| **MIXED** | 140K ops/sec | 28 μs | 70% reads, 30% writes |
| **DELETE** | 165K ops/sec | 23 μs | Fast tombstones |

**Batch Operations:**
- BatchSet: 500K-1M+ ops/sec
- BatchGet: 1M-2M+ ops/sec

## 📚 Documentation

- [BENCHMARKS.md](./BENCHMARKS.md) - Detailed performance analysis
- [DOCKER.md](./DOCKER.md) - Docker deployment guide
- [performance.md](./performance.md) - Performance tuning guide
- [cmd/kvserver/README.md](./cmd/kvserver/README.md) - Server architecture

## 🛠️ Development

```bash
# Build everything
make build

# Run tests
make test

# Run benchmark
make benchmark

# Docker build
make docker-build

# Clean
make clean
```

## 🎯 Use Cases

### Session Store
```go
// 90% reads, 10% writes
// Throughput: ~600K ops/sec
// Latency: <5μs p99
store.Set("session:"+id, sessionData, 24*time.Hour)
```

### Cache Layer
```go
// 95% reads, 5% writes
// Throughput: ~700K ops/sec
// Latency: <4μs p99
store.Set("cache:"+key, data, 1*time.Hour)
```

### Event Store
```go
// 20% reads, 80% writes
// Throughput: ~90K ops/sec
// Latency: <50μs p99
store.Set("event:"+id, eventData, 0)
```

## 🔮 Roadmap

- ✅ **KV Store** - Production ready
- 🚧 **Queue** - Coming soon
- 🚧 **Stream** - Coming soon
- 🚧 **Clustering** - Planned
- 🚧 **Replication** - Planned

## 📄 License

MIT License - see LICENSE file for details

## 🤝 Contributing

Contributions welcome! Please open an issue or PR.

---

**Built with ❤️ using Go, BadgerDB, and NATS-style architecture**