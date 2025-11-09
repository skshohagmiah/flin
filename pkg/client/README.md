# 🚀 Flin Go SDK Client

Official Go client library for Flin KV Store.

## Features

- ✅ **Simple API** - Easy to use interface
- ✅ **Connection Pooling** - High-performance connection management
- ✅ **Auto Reconnect** - Automatic reconnection on failure
- ✅ **Thread-Safe** - Safe for concurrent use
- ✅ **Timeout Support** - Configurable timeouts
- ✅ **Health Checks** - Built-in ping/health check

## Installation

```bash
go get github.com/skshohagmiah/flin/pkg/client
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/skshohagmiah/flin/pkg/client"
)

func main() {
    // Create client
    c, err := client.New("localhost:6380")
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // Set a key
    err = c.Set("mykey", []byte("hello world"))
    if err != nil {
        log.Fatal(err)
    }
    
    // Get a key
    value, err := c.Get("mykey")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(value)) // "hello world"
    
    // Check if key exists
    exists, err := c.Exists("mykey")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(exists) // true
    
    // Delete a key
    err = c.Delete("mykey")
    if err != nil {
        log.Fatal(err)
    }
}
```

### Using Connection Pool (Recommended for Production)

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/skshohagmiah/flin/pkg/client"
)

func main() {
    // Create pooled client
    config := client.DefaultPoolConfig()
    config.Address = "localhost:6380"
    config.MinConns = 5
    config.MaxConns = 20
    
    pc, err := client.NewPooledClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer pc.Close()
    
    // Use like regular client
    err = pc.Set("key1", []byte("value1"))
    value, err := pc.Get("key1")
    
    // Check pool stats
    stats := pc.Stats()
    fmt.Printf("Pool stats: %+v\n", stats)
}
```

## API Reference

### Client

#### Creating a Client

```go
// Simple client
client, err := client.New("localhost:6380")

// Client with custom config
config := &client.Config{
    Address:    "localhost:6380",
    Timeout:    10 * time.Second,
    MaxRetries: 5,
}
client, err := client.NewWithConfig(config)
```

#### Methods

**Set(key string, value []byte) error**

Store a key-value pair.

```go
err := client.Set("user:1", []byte("John Doe"))
```

**Get(key string) ([]byte, error)**

Retrieve a value by key.

```go
value, err := client.Get("user:1")
if err != nil {
    // Key not found or error
}
```

**Delete(key string) error**

Remove a key.

```go
err := client.Delete("user:1")
```

**Exists(key string) (bool, error)**

Check if a key exists.

```go
exists, err := client.Exists("user:1")
```

**Ping() error**

Check if server is alive.

```go
err := client.Ping()
if err != nil {
    // Server is down
}
```

**Close() error**

Close the connection.

```go
defer client.Close()
```

**Reconnect() error**

Manually reconnect to the server.

```go
err := client.Reconnect()
```

### Connection Pool

#### Creating a Pool

```go
config := &client.PoolConfig{
    Address:     "localhost:6380",
    MinConns:    2,
    MaxConns:    10,
    Timeout:     5 * time.Second,
    IdleTimeout: 5 * time.Minute,
}

pool, err := client.NewPool(config)
defer pool.Close()
```

#### Using the Pool

```go
// Get connection from pool
conn, err := pool.Get()
if err != nil {
    log.Fatal(err)
}

// Use connection
err = conn.Set("key", []byte("value"))

// Return to pool
pool.Put(conn)
```

#### PooledClient (Easier)

```go
pc, err := client.NewPooledClient(config)
defer pc.Close()

// Automatically manages connections
pc.Set("key", []byte("value"))
pc.Get("key")
```

## Configuration

### Client Config

```go
type Config struct {
    Address    string        // Server address (e.g., "localhost:6380")
    Timeout    time.Duration // Connection timeout (default: 5s)
    MaxRetries int           // Max retry attempts (default: 3)
}
```

### Pool Config

```go
type PoolConfig struct {
    Address     string        // Server address
    MinConns    int           // Minimum connections (default: 2)
    MaxConns    int           // Maximum connections (default: 10)
    Timeout     time.Duration // Connection timeout (default: 5s)
    IdleTimeout time.Duration // Idle timeout (default: 5m)
}
```

## Examples

### Session Store

```go
package main

import (
    "encoding/json"
    "time"
    
    "github.com/skshohagmiah/flin/pkg/client"
)

type Session struct {
    UserID    string
    ExpiresAt time.Time
}

func main() {
    c, _ := client.New("localhost:6380")
    defer c.Close()
    
    // Store session
    session := Session{
        UserID:    "user123",
        ExpiresAt: time.Now().Add(30 * time.Minute),
    }
    
    data, _ := json.Marshal(session)
    c.Set("session:abc123", data)
    
    // Retrieve session
    data, _ = c.Get("session:abc123")
    var retrieved Session
    json.Unmarshal(data, &retrieved)
}
```

### Cache Layer

```go
func GetUser(id string, c *client.Client) (*User, error) {
    // Try cache first
    cacheKey := "cache:user:" + id
    if data, err := c.Get(cacheKey); err == nil {
        var user User
        json.Unmarshal(data, &user)
        return &user, nil
    }
    
    // Cache miss - fetch from database
    user := fetchFromDatabase(id)
    
    // Store in cache
    data, _ := json.Marshal(user)
    c.Set(cacheKey, data)
    
    return user, nil
}
```

### Concurrent Operations

```go
func main() {
    pc, _ := client.NewPooledClient(client.DefaultPoolConfig())
    defer pc.Close()
    
    var wg sync.WaitGroup
    
    // Concurrent writes
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            key := fmt.Sprintf("key_%d", i)
            pc.Set(key, []byte(fmt.Sprintf("value_%d", i)))
        }(i)
    }
    
    wg.Wait()
}
```

### Health Check

```go
func checkHealth(c *client.Client) bool {
    err := c.Ping()
    return err == nil
}

func main() {
    c, _ := client.New("localhost:6380")
    defer c.Close()
    
    // Periodic health check
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        if !checkHealth(c) {
            log.Println("Server is down, attempting reconnect...")
            c.Reconnect()
        }
    }
}
```

## Performance

### Benchmarks

```go
// Single client: ~100K ops/sec
// Pooled client (10 conns): ~500K ops/sec
```

### Best Practices

1. **Use Connection Pooling** for high-throughput applications
2. **Reuse Clients** - Don't create new clients for each operation
3. **Handle Errors** - Always check for connection errors
4. **Set Timeouts** - Use appropriate timeouts for your use case
5. **Close Connections** - Always defer Close()

### Optimal Pool Size

```go
// For CPU-bound workloads
config.MaxConns = runtime.NumCPU()

// For I/O-bound workloads
config.MaxConns = runtime.NumCPU() * 2

// For high-throughput
config.MaxConns = 20-50
```

## Error Handling

```go
value, err := client.Get("key")
if err != nil {
    switch {
    case errors.Is(err, client.ErrPoolClosed):
        // Pool is closed
    case errors.Is(err, client.ErrPoolEmpty):
        // No available connections
    default:
        // Other errors (network, server, etc.)
    }
}
```

## Testing

```go
func TestClient(t *testing.T) {
    c, err := client.New("localhost:6380")
    if err != nil {
        t.Fatal(err)
    }
    defer c.Close()
    
    // Test Set
    err = c.Set("testkey", []byte("testvalue"))
    if err != nil {
        t.Fatal(err)
    }
    
    // Test Get
    value, err := c.Get("testkey")
    if err != nil {
        t.Fatal(err)
    }
    
    if string(value) != "testvalue" {
        t.Errorf("Expected 'testvalue', got '%s'", value)
    }
    
    // Cleanup
    c.Delete("testkey")
}
```

## Troubleshooting

### Connection Refused

```go
// Check if server is running
// Start server: ./kvserver

// Check address
config.Address = "localhost:6380" // Correct port?
```

### Timeout Errors

```go
// Increase timeout
config.Timeout = 30 * time.Second

// Check network latency
// Check server load
```

### Pool Exhaustion

```go
// Increase max connections
config.MaxConns = 50

// Check if connections are being returned
defer pool.Put(conn) // Don't forget!
```

## Comparison with Other Clients

| Feature | Flin Client | Redis Client |
|---------|-------------|--------------|
| Connection Pooling | ✅ | ✅ |
| Auto Reconnect | ✅ | ✅ |
| Pipelining | ⏳ Coming | ✅ |
| Pub/Sub | ⏳ Coming | ✅ |
| Transactions | ⏳ Coming | ✅ |
| Simple API | ✅ | ✅ |

## Roadmap

- [ ] Pipelining support
- [ ] Batch operations
- [ ] Pub/Sub messaging
- [ ] Transactions
- [ ] Cluster support
- [ ] Metrics/observability

## License

MIT License - see LICENSE file for details

## Support

- **Issues**: Open a GitHub issue
- **Documentation**: See main Flin docs
- **Examples**: Check `/examples` directory

---

**Start using Flin SDK today!** 🚀

```bash
go get github.com/skshohagmiah/flin/pkg/client
```
