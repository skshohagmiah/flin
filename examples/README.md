# 📚 Flin KV Store - Examples

Production-ready examples demonstrating Flin in real-world scenarios.

## Available Examples

### 1. Web Application (`web-app/`)

A complete REST API demonstrating:
- ✅ Session management with TTL
- ✅ User data caching
- ✅ Authentication/Authorization
- ✅ High-performance reads (787K ops/sec)
- ✅ Docker deployment

**Quick Start:**
```bash
cd web-app
docker-compose up -d
./test.sh
```

**Features:**
- Session store (30-minute TTL)
- User cache (5-minute TTL)
- RESTful API
- Health checks
- Production-ready

[Full Documentation](./web-app/README.md)

## Running Examples

### Option 1: Docker (Recommended)

```bash
# Navigate to example
cd examples/web-app

# Start with Docker Compose
docker-compose up -d

# Run tests
./test.sh

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Option 2: Local Development

```bash
# Navigate to example
cd examples/web-app

# Install dependencies
go mod tidy

# Run
go run main.go

# In another terminal, run tests
./test.sh
```

## Example Use Cases

### 1. Session Store

```go
// Store session with 30-minute TTL
sessionData, _ := json.Marshal(session)
store.Set("session:"+token, sessionData, 30*time.Minute)

// Retrieve session
data, err := store.Get("session:" + token)
if err != nil {
    // Session expired or invalid
}
```

**Performance:**
- Write: ~35μs
- Read: ~3μs
- Throughput: 103K sessions/sec

### 2. Cache Layer

```go
// Try cache first
if cachedData, err := store.Get("cache:"+key); err == nil {
    return cachedData // Cache hit - super fast!
}

// Cache miss - fetch and store
data := fetchFromDatabase(key)
store.Set("cache:"+key, data, 5*time.Minute)
```

**Performance:**
- Cache Hit: ~3μs (787K ops/sec)
- Cache Miss: ~35μs
- Hit Rate: >90% in production

### 3. Rate Limiting

```go
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

### 4. Distributed Locks

```go
lockKey := "lock:" + resourceID
if err := store.Set(lockKey, []byte("locked"), 10*time.Second); err == nil {
    defer store.Delete(lockKey)
    // Do work with lock
}
```

## Performance Benchmarks

### Web Application Example

```bash
# Run performance test
curl http://localhost:8080/api/cache
```

**Expected Results:**
```json
{
  "write_latency_us": 35-45,
  "read_latency_us": 3-8,
  "message": "Flin provides sub-40μs writes and sub-5μs reads!"
}
```

### Load Testing

```bash
# Install Apache Bench
sudo apt-get install apache2-utils

# Test cache reads (should handle 50K+ req/sec)
ab -n 100000 -c 100 http://localhost:8080/api/users/USER_ID
```

**Expected:**
- Requests/sec: 50,000+
- Time/request: 2ms
- Failed requests: 0

## Comparison with Redis

| Metric | Flin (Example) | Redis (Network) |
|--------|----------------|-----------------|
| Session Read | 3μs | ~100μs |
| Session Write | 35μs | ~100μs |
| Cache Hit | 3μs | ~100μs |
| Throughput | 787K ops/sec | 80-100K ops/sec |
| Network | None | TCP overhead |
| Deployment | Embedded | Separate service |

**Flin is 30x faster for reads!**

## Production Deployment

### Docker Swarm

```bash
cd examples/web-app

# Initialize swarm
docker swarm init

# Deploy
docker stack deploy -c docker-compose.yml flin-app

# Scale to 3 replicas
docker service scale flin-app_web-app=3

# Check status
docker service ls
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flin-web-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flin-web-app
  template:
    metadata:
      labels:
        app: flin-web-app
    spec:
      containers:
      - name: web-app
        image: flin-web-app:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: data
          mountPath: /app/data
        livenessProbe:
          httpGet:
            path: /api/stats
            port: 8080
        readinessProbe:
          httpGet:
            path: /api/stats
            port: 8080
```

## Best Practices

### 1. Use Appropriate TTLs

```go
// Sessions: 30 minutes
sessionTTL := 30 * time.Minute

// Cache: 5 minutes
cacheTTL := 5 * time.Minute

// Rate limits: 1 minute
rateLimitTTL := 1 * time.Minute
```

### 2. Handle Cache Misses Gracefully

```go
// Try cache first
if data, err := store.Get(cacheKey); err == nil {
    return data
}

// Fallback to source
data := fetchFromSource()
store.Set(cacheKey, data, cacheTTL)
return data
```

### 3. Invalidate Cache on Updates

```go
// Update user
updateUser(userID, newData)

// Invalidate cache
store.Delete("cache:user:" + userID)
```

### 4. Use Batch Operations for Bulk Data

```go
// Slow: 103K ops/sec
for _, user := range users {
    store.Set("user:"+user.ID, userData, 0)
}

// Fast: 500K+ ops/sec
batch := make(map[string][]byte)
for _, user := range users {
    batch["user:"+user.ID] = userData
}
store.BatchSet(batch)
```

### 5. Monitor Performance

```go
// Add metrics
start := time.Now()
store.Get(key)
latency := time.Since(start)

// Log slow operations
if latency > 1*time.Millisecond {
    log.Printf("Slow operation: %v", latency)
}
```

## Troubleshooting

### High Latency

```bash
# Check disk I/O
iostat -x 1

# Use SSD for data directory
# Increase cache size in code
```

### Memory Usage

```bash
# Monitor memory
docker stats flin-web-app

# Reduce cache if needed
opts.BlockCacheSize = 256 << 20  // 256MB
```

### Connection Issues

```bash
# Check if server is running
curl http://localhost:8080/api/stats

# Check logs
docker-compose logs -f
```

## Adding Your Own Example

1. Create directory: `examples/your-example/`
2. Add `main.go`, `Dockerfile`, `docker-compose.yml`
3. Add `README.md` with documentation
4. Add `test.sh` for automated testing
5. Update this README

## Resources

- [Main Documentation](../README.md)
- [Performance Guide](../BENCHMARKS.md)
- [Docker Guide](../DOCKER.md)
- [Getting Started](../GETTING_STARTED.md)

## Support

- **Issues**: Open a GitHub issue
- **Questions**: Check documentation
- **Contributions**: PRs welcome!

---

**Start with the web-app example to see Flin in action!** 🚀

```bash
cd examples/web-app
docker-compose up -d
./test.sh
```
