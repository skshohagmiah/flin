# 🐳 Docker Guide for Flin KV Store

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Build and start the server
make docker-build
make docker-run

# Or manually:
docker-compose up -d
```

### Using CLI in Docker

```bash
# Set a key
make docker-cli ARGS='set mykey "hello world"'

# Get a key
make docker-cli ARGS='get mykey'

# Delete a key
make docker-cli ARGS='delete mykey'

# Check if key exists
make docker-cli ARGS='exists mykey'

# Run benchmark
make docker-benchmark
```

## Building the Image

### Build with Docker

```bash
docker build -t flin-kv:latest .
```

### Build with Make

```bash
make docker-build
```

This creates a multi-stage build:
- **Builder stage**: Compiles Go binaries
- **Final stage**: Minimal Alpine image (~20MB)

## Running the Server

### With Docker Compose

```bash
# Start server
docker-compose up -d

# View logs
docker-compose logs -f flin-server

# Stop server
docker-compose down
```

### With Docker Run

```bash
# Run server
docker run -d \
  --name flin-server \
  -p 6380:6380 \
  -v flin-data:/data \
  flin-kv:latest ./kvserver

# Run CLI
docker run --rm \
  -v flin-data:/data \
  flin-kv:latest ./flin set mykey "hello"
```

## CLI Usage in Docker

### Basic Operations

```bash
# Set a key-value pair
docker-compose run --rm flin-cli ./flin set mykey "hello world"

# Set with TTL (expires in 3600 seconds)
docker-compose run --rm flin-cli ./flin set session:123 "data" 3600

# Get a value
docker-compose run --rm flin-cli ./flin get mykey

# Delete a key
docker-compose run --rm flin-cli ./flin delete mykey

# Check existence
docker-compose run --rm flin-cli ./flin exists mykey
```

### Benchmark in Docker

```bash
# Run full benchmark
docker-compose run --rm flin-cli ./flin benchmark

# With custom duration
docker-compose run --rm \
  -e FLIN_BENCHMARK_DURATION=30s \
  flin-cli ./flin benchmark
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FLIN_DATA_DIR` | `/data` | Data directory path |
| `FLIN_BENCHMARK_DURATION` | `10s` | Benchmark duration |

### Docker Compose Configuration

Edit `docker-compose.yml`:

```yaml
services:
  flin-server:
    environment:
      - FLIN_DATA_DIR=/data
    ports:
      - "6380:6380"  # Change port mapping
    volumes:
      - flin-data:/data  # Persistent storage
```

## Volumes

### Data Persistence

Data is stored in a Docker volume:

```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect flin_flin-data

# Backup data
docker run --rm \
  -v flin_flin-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/flin-backup.tar.gz /data

# Restore data
docker run --rm \
  -v flin_flin-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/flin-backup.tar.gz -C /
```

### Remove Data

```bash
# Stop and remove everything
docker-compose down -v

# Or with make
make clean
```

## Health Checks

The server includes a health check:

```yaml
healthcheck:
  test: ["CMD", "./flin", "set", "healthcheck", "ok"]
  interval: 30s
  timeout: 10s
  retries: 3
```

Check health status:

```bash
docker ps
# Look for "healthy" in STATUS column
```

## Performance Testing in Docker

### Run Benchmark

```bash
# Quick benchmark (10s)
make docker-benchmark

# Extended benchmark (60s)
docker-compose run --rm \
  -e FLIN_BENCHMARK_DURATION=60s \
  flin-cli ./flin benchmark
```

### Expected Results

```
📝 Benchmarking SET operations (10s)...
   Throughput: 80-100K ops/sec

📖 Benchmarking GET operations (10s)...
   Throughput: 400-600K ops/sec
```

**Note:** Docker adds ~20% overhead compared to native performance.

## Multi-Container Setup

### Multiple Flin Instances

```yaml
version: '3.8'

services:
  flin-1:
    build: .
    command: ./kvserver
    ports:
      - "6380:6380"
    volumes:
      - flin-data-1:/data

  flin-2:
    build: .
    command: ./kvserver
    ports:
      - "6381:6380"
    volumes:
      - flin-data-2:/data

volumes:
  flin-data-1:
  flin-data-2:
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs flin-server

# Check if port is in use
lsof -i :6380

# Restart container
docker-compose restart flin-server
```

### Permission Issues

```bash
# Fix volume permissions
docker-compose run --rm flin-cli chown -R root:root /data
```

### Out of Memory

Increase Docker memory limit:

```yaml
services:
  flin-server:
    deploy:
      resources:
        limits:
          memory: 2G
```

### Slow Performance

1. **Use volumes, not bind mounts** - Volumes are faster
2. **Increase memory** - More cache = better performance
3. **Use SSD** - Docker Desktop should use SSD
4. **Check CPU limits** - Remove CPU constraints

## Production Deployment

### Docker Swarm

```bash
# Initialize swarm
docker swarm init

# Deploy stack
docker stack deploy -c docker-compose.yml flin

# Scale service
docker service scale flin_flin-server=3
```

### Kubernetes

Example deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flin-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flin
  template:
    metadata:
      labels:
        app: flin
    spec:
      containers:
      - name: flin
        image: flin-kv:latest
        command: ["./kvserver"]
        ports:
        - containerPort: 6380
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: flin-pvc
```

## Makefile Commands

All available commands:

```bash
make help                    # Show all commands
make docker-build            # Build Docker image
make docker-run              # Start containers
make docker-stop             # Stop containers
make docker-logs             # View logs
make docker-cli ARGS='...'   # Run CLI command
make docker-benchmark        # Run benchmark
make clean                   # Clean everything
```

## Examples

### Complete Workflow

```bash
# 1. Build image
make docker-build

# 2. Start server
make docker-run

# 3. Set some data
make docker-cli ARGS='set user:1 "John Doe"'
make docker-cli ARGS='set user:2 "Jane Smith"'

# 4. Get data
make docker-cli ARGS='get user:1'

# 5. Run benchmark
make docker-benchmark

# 6. View logs
make docker-logs

# 7. Stop server
make docker-stop
```

### Batch Operations

```bash
# Create a script
cat > batch.sh << 'EOF'
#!/bin/bash
for i in {1..100}; do
  docker-compose run --rm flin-cli ./flin set "key_$i" "value_$i"
done
EOF

chmod +x batch.sh
./batch.sh
```

## Image Details

### Size

```bash
docker images flin-kv
# REPOSITORY   TAG      SIZE
# flin-kv      latest   ~20MB
```

### Layers

- Base: Alpine Linux (~5MB)
- Binaries: flin + kvserver (~15MB)
- Total: ~20MB

### Security

- Non-root user (root in Alpine, but minimal attack surface)
- No unnecessary packages
- Minimal base image
- Static binaries (no dynamic linking)

## Monitoring

### Container Stats

```bash
# Real-time stats
docker stats flin-server

# Resource usage
docker-compose top
```

### Logs

```bash
# Follow logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100

# Specific service
docker-compose logs flin-server
```

## Backup and Restore

### Automated Backup

```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
docker run --rm \
  -v flin_flin-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/flin-$DATE.tar.gz /data
```

### Restore from Backup

```bash
#!/bin/bash
# restore.sh
BACKUP_FILE=$1
docker run --rm \
  -v flin_flin-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar xzf /backup/$BACKUP_FILE -C /
```

---

## Summary

✅ **Multi-stage build** - Minimal 20MB image
✅ **Docker Compose** - Easy orchestration
✅ **Makefile** - Simple commands
✅ **Persistent volumes** - Data safety
✅ **Health checks** - Production ready
✅ **CLI in Docker** - Full functionality

**Get started:**

```bash
make docker-build
make docker-run
make docker-cli ARGS='benchmark'
```
