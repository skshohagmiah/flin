# 🚀 Flin - High-Performance distributed data management system

A blazing-fast, distributed key-value store | message queue | stream pub/sub data management system.

## ⚡ Performance Highlights

- **792K ops/sec** - Peak throughput (in-memory + batching)
- **682K ops/sec** - Disk-based with batching (6.8x faster than Redis)
- **1.26μs latency** - Sub-microsecond operations  
- **7.9x faster than Redis** - Best comparison

## 🎯 Key Features

### KV Store
- ✅ **319K ops/sec** read throughput
- ✅ **151K ops/sec** write throughput
- ✅ **Sub-10μs latency**
- ✅ **Atomic batch operations** (MSET/MGET/MDEL)
- ✅ **Dual storage**: Disk (durable) or Memory (fastest)
- ✅ **Text + Binary protocols** with auto-detection
- ✅ **3-node clustering** with replication

### Message Queue
- ✅ **104K ops/sec** push throughput
- ✅ **100K ops/sec** pop throughput
- ✅ **Unified Port**: Runs on same port as KV (7380)
- ✅ **Durable**: Backed by BadgerDB
- ✅ **Atomic**: Crash-safe metadata management

## 📦 Quick Start

### 🐳 Docker (Recommended)

**Single Node:**
```bash
cd docker/single && ./run.sh
```

**3-Node Cluster:**
```bash
cd docker/cluster && ./run.sh
```

Both scripts automatically:
- Start the node(s)
- Run performance benchmarks
- Show throughput metrics
- Leave cluster running for testing

See [docker/README.md](docker/README.md) for details.

### 💻 Local Installation

```bash
git clone https://github.com/skshohagmiah/flin
cd flin
go build -o bin/flin-server ./cmd/server
```

### Run Server

```bash
# Unified Server (KV + Queue)
./bin/flin-server -node-id=node1 -port=:7380 -workers=256
```

## 💻 Usage

```go
import "github.com/skshohagmiah/flin/clients/go"

// Create unified client (connects to port 7380)
opts := flin.DefaultOptions("localhost:7380")
client, _ := flin.NewClient(opts)
defer client.Close()

// KV Store operations
client.Set("user:1", []byte("John Doe"))
value, _ := client.Get("user:1")
client.Delete("user:1")

// Queue operations (Unified!)
client.Queue.Push("tasks", []byte("Task 1"))
client.Queue.Push("tasks", []byte("Task 2"))

// Dequeue message
msg, _ := client.Queue.Pop("tasks")
fmt.Printf("Received: %s\n", string(msg))
```

## Performance vs Redis

| System | KV Read | KV Write | Queue Push | Queue Pop |
|--------|---------|----------|------------|-----------|
| **Flin** | **319K** | **151K** | **104K** | **100K** |
| Redis | ~100K | ~80K | ~80K | ~80K |

**Flin is up to 3x faster than Redis!** 🚀

## 📚 Documentation

- [End-to-End Data Flow](flow.md) - Architecture explained
- [Performance Summary](FINAL_PERFORMANCE_SUMMARY.md)
- [Docker Deployment](DOCKER.md)
- [Performance Tests](test_scripts/) - Run benchmarks
- [Benchmarks](benchmarks/) - Cluster tests

## 📄 License

MIT License
