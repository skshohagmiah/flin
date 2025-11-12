# Flin Client SDKs

Official client SDKs for Flin distributed key-value store with cluster-aware routing and client-side partitioning.

## Available SDKs

### 🐹 Go SDK
**Location**: `./go/`

High-performance Go client with cluster-aware routing, connection pooling, and full protocol support.

**Features**:
- ✅ Cluster-aware routing with client-side partitioning
- ✅ Connection pooling per node
- ✅ Automatic topology discovery via `/cluster` API
- ✅ Failover to replica nodes
- ✅ Parallel batch operations across nodes
- ✅ Uses `internal/net` for networking
- ✅ Uses `internal/protocol` for binary protocol

**Installation**:
```bash
go get github.com/skshohagmiah/flin/clients/go
```

**Quick Start**:
```go
import flin "github.com/skshohagmiah/flin/clients/go"

// Simple client
client, _ := flin.NewClient("localhost:7380")
client.Set("key", []byte("value"))

// Cluster-aware client
opts := flin.DefaultClusterOptions([]string{"localhost:7080", "localhost:7081"})
cluster, _ := flin.NewClusterClient(opts)
cluster.Set("key", []byte("value"))
```

[Full Go SDK Documentation →](./go/README.md)

---

### 🟢 Node.js SDK
**Location**: `./nodejs/`

TypeScript-first Node.js client with cluster-aware routing, async/await API, and connection pooling.

**Features**:
- ✅ Cluster-aware routing with client-side partitioning
- ✅ Connection pooling per node
- ✅ Automatic topology discovery via `/cluster` API
- ✅ Failover to replica nodes
- ✅ Parallel batch operations across nodes
- ✅ Full TypeScript support
- ✅ Promise-based async/await API

**Installation**:
```bash
npm install @flin/client
```

**Quick Start**:
```typescript
import { FlinClient, ClusterClient } from '@flin/client';

// Simple client
const client = new FlinClient({ host: 'localhost', port: 7380 });
await client.set('key', 'value');

// Cluster-aware client
const cluster = new ClusterClient({
  httpAddresses: ['localhost:7080', 'localhost:7081']
});
await cluster.initialize();
await cluster.set('key', 'value');
```

[Full Node.js SDK Documentation →](./nodejs/README.md)

---

## Architecture

### Cluster-Aware Routing

Both SDKs implement client-side partitioning for optimal performance:

```
┌─────────────┐
│   Client    │
│  (Go/Node)  │
└──────┬──────┘
       │
       │ 1. Hash key → Partition ID
       │ 2. Lookup partition owner
       │ 3. Send directly to node
       │
       ├──────────────┬──────────────┬──────────────┐
       ▼              ▼              ▼              ▼
   ┌───────┐      ┌───────┐      ┌───────┐      ┌───────┐
   │Node 1 │      │Node 2 │      │Node 3 │      │Node 4 │
   │P0-P15 │      │P16-P31│      │P32-P47│      │P48-P63│
   └───────┘      └───────┘      └───────┘      └───────┘
```

**Benefits**:
- ⚡ **No server-side routing** - Direct connection to partition owner
- 🚀 **Lower latency** - One network hop instead of two
- 📊 **Better throughput** - Reduced server load
- 🔄 **Automatic failover** - Falls back to replicas if primary fails

### Topology Discovery

Clients automatically discover and maintain cluster topology:

```
1. Client queries /cluster API endpoint
2. Receives node list and partition map
3. Creates connection pool for each node
4. Periodically refreshes topology (default: 30s)
5. Automatically adapts to cluster changes
```

### Connection Pooling

Each node gets its own connection pool:

```
Client
├── Node 1 Pool (min: 2, max: 10)
├── Node 2 Pool (min: 2, max: 10)
├── Node 3 Pool (min: 2, max: 10)
└── Node 4 Pool (min: 2, max: 10)
```

**Benefits**:
- 🔄 Connection reuse across requests
- ⚡ Pre-warmed connections ready to use
- 📊 Configurable per-node limits
- 🛡️ Automatic connection health checks

## Features Comparison

| Feature | Go SDK | Node.js SDK |
|---------|--------|-------------|
| Simple Client | ✅ | ✅ |
| Cluster Client | ✅ | ✅ |
| Client-Side Partitioning | ✅ | ✅ |
| Topology Discovery | ✅ | ✅ |
| Connection Pooling | ✅ | ✅ |
| Automatic Failover | ✅ | ✅ |
| Batch Operations | ✅ | ✅ |
| Atomic Counters | ✅ | ✅ |
| TypeScript Support | ❌ | ✅ |
| Zero Dependencies | ✅ | ✅ |

## Protocol Support

Both SDKs support the full Flin binary protocol:

### Basic Operations
- `SET` - Store key-value pair
- `GET` - Retrieve value by key
- `DEL` - Delete key
- `EXISTS` - Check if key exists

### Atomic Operations
- `INCR` - Increment counter
- `DECR` - Decrement counter

### Batch Operations
- `MSET` - Batch set multiple keys
- `MGET` - Batch get multiple keys
- `MDEL` - Batch delete multiple keys

## Performance

### Single Node (localhost)

| Operation | Go SDK | Node.js SDK |
|-----------|--------|-------------|
| SET | ~150K ops/sec | ~120K ops/sec |
| GET | ~280K ops/sec | ~200K ops/sec |
| MSET (100 keys) | ~50K ops/sec | ~40K ops/sec |
| MGET (100 keys) | ~80K ops/sec | ~60K ops/sec |

### Cluster (3 nodes, localhost)

| Operation | Go SDK | Node.js SDK |
|-----------|--------|-------------|
| SET | ~140K ops/sec | ~110K ops/sec |
| GET | ~260K ops/sec | ~180K ops/sec |
| MSET (100 keys) | ~80K ops/sec* | ~65K ops/sec* |
| MGET (100 keys) | ~120K ops/sec* | ~95K ops/sec* |

*Cluster clients benefit from parallel requests to multiple nodes

## Examples

See the `examples/` directory in each SDK for complete examples:

### Go Examples
- `go/examples/basic/` - Simple client usage
- `go/examples/cluster/` - Cluster-aware client usage

### Node.js Examples
- `nodejs/examples/basic.ts` - Simple client usage
- `nodejs/examples/cluster.ts` - Cluster-aware client usage

## Configuration

### Go SDK

```go
// Simple client
opts := &flin.ClientOptions{
    Address:      "localhost:7380",
    MinConns:     5,
    MaxConns:     50,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}

// Cluster client
clusterOpts := &flin.ClusterOptions{
    HTTPAddresses:         []string{"localhost:7080", "localhost:7081"},
    MinConnectionsPerNode: 2,
    MaxConnectionsPerNode: 10,
    RefreshInterval:       30 * time.Second,
    PartitionCount:        64,
}
```

### Node.js SDK

```typescript
// Simple client
const opts = {
    host: 'localhost',
    port: 7380,
    minConnections: 5,
    maxConnections: 50,
    connectionTimeout: 5000,
    requestTimeout: 10000,
};

// Cluster client
const clusterOpts = {
    httpAddresses: ['localhost:7080', 'localhost:7081'],
    minConnectionsPerNode: 2,
    maxConnectionsPerNode: 10,
    refreshInterval: 30000,
    partitionCount: 64,
};
```

## Internal Packages

Both SDKs use shared internal packages:

- **`internal/net`** - Connection management and pooling
- **`internal/protocol`** - Binary protocol encoding/decoding

This ensures consistency between SDKs and the server implementation.

## Contributing

Contributions are welcome! Please see the main repository for contribution guidelines.

## License

MIT License - see LICENSE file for details

## Support

- 📖 [Documentation](https://github.com/skshohagmiah/flin)
- 🐛 [Issue Tracker](https://github.com/skshohagmiah/flin/issues)
- 💬 [Discussions](https://github.com/skshohagmiah/flin/discussions)
