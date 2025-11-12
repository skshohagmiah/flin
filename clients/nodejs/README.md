# Flin Node.js Client SDK

Official Node.js/TypeScript client SDK for Flin distributed key-value store with cluster-aware routing.

## Features

- 🚀 **High Performance**: Binary protocol with connection pooling
- 🌐 **Cluster-Aware**: Client-side partitioning and smart routing
- 🔄 **Connection Pooling**: Automatic connection management per node
- 📦 **Batch Operations**: MSet, MGet, MDelete for bulk operations
- 🔢 **Atomic Counters**: Incr/Decr operations with BigInt support
- 📘 **TypeScript**: Full TypeScript support with type definitions
- ⚡ **Zero Dependencies**: Pure Node.js implementation
- 🔒 **Promise-based**: Modern async/await API
- 🔍 **Topology Discovery**: Automatic cluster state updates

## Installation

```bash
npm install @flin/client
```

## Quick Start

### Simple Client (Single Node)

```typescript
import { FlinClient } from '@flin/client';

async function main() {
  const client = new FlinClient({
    host: 'localhost',
    port: 7380,
  });

  // Set a value
  await client.set('user:1', 'John Doe');

  // Get a value
  const value = await client.getString('user:1');
  console.log('Value:', value);

  // Close client
  await client.close();
}

main().catch(console.error);
```

### Cluster-Aware Client

```typescript
import { ClusterClient } from '@flin/client';

async function main() {
  // Create cluster-aware client
  const client = new ClusterClient({
    httpAddresses: [
      'localhost:7080',  // HTTP addresses for topology discovery
      'localhost:7081',
      'localhost:7082',
    ],
  });

  // Initialize and discover topology
  await client.initialize();

  // Operations are automatically routed to the correct node
  await client.set('user:1', 'John Doe');
  const value = await client.getString('user:1');
  console.log('Value:', value);

  // Batch operations are distributed across nodes
  await client.mset([
    ['key1', 'value1'],
    ['key2', 'value2'],
    ['key3', 'value3'],
  ]);

  const values = await client.mget(['key1', 'key2', 'key3']);
  console.log('Values:', values);

  await client.close();
}

main().catch(console.error);
```

## API Reference

### Simple Client

#### Constructor
```typescript
const client = new FlinClient({
  host: 'localhost',           // Server host
  port: 7380,                  // Server port
  minConnections: 5,           // Minimum pool size
  maxConnections: 50,          // Maximum pool size
  connectionTimeout: 5000,     // Connection timeout (ms)
  requestTimeout: 10000,       // Request timeout (ms)
});
```

#### Basic Operations

```typescript
// Set
await client.set('key', 'value');
await client.set('key', Buffer.from('value'));

// Get
const value = await client.get('key');        // Returns Buffer
const str = await client.getString('key');    // Returns string

// Delete
await client.delete('key');

// Exists
const exists = await client.exists('key');
```

#### Atomic Counters

```typescript
// Increment
const newValue = await client.incr('counter');  // Returns BigInt

// Decrement
const newValue = await client.decr('counter');  // Returns BigInt
```

#### Batch Operations

```typescript
// MSet
await client.mset([
  ['key1', 'value1'],
  ['key2', Buffer.from('value2')],
]);

// MGet
const values = await client.mget(['key1', 'key2']);
// Returns: Array<Buffer | null>

// MDelete
await client.mdelete(['key1', 'key2']);
```

### Cluster Client

#### Constructor
```typescript
const client = new ClusterClient({
  httpAddresses: ['localhost:7080', 'localhost:7081'],
  minConnectionsPerNode: 2,
  maxConnectionsPerNode: 10,
  connectionTimeout: 5000,
  requestTimeout: 10000,
  refreshInterval: 30000,      // Topology refresh interval (ms)
  partitionCount: 64,           // Number of partitions
});

await client.initialize();
```

#### Cluster Features

```typescript
// Get topology information
const topology = client.getTopology();
console.log('Nodes:', topology.nodes.length);
console.log('Partitions:', topology.partitionMap.size);
console.log('Last update:', topology.lastUpdate);

// All operations same as simple client
await client.set('key', 'value');
const value = await client.get('key');
```

## Examples

### Working with JSON

```typescript
interface User {
  id: number;
  name: string;
  email: string;
}

// Save JSON
const user: User = { id: 1, name: 'John', email: 'john@example.com' };
await client.set('user:1', JSON.stringify(user));

// Load JSON
const data = await client.getString('user:1');
const loadedUser: User = JSON.parse(data!);
```

### Counter Example

```typescript
// Initialize counter
await client.set('page:views', Buffer.alloc(8));

// Increment page views
const views = await client.incr('page:views');
console.log(`Page views: ${views}`);
```

### Batch Operations

```typescript
// Prepare data
const entries: Array<[string, string]> = [
  ['user:1', 'Alice'],
  ['user:2', 'Bob'],
  ['user:3', 'Charlie'],
];

// Batch set
await client.mset(entries);

// Batch get
const keys = ['user:1', 'user:2', 'user:3'];
const values = await client.mget(keys);

for (let i = 0; i < keys.length; i++) {
  if (values[i]) {
    console.log(`${keys[i]}: ${values[i]!.toString()}`);
  }
}
```

### Using with Express

```typescript
import express from 'express';
import { FlinClient } from '@flin/client';

const app = express();
const flin = new FlinClient({ host: 'localhost', port: 7380 });

app.get('/user/:id', async (req, res) => {
  try {
    const user = await flin.getString(`user:${req.params.id}`);
    if (user) {
      res.json(JSON.parse(user));
    } else {
      res.status(404).json({ error: 'User not found' });
    }
  } catch (err: any) {
    res.status(500).json({ error: err.message });
  }
});

app.post('/user/:id', async (req, res) => {
  try {
    await flin.set(`user:${req.params.id}`, JSON.stringify(req.body));
    res.json({ success: true });
  } catch (err: any) {
    res.status(500).json({ error: err.message });
  }
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  await flin.close();
  process.exit(0);
});

app.listen(3000);
```

### Cluster-Aware Example

```typescript
import { ClusterClient } from '@flin/client';

async function clusterExample() {
  const client = new ClusterClient({
    httpAddresses: ['localhost:7080', 'localhost:7081', 'localhost:7082'],
    minConnectionsPerNode: 2,
    maxConnectionsPerNode: 10,
    refreshInterval: 30000,
  });

  await client.initialize();

  // Keys are automatically routed to correct nodes
  await client.set('user:123', 'John');
  await client.set('product:456', 'Widget');
  await client.set('order:789', 'Pending');

  // Batch operations are distributed across nodes
  const keys = ['user:123', 'product:456', 'order:789'];
  const values = await client.mget(keys);
  
  console.log('Retrieved from multiple nodes:', values);

  // Get topology info
  const topology = client.getTopology();
  console.log(`Cluster has ${topology.nodes.length} nodes`);
  console.log(`Managing ${topology.partitionMap.size} partitions`);

  await client.close();
}
```

## Cluster-Aware Features

### Automatic Topology Discovery
The cluster client automatically discovers and maintains cluster topology:

```typescript
const client = new ClusterClient({
  httpAddresses: ['localhost:7080', 'localhost:7081'],
  refreshInterval: 30000, // Refresh every 30 seconds
});

await client.initialize();

// Topology is automatically kept up-to-date
```

### Client-Side Partitioning
Keys are hashed and routed directly to the owning node:

```typescript
// Key is hashed using FNV-1a
// Request goes directly to the partition owner
// No server-side routing overhead!
await client.set('user:123', data);
```

### Automatic Failover
If primary node fails, requests automatically failover to replicas:

```typescript
// Tries primary node first
// Falls back to replica nodes if primary is unavailable
const value = await client.get('key');
```

### Parallel Batch Operations
Batch operations are intelligently distributed:

```typescript
// Keys are grouped by node
// Parallel requests to each node
// Results are merged and returned in order
const values = await client.mget(['key1', 'key2', 'key3']);
```

## Error Handling

```typescript
try {
  const value = await client.get('key');
  if (value === null) {
    console.log('Key not found');
  }
} catch (err) {
  console.error('Error:', err.message);
}
```

## Performance Tips

1. **Use Batch Operations**: For multiple keys, use `mset`/`mget` instead of multiple `set`/`get` calls
2. **Reuse Client**: Create one client instance and reuse it across your application
3. **Tune Pool Size**: Adjust connection pool settings based on your workload
4. **Use Cluster Client**: For distributed deployments, use `ClusterClient` for better performance
5. **Buffer vs String**: Use `Buffer` directly for binary data to avoid encoding overhead

## Building from Source

```bash
# Install dependencies
npm install

# Build TypeScript
npm run build

# The compiled JavaScript will be in the dist/ directory
```

## License

MIT License
