## 🚀 Flin KV Server - NATS-Style Architecture

A high-performance key-value server implementing NATS messaging patterns for extreme throughput.

### Architecture

```
Client 1 ──→ [Connection: readLoop + writeLoop] ──┐
                                                   ├──→ KV Store (BadgerDB)
Client 2 ──→ [Connection: readLoop + writeLoop] ──┤
                                                   ├──→ Inline processing
Client 3 ──→ [Connection: readLoop + writeLoop] ──┘    No goroutine-per-request!
```

### Key Design Principles

1. **Per-Connection Goroutines**
   - 2 goroutines per connection (readLoop + writeLoop)
   - Not 1 goroutine per request!

2. **Inline Processing**
   - Parse and execute in readLoop (no spawning)
   - Fast path optimization

3. **Buffered Channels**
   - Non-blocking dispatch to writeLoop
   - 1000-message buffer per connection

4. **Lock-Free Where Possible**
   - Atomic counters for metrics
   - sync.Map for connection tracking

### Running the Server

```bash
# Terminal 1: Start server
./scripts/run_server.sh

# Terminal 2: Run benchmark
./scripts/run_server_benchmark.sh
```

Or manually:

```bash
# Start server
go run ./cmd/kvserver/main.go

# In another terminal, benchmark it
go run ./scripts/benchmark_server.go
```

### Protocol

Simple Redis-style text protocol:

```
SET key value\r\n
GET key\r\n
DEL key\r\n
```

Responses:
```
+OK\r\n                    # Success
$5\r\nvalue\r\n           # Bulk string
-ERR message\r\n          # Error
```

### Expected Performance

With NATS-style architecture:

**Single Connection:**
- SET: 80-100K ops/sec
- GET: 500K+ ops/sec

**Multiple Connections (8-16):**
- SET: 200-400K ops/sec
- GET: 1M+ ops/sec

### Why This is Fast

1. **No goroutine spawning** - Process requests inline
2. **Buffered I/O** - 32KB buffers for read/write
3. **Async writes** - writeLoop handles all responses
4. **Optimized BadgerDB** - Large caches, async writes
5. **Minimal allocations** - Reuse buffers per connection

### Comparison with Other Approaches

| Approach | Goroutines | Throughput | Latency |
|----------|------------|------------|---------|
| **NATS-style (this)** | 2 per conn | 🔥 High | ⚡ Low |
| Goroutine-per-request | 1000s | ❌ Low | 🐌 High |
| Single event loop | 1 | ⚠️ Medium | ⚡ Low |
| Worker pool | 4-16 | ✅ Good | ✅ Good |

### Tuning

Adjust in `kv_server.go`:

```go
outQueue: make(chan []byte, 1000)  // Response buffer size
readBuf:  make([]byte, 32768)      // Read buffer (32KB)
writeBuf: make([]byte, 32768)      // Write buffer (32KB)
```

For high-throughput scenarios:
- Increase buffer sizes
- Increase outQueue capacity
- Tune BadgerDB cache sizes

### Monitoring

Server exposes stats:

```go
stats := server.Stats()
// {
//   "active_connections": 100,
//   "ops_processed": 1000000
// }
```

---

**This is how NATS achieves 10M+ msgs/sec!** 🚀
