# Flin Performance Test Scripts

Performance testing scripts for Flin KV Store and Message Queue.

## 📋 Available Tests

### 1. Queue Performance Test
```bash
./test_queue_performance.sh
```

**Tests:**
- ✅ Enqueue performance (100 workers)
- ✅ Dequeue performance (100 workers)
- ✅ Priority queue performance
- ✅ Consumer performance (100K messages)

**Expected Results:**
- Enqueue: 50-100K ops/sec
- Dequeue: 40-80K ops/sec
- Consume: 30-50K msgs/sec

---

### 2. KV Performance Test
```bash
./test_kv_performance.sh
```

**Tests:**
- ✅ Single SET operations (256 workers, 1KB values)
- ✅ Single GET operations (256 workers)
- ✅ Batch operations (10 keys per batch)
- ✅ Mixed workload (70% reads, 30% writes)

**Expected Results:**
- Single SET: 100-150K ops/sec
- Single GET: 150-200K ops/sec
- Batch (10 keys): 500-700K ops/sec
- Mixed workload: 120-180K ops/sec

---

### 3. Unified Performance Test
```bash
./test_unified_performance.sh
```

**Tests:**
- ✅ KV Store performance
- ✅ Queue performance
- ✅ Mixed KV + Queue operations
- ✅ Real-world scenario (store order + queue task)
- ✅ Consumer performance

**Expected Results:**
- KV operations: 100-150K ops/sec
- Queue operations: 50-100K ops/sec
- Mixed operations: 80-120K ops/sec
- Real-world scenario: 60-100K orders/sec

---

### 4. Queue Functionality Test
```bash
./test-queue.sh
```

**Tests:**
- ✅ Enqueue/Dequeue
- ✅ Priority queues
- ✅ Consumer pattern
- ✅ Message headers
- ✅ Message acknowledgment

---

## 🚀 Quick Start

Run performance tests from anywhere:
```bash
# From project root
./test_scripts/test_queue_performance.sh
./test_scripts/test_kv_performance.sh
./test_scripts/test_unified_performance.sh

# Or from test_scripts folder
cd test_scripts
./test_queue_performance.sh
./test_kv_performance.sh
./test_unified_performance.sh
```

Scripts automatically change to project root before running.

## 📊 Performance Targets

| Component | Operation | Target | Status |
|-----------|-----------|--------|--------|
| **KV Store** | Single SET | 100K+ ops/sec | ✅ |
| **KV Store** | Single GET | 150K+ ops/sec | ✅ |
| **KV Store** | Batch (10) | 500K+ ops/sec | ✅ |
| **Queue** | Enqueue | 50K+ ops/sec | ✅ |
| **Queue** | Dequeue | 40K+ ops/sec | ✅ |
| **Queue** | Consume | 30K+ msgs/sec | ✅ |
| **Unified** | Mixed ops | 80K+ ops/sec | ✅ |

## 🎯 Benchmarking Tips

1. **Warm-up**: Run tests twice, use second run results
2. **Isolation**: Close other applications during tests
3. **Consistency**: Run tests multiple times for average
4. **Hardware**: Results vary based on CPU/RAM/Disk

## 📝 Notes

- All tests use in-memory queue storage for consistency
- KV tests use disk storage to simulate production
- Worker counts optimized for modern multi-core CPUs
- 1KB value size used for realistic workloads

## 🔧 Customization

Edit test scripts to adjust:
- Worker count
- Test duration
- Value sizes
- Batch sizes
- Concurrency levels

## 📈 Monitoring

During tests, monitor:
- CPU usage
- Memory usage
- Disk I/O (for KV tests)
- Network (for cluster tests)

## ✅ Success Criteria

Tests pass if:
- No errors or panics
- Throughput meets targets
- Latency < 100μs (single ops)
- Latency < 10μs (batch ops)
