# Flin Go Client SDK

Official Go client SDK for Flin distributed key-value store with **smart, cluster-aware routing** by default.

## Features

- 🧠 **Smart Client**: Automatically adapts to single-node or cluster mode
- 🌐 **Cluster-Aware**: Client-side partitioning with automatic topology discovery
- 🚀 **High Performance**: Binary protocol with zero-copy operations
- 🔄 **Connection Pooling**: Per-node connection management
- 📦 **Batch Operations**: Parallel MSet/MGet/MDelete across nodes
- 🔢 **Atomic Counters**: Incr/Decr operations
- ⚡ **Zero External Dependencies**: Uses internal/net package
- 🛡️ **Thread-Safe**: Safe for concurrent use
- 🔍 **Topology Discovery**: Automatic cluster state updates via `/cluster` API
- 🔄 **Automatic Failover**: Falls back to replicas if primary fails

## Installation

```bash
go get github.com/skshohagmiah/flin/clients/go
```

## Quick Start

### Single-Node Mode

```go
package main

import (
    "fmt"
    "log"
    
    flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
    // Create client for single node
    opts := flin.DefaultOptions("localhost:7380")
    client, err := flin.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Client automatically uses smart routing
    err = client.Set("user:1", []byte("John Doe"))
    value, err := client.Get("user:1")
    fmt.Printf("Value: %s\n", value)
}
```

### Cluster Mode (Automatic Topology Discovery)

```go
package main

import (
    "fmt"
    "log"
    
    flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
    // Create cluster-aware client with topology discovery
    opts := flin.DefaultClusterOptions([]string{
        "localhost:7080",  // HTTP addresses for /cluster API
        "localhost:7081",
        "localhost:7082",
    })
    
    client, err := flin.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Check mode
    fmt.Printf("Cluster mode: %v\n", client.IsClusterMode())
    
    // Operations are automatically routed to the correct node
    // No server-side routing overhead!
    err = client.Set("user:1", []byte("John Doe"))
    value, err := client.Get("user:1")
    
    // Batch operations are distributed across nodes in parallel
    keys := []string{"key1", "key2", "key3"}
    values := [][]byte{[]byte("val1"), []byte("val2"), []byte("val3")}
    err = client.MSet(keys, values)
}
```

## API Reference

### Basic Operations

#### Set
```go
err := client.Set("key", []byte("value"))
```

#### Get
```go
value, err := client.Get("key")
```

#### Delete
```go
err := client.Delete("key")
```

#### Exists
```go
exists, err := client.Exists("key")
```

### Atomic Counters

#### Increment
```go
newValue, err := client.Incr("counter")
```

#### Decrement
```go
newValue, err := client.Decr("counter")
```

### Batch Operations

#### MSet - Batch Set
```go
keys := []string{"key1", "key2", "key3"}
values := [][]byte{
    []byte("value1"),
    []byte("value2"),
    []byte("value3"),
}
err := client.MSet(keys, values)
```

#### MGet - Batch Get
```go
keys := []string{"key1", "key2", "key3"}
values, err := client.MGet(keys)
```

#### MDelete - Batch Delete
```go
keys := []string{"key1", "key2", "key3"}
err := client.MDelete(keys)
```

## Configuration

### Simple Client Options
```go
opts := &flin.ClientOptions{
    Address:      "localhost:7380",
    MinConns:     5,
    MaxConns:     50,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}
client, err := flin.NewClientWithOptions(opts)
```

### Cluster Client Options
```go
opts := &flin.ClusterOptions{
    HTTPAddresses:         []string{"localhost:7080", "localhost:7081"},
    MinConnectionsPerNode: 2,
    MaxConnectionsPerNode: 10,
    RefreshInterval:       30 * time.Second,
    PartitionCount:        64,
}
client, err := flin.NewClusterClient(opts)
```

## Cluster-Aware Features

### Automatic Topology Discovery
The cluster client automatically discovers the cluster topology from the HTTP API:

```go
// Client queries /cluster endpoint
topology := client.GetTopology()
fmt.Printf("Nodes: %d\n", len(topology.Nodes))
fmt.Printf("Partitions: %d\n", len(topology.PartitionMap))
```

### Client-Side Partitioning
Keys are automatically routed to the correct node:

```go
// Key is hashed to determine partition
// Request is sent directly to the partition owner
// No server-side routing overhead!
client.Set("user:123", data)
```

### Automatic Failover
If the primary node fails, requests automatically failover to replicas:

```go
// Tries primary node first
// Falls back to replica nodes if primary is unavailable
value, err := client.Get("key")
```

### Batch Operations Across Nodes
Batch operations are intelligently distributed:

```go
// Keys are grouped by node
// Parallel requests to each node
// Results are merged and returned in order
values, err := client.MGet([]string{"key1", "key2", "key3"})
```

## Performance

### Benchmarks (localhost)

| Operation | Simple Client | Cluster Client |
|-----------|--------------|----------------|
| SET | ~150K ops/sec | ~140K ops/sec |
| GET | ~280K ops/sec | ~260K ops/sec |
| MSET (100 keys) | ~50K ops/sec | ~80K ops/sec* |
| MGET (100 keys) | ~80K ops/sec | ~120K ops/sec* |

*Cluster client benefits from parallel requests to multiple nodes

## Examples

See the `examples/` directory for complete examples:
- `basic/` - Simple client usage
- `cluster/` - Cluster-aware client usage
- `batch/` - Batch operations
- `counters/` - Atomic counters

## License

MIT License
