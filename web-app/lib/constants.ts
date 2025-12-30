// Performance metrics
export const PERFORMANCE_METRICS = {
    kv: {
        read: { throughput: '319K', latency: '3.1μs', speedup: '3.2x' },
        write: { throughput: '151K', latency: '6.6μs', speedup: '1.9x' },
        batch: { throughput: '792K', latency: '1.26μs', speedup: '7.9x' },
    },
    queue: {
        push: { throughput: '104K', latency: '9.6μs', speedup: '1.3x' },
        pop: { throughput: '100K', latency: '10μs', speedup: '1.25x' },
    },
    stream: {
        throughput: 'High',
        latency: 'Low',
        description: 'Kafka-like pub/sub',
    },
    db: {
        insert: { throughput: '76K', latency: '13μs' },
    },
};

// Features data
export const FEATURES = [
    {
        id: 'kv',
        title: 'Key-Value Store',
        icon: '🔑',
        description: 'Lightning-fast KV operations with 319K reads/sec and sub-10μs latency',
        highlights: [
            '319K ops/sec read throughput',
            '151K ops/sec write throughput',
            'Atomic batch operations',
            'Dual storage modes',
        ],
        color: 'cyan',
    },
    {
        id: 'queue',
        title: 'Message Queue',
        icon: '📬',
        description: 'Durable message queue with 104K push/sec on unified port',
        highlights: [
            '104K ops/sec push throughput',
            '100K ops/sec pop throughput',
            'Unified port with KV',
            'BadgerDB persistence',
        ],
        color: 'purple',
    },
    {
        id: 'stream',
        title: 'Stream Processing',
        icon: '🌊',
        description: 'Kafka-like pub/sub with partitions and consumer groups',
        highlights: [
            'Partitioned topics',
            'Consumer groups',
            'Offset management',
            'At-least-once delivery',
        ],
        color: 'cyan',
    },
    {
        id: 'db',
        title: 'Document Database',
        icon: '📄',
        description: 'MongoDB-like document store with Prisma-like query builder',
        highlights: [
            '76K inserts/sec throughput',
            '13μs average latency',
            'Fluent query builder',
            'Secondary indexes',
        ],
        color: 'purple',
    },
];

// Navigation links
export const NAV_LINKS = [
    { href: '/', label: 'Home' },
    { href: '/docs', label: 'Documentation' },
    { href: 'https://github.com/skshohagmiah/flin', label: 'GitHub', external: true },
];

// Documentation sections
export const DOC_SECTIONS = [
    {
        title: 'Getting Started',
        items: [
            { href: '/docs', label: 'Overview' },
            { href: '/docs/getting-started', label: 'Installation' },
            { href: '/docs/getting-started#quick-start', label: 'Quick Start' },
        ],
    },
    {
        title: 'API Reference',
        items: [
            { href: '/docs/kv-store', label: 'Key-Value Store' },
            { href: '/docs/queue', label: 'Message Queue' },
            { href: '/docs/stream', label: 'Stream Processing' },
            { href: '/docs/database', label: 'Document Database' },
        ],
    },
    {
        title: 'Deployment',
        items: [
            { href: '/docs/clustering', label: 'Clustering' },
            { href: '/docs/clustering#docker', label: 'Docker Setup' },
            { href: '/docs/clustering#configuration', label: 'Configuration' },
        ],
    },
];

// Code examples
export const CODE_EXAMPLES = {
    docker: `# Single node
cd docker/single && ./run.sh

# 3-node cluster
cd docker/cluster && ./run.sh`,

    local: `# Clone and build
git clone https://github.com/skshohagmiah/flin
cd flin
go build -o bin/flin-server ./cmd/server

# Run server
./bin/flin-server \\
  -node-id=node1 \\
  -http=:8080 \\
  -raft=:9080 \\
  -port=:7380 \\
  -data=./data/node1 \\
  -workers=256`,

    client: `import flin "github.com/skshohagmiah/flin/clients/go"

// Create unified client
opts := flin.DefaultOptions("localhost:7380")
client, _ := flin.NewClient(opts)
defer client.Close()

// KV Store
client.KV.Set("user:1", []byte("John Doe"))
value, _ := client.KV.Get("user:1")

// Message Queue
client.Queue.Push("tasks", []byte("Task 1"))
msg, _ := client.Queue.Pop("tasks")

// Stream Processing
client.Stream.CreateTopic("events", 4, 7*24*60*60*1000)
client.Stream.Publish("events", -1, "key", []byte("data"))

// Document Database
id, _ := client.DB.Insert("users", map[string]interface{}{
    "name": "John Doe",
    "email": "john@example.com",
})`,

    kvStore: `// Set a value
err := client.KV.Set("user:101", []byte("Alice"))

// Get a value
val, err := client.KV.Get("user:101")

// Delete a key
err := client.KV.Delete("user:101")

// Atomic counter
newVal, err := client.KV.Incr("visits:page:home")

// Batch operations
client.KV.MSet([]string{"k1", "k2"}, [][]byte{
    []byte("v1"), 
    []byte("v2"),
})
values, _ := client.KV.MGet([]string{"k1", "k2"})`,

    queue: `// Push items to queue
err := client.Queue.Push("email_tasks", 
    []byte(\`{"to":"user@example.com"}\`))

// Pop item from queue
task, err := client.Queue.Pop("email_tasks")

// Peek without removing
task, err := client.Queue.Peek("email_tasks")

// Get queue length
count, err := client.Queue.Len("email_tasks")

// Clear queue
err := client.Queue.Clear("email_tasks")`,

    stream: `// Create topic with 4 partitions, 7 days retention
err := client.Stream.CreateTopic("logs", 4, 7*24*60*60*1000)

// Publish message (auto-partition with key hash)
err := client.Stream.Publish("logs", -1, "server-1", 
    []byte("Error: 500"))

// Subscribe consumer to group
err := client.Stream.Subscribe("logs", "processors", "worker-1")

// Consume messages
msgs, err := client.Stream.Consume("logs", "processors", "worker-1", 10)
for _, msg := range msgs {
    fmt.Printf("Partition: %d, Offset: %d\\n", 
        msg.Partition, msg.Offset)
    
    // Commit offset after processing
    client.Stream.Commit("logs", "processors", 
        msg.Partition, msg.Offset+1)
}`,

    database: `// Insert document
id, err := client.DB.Insert("users", map[string]interface{}{
    "name":  "John Doe",
    "email": "john@example.com",
    "age":   30,
})

// Query with fluent API (Prisma-like)
users, err := client.DB.Query("users").
    Where("age", flin.Gte, 18).
    Where("status", flin.Eq, "active").
    OrderBy("created_at", flin.Desc).
    Skip(0).
    Take(10).
    Exec()

// Update documents
err := client.DB.Update("users").
    Where("email", flin.Eq, "john@example.com").
    Set("age", 31).
    Set("verified", true).
    Exec()

// Delete documents
err := client.DB.Delete("users").
    Where("status", flin.Eq, "inactive").
    Exec()`,

    clustering: `# Node 1 (bootstrap)
./bin/flin-server \\
  -node-id=node1 \\
  -http=:8080 \\
  -raft=:9080 \\
  -port=:7380

# Node 2 (join cluster)
./bin/flin-server \\
  -node-id=node2 \\
  -http=:8081 \\
  -raft=:9081 \\
  -port=:7381 \\
  -join=localhost:8080

# Node 3 (join cluster)
./bin/flin-server \\
  -node-id=node3 \\
  -http=:8082 \\
  -raft=:9082 \\
  -port=:7382 \\
  -join=localhost:8080`,
};

// Architecture layers
export const ARCHITECTURE_LAYERS = [
    {
        name: 'Client SDKs',
        description: 'Go, Python, and other language clients',
        color: 'cyan',
        tech: ['Go Client', 'Python Client', 'Binary Protocol'],
    },
    {
        name: 'Binary Protocol',
        description: 'Auto-detection for optimal performance',
        color: 'purple',
        tech: ['Auto-detection', 'Optimized Encoding', 'Fast Path'],
    },
    {
        name: 'Server Layer',
        description: 'Hybrid fast path + worker pool',
        color: 'cyan',
        tech: ['Fast Path', 'Worker Pool', 'Connection Manager'],
    },
    {
        name: 'Abstraction Layer',
        description: 'KV, Queue, Stream, Document operations',
        color: 'purple',
        tech: ['KV Store', 'Message Queue', 'Stream', 'Document DB'],
    },
    {
        name: 'Storage Layer',
        description: 'BadgerDB for persistence',
        color: 'cyan',
        tech: ['BadgerDB', 'LSM Tree', 'WAL'],
    },
    {
        name: 'ClusterKit',
        description: 'Raft consensus & replication',
        color: 'purple',
        tech: ['Raft Consensus', 'Replication', 'Leader Election'],
    },
];
