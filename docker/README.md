# Flin Docker Deployments

Organized Docker configurations for running Flin in different modes.

## 📁 Structure

```
docker/
├── single/              # Single node deployment
│   ├── docker-compose.yml
│   ├── run.sh          # Start + benchmark
│   └── README.md
│
└── cluster/             # 3-node cluster deployment
    ├── docker-compose.yml
    ├── run.sh          # Start + benchmark
    └── README.md
```

## 🚀 Quick Start

### Single Node (Development)

```bash
cd docker/single
./run.sh
```

Access at `http://localhost:6380` (KV) and `http://localhost:8080` (HTTP)

### 3-Node Cluster (Production)

```bash
cd docker/cluster
./run.sh
```

Access nodes at:
- Node 1: `localhost:6380` / `localhost:8080`
- Node 2: `localhost:6381` / `localhost:8081`
- Node 3: `localhost:6382` / `localhost:8082`

Both scripts automatically run performance benchmarks after starting!

## 📖 Detailed Documentation

Each directory has its own README with detailed instructions:

- [single/README.md](single/README.md) - Single node setup
- [cluster/README.md](cluster/README.md) - Cluster setup

## 🎯 Use Cases

| Setup | Use Case | Command |
|-------|----------|---------|
| **Single** | Development, quick testing | `cd docker/single && ./run.sh` |
| **Cluster** | Production, high availability | `cd docker/cluster && ./run.sh` |

## 🛠️ Common Commands

```bash
# Start
cd docker/<single|cluster>
./run.sh

# Stop
docker compose down -v

# View logs
docker compose logs -f

# Restart
docker compose restart

# Check status
docker compose ps
```

## 📊 Performance Output Example

```
📊 Running Cluster Performance Benchmark
=========================================

⚡ Write Performance (5 seconds, distributed across 3 nodes)...
  ✓ Writes: 1250 operations
  ✓ Throughput: 250 ops/sec
  ✓ Per node: ~83 ops/sec

⚡ Read Performance (5 seconds, distributed across 3 nodes)...
  ✓ Reads: 2340 operations
  ✓ Throughput: 468 ops/sec
  ✓ Per node: ~156 ops/sec

=========================================
✅ Cluster running with performance metrics!
```
