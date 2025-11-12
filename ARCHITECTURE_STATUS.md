# Flin Architecture - Current Status

## 🎯 What You Have Now

### ✅ Working Components

1. **ClusterKit Integration** ✅
   - Raft consensus for cluster coordination
   - Partition management (64 partitions)
   - Primary + replica assignment
   - Health checking and failure detection
   - Automatic rebalancing

2. **Client-Side Partitioning** ✅
   - Go client uses consistent hashing
   - Routes requests to correct node based on key
   - Connection pooling per node

3. **Local Storage** ✅
   - BadgerDB for persistence
   - Fast read/write operations
   - 96K writes/sec, 120K reads/sec (single node)

### ❌ Missing: Actual Data Replication

**Current Flow:**
```
Client → Hash(key) → Node X → Write to local BadgerDB
                              (NO replication!)
```

**Problem:**
- Each node only has its own partition's data
- No redundancy - data loss if node fails
- ClusterKit knows about replicas but doesn't use them

---

## 🏗️ Target Architecture (What You Want)

### Async Replication Model

```
Write Request
    ↓
1. Client routes to Primary Node (via consistent hashing)
    ↓
2. Primary writes to local storage (fast)
    ↓
3. Primary returns SUCCESS immediately
    ↓
4. Primary async replicates to Replicas (background)
    ↓
5. Replicas write to their local storage
```

**Benefits:**
- ✅ Fast writes (no waiting for replicas)
- ✅ High throughput
- ✅ Eventual consistency
- ✅ Survives node failures (data on replicas)

---

## 🔧 What Needs to Be Implemented

### 1. Replication Layer in KV Server

Add async replication after local write:

```go
func (s *KVServer) handleSet(key string, value []byte) error {
    // 1. Get partition info from ClusterKit
    partition, err := s.ck.GetPartition(key)
    if err != nil {
        return err
    }
    
    // 2. Check if I'm primary or replica
    if !s.ck.IsPrimary(partition) && !s.ck.IsReplica(partition) {
        // Forward to primary
        primary := s.ck.GetPrimary(partition)
        return s.forwardToPrimary(primary, key, value)
    }
    
    // 3. Write to local storage (fast)
    if err := s.store.Set(key, value, 0); err != nil {
        return err
    }
    
    // 4. If I'm primary, async replicate to replicas
    if s.ck.IsPrimary(partition) {
        replicas := s.ck.GetReplicas(partition)
        go s.replicateAsync(replicas, key, value)
    }
    
    return nil
}

func (s *KVServer) replicateAsync(replicas []clusterkit.Node, key string, value []byte) {
    for _, replica := range replicas {
        go func(node clusterkit.Node) {
            // Send replication request to replica
            conn, err := net.Dial("tcp", node.Address)
            if err != nil {
                log.Printf("Replication failed to %s: %v", node.ID, err)
                return
            }
            defer conn.Close()
            
            // Send SET command with replication flag
            request := protocol.EncodeReplicationSet(key, value)
            conn.Write(request)
        }(replica)
    }
}
```

### 2. Replication Protocol

Add new protocol commands:

```go
const (
    OpReplicationSet    = 0x10  // Replicate SET
    OpReplicationDelete = 0x11  // Replicate DELETE
)
```

### 3. Handle Replica Writes

Replicas should accept replication writes without re-replicating:

```go
func (s *KVServer) handleReplicationSet(key string, value []byte) error {
    // Just write locally, don't replicate again
    return s.store.Set(key, value, 0)
}
```

---

## 📊 Performance Impact

### Current (No Replication)
- **Writes**: 96K ops/sec
- **Reads**: 120K ops/sec
- **Consistency**: None
- **Durability**: Single node only

### With Async Replication
- **Writes**: ~85-90K ops/sec (slight overhead for async dispatch)
- **Reads**: 120K ops/sec (unchanged)
- **Consistency**: Eventual (replicas catch up within ms)
- **Durability**: 3x (data on primary + 2 replicas)

---

## 🎯 Implementation Plan

### Phase 1: Basic Async Replication (1-2 hours)
1. Add replication protocol commands
2. Implement `replicateAsync()` function
3. Handle replication writes on replicas
4. Test with 3-node cluster

### Phase 2: Read Optimization (30 min)
1. Allow reads from replicas (not just primary)
2. Add read preference (primary vs any replica)

### Phase 3: Failure Handling (1 hour)
1. Retry failed replications
2. Track replication lag
3. Handle replica catch-up after failure

### Phase 4: Benchmarking (30 min)
1. Update cluster benchmark to test replication
2. Measure replication lag
3. Compare with/without replication

---

## 🚀 Quick Fix for Benchmarks (5 min)

To make current benchmarks work without replication:

**Option A: Test Local Storage Only**
```go
// In cluster benchmark, each worker writes to same node it reads from
nodeIdx := workerID % 3  // Sticky assignment
clients[nodeIdx].Set(key, value)
clients[nodeIdx].Get(key)  // Read from same node
```

**Option B: Single Node Test**
```go
// Just test one node at a time
client := clients[0]  // Use only node 0
client.Set(key, value)
client.Get(key)
```

---

## 💡 Recommendation

**Implement async replication** - it's the right architecture for your use case:

1. ✅ Matches ClusterKit's design (async by default)
2. ✅ High performance (no blocking on replicas)
3. ✅ Simple to implement (just background goroutines)
4. ✅ Production-ready (eventual consistency is fine for KV store)

The implementation is straightforward and will give you true distributed storage with replication!

Want me to implement it?
