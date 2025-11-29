package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrInvalidKey  = errors.New("invalid key")
)

// Storage represents a single BadgerDB storage instance
type Storage struct {
	db *badger.DB
}

// KVStorage represents a sharded KV store with 64 shards by default
// Each shard is independent, enabling true parallelism and eliminating lock contention
type KVStorage struct {
	shards []*Storage
	mu     []sync.RWMutex // Per-shard locks
	count  int
}

const (
	// DefaultShardCount is the default number of shards (one per typical worker)
	// With 64 shards, 64 concurrent workers each get their own shard with no contention
	DefaultShardCount = 64
)

// ============================================================================
// Public API: KVStorage (Sharded by default)
// ============================================================================

// New creates a new KV store with 64 shards by default for maximum parallelism
// Each shard is an independent BadgerDB instance
// Recommended for all use cases requiring concurrency (most applications)
func New(path string) (*KVStorage, error) {
	return NewWithShards(path, DefaultShardCount)
}

// NewWithShards creates a new KV store with configurable shard count
// shardCount: Number of independent storage shards (typically 1, 16, or 64)
// - 1: Sequential mode (same as non-sharded)
// - 16: Moderate concurrency (good balance of throughput and memory)
// - 64: Maximum concurrency (recommended for high-concurrency workloads)
func NewWithShards(path string, shardCount int) (*KVStorage, error) {
	if shardCount <= 0 || shardCount > 256 {
		return nil, fmt.Errorf("invalid shard count: %d (must be 1-256)", shardCount)
	}

	shards := make([]*Storage, shardCount)
	locks := make([]sync.RWMutex, shardCount)

	// Create independent storage instance for each shard
	for i := 0; i < shardCount; i++ {
		shardPath := fmt.Sprintf("%s/shard_%d", path, i)
		store, err := newStorage(shardPath, false)
		if err != nil {
			// Clean up previously created shards
			for j := 0; j < i; j++ {
				shards[j].Close()
			}
			return nil, fmt.Errorf("failed to create shard %d: %w", i, err)
		}
		shards[i] = store
	}

	return &KVStorage{
		shards: shards,
		mu:     locks,
		count:  shardCount,
	}, nil
}

// NewMemory creates a new in-memory KV store with 64 shards
// No persistence - data is lost on shutdown
// Perfect for caching, session storage, or testing
// Uses BadgerDB's in-memory mode with sharding for maximum speed
func NewMemory() (*KVStorage, error) {
	return NewMemoryWithShards(DefaultShardCount)
}

// NewMemoryWithShards creates a new in-memory KV store with configurable shards
func NewMemoryWithShards(shardCount int) (*KVStorage, error) {
	if shardCount <= 0 || shardCount > 256 {
		return nil, fmt.Errorf("invalid shard count: %d (must be 1-256)", shardCount)
	}

	shards := make([]*Storage, shardCount)
	locks := make([]sync.RWMutex, shardCount)

	// Create independent in-memory storage for each shard
	for i := 0; i < shardCount; i++ {
		store, err := newStorage("", true)
		if err != nil {
			// Clean up previously created shards
			for j := 0; j < i; j++ {
				shards[j].Close()
			}
			return nil, fmt.Errorf("failed to create in-memory shard %d: %w", i, err)
		}
		shards[i] = store
	}

	return &KVStorage{
		shards: shards,
		mu:     locks,
		count:  shardCount,
	}, nil
}

// getShard returns the shard index for a given key using consistent hashing
// Uses FNV-1a 32-bit hash for fast, deterministic routing
func (kv *KVStorage) getShard(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % kv.count
}

// Close closes all shard instances
func (kv *KVStorage) Close() error {
	var lastErr error
	for i := 0; i < kv.count; i++ {
		if err := kv.shards[i].Close(); err != nil {
			lastErr = err // Keep trying to close all shards
		}
	}
	return lastErr
}

// ============================================================================
// KVStorage Operations (automatically routed to appropriate shard)
// ============================================================================

// Set stores a key-value pair in the appropriate shard
func (kv *KVStorage) Set(key string, value []byte, ttl time.Duration) error {
	shardID := kv.getShard(key)
	kv.mu[shardID].Lock()
	defer kv.mu[shardID].Unlock()
	return kv.shards[shardID].Set(key, value, ttl)
}

// Get retrieves a value by key from the appropriate shard
func (kv *KVStorage) Get(key string) ([]byte, error) {
	shardID := kv.getShard(key)
	kv.mu[shardID].RLock()
	defer kv.mu[shardID].RUnlock()
	return kv.shards[shardID].Get(key)
}

// Delete removes a key from the appropriate shard
func (kv *KVStorage) Delete(key string) error {
	shardID := kv.getShard(key)
	kv.mu[shardID].Lock()
	defer kv.mu[shardID].Unlock()
	return kv.shards[shardID].Delete(key)
}

// Exists checks if a key exists in the appropriate shard
func (kv *KVStorage) Exists(key string) (bool, error) {
	shardID := kv.getShard(key)
	kv.mu[shardID].RLock()
	defer kv.mu[shardID].RUnlock()
	return kv.shards[shardID].Exists(key)
}

// Incr increments a numeric value in the appropriate shard
func (kv *KVStorage) Incr(key string) error {
	shardID := kv.getShard(key)
	kv.mu[shardID].Lock()
	defer kv.mu[shardID].Unlock()
	return kv.shards[shardID].Incr(key)
}

// Decr decrements a numeric value in the appropriate shard
func (kv *KVStorage) Decr(key string) error {
	shardID := kv.getShard(key)
	kv.mu[shardID].Lock()
	defer kv.mu[shardID].Unlock()
	return kv.shards[shardID].Decr(key)
}

// Scan retrieves all values matching a prefix from all shards concurrently
func (kv *KVStorage) Scan(prefix string) ([][]byte, error) {
	var allValues [][]byte
	errChan := make(chan error, kv.count)
	valuesChan := make(chan [][]byte, kv.count)

	for i := 0; i < kv.count; i++ {
		go func(shardID int) {
			kv.mu[shardID].RLock()
			defer kv.mu[shardID].RUnlock()
			values, err := kv.shards[shardID].Scan(prefix)
			if err != nil {
				errChan <- err
			} else {
				valuesChan <- values
			}
		}(i)
	}

	// Collect results
	for i := 0; i < kv.count; i++ {
		select {
		case err := <-errChan:
			return nil, err
		case values := <-valuesChan:
			allValues = append(allValues, values...)
		}
	}

	return allValues, nil
}

// ScanKeys retrieves all keys matching a prefix from all shards concurrently
func (kv *KVStorage) ScanKeys(prefix string) ([]string, error) {
	var allKeys []string
	errChan := make(chan error, kv.count)
	keysChan := make(chan []string, kv.count)

	for i := 0; i < kv.count; i++ {
		go func(shardID int) {
			kv.mu[shardID].RLock()
			defer kv.mu[shardID].RUnlock()
			keys, err := kv.shards[shardID].ScanKeys(prefix)
			if err != nil {
				errChan <- err
			} else {
				keysChan <- keys
			}
		}(i)
	}

	// Collect results
	for i := 0; i < kv.count; i++ {
		select {
		case err := <-errChan:
			return nil, err
		case keys := <-keysChan:
			allKeys = append(allKeys, keys...)
		}
	}

	return allKeys, nil
}

// ScanKeysWithValues retrieves all keys and values matching a prefix from all shards concurrently
func (kv *KVStorage) ScanKeysWithValues(prefix string) (map[string][]byte, error) {
	allKVs := make(map[string][]byte)
	mu := sync.Mutex{}
	errChan := make(chan error, kv.count)
	kvsChan := make(chan map[string][]byte, kv.count)

	for i := 0; i < kv.count; i++ {
		go func(shardID int) {
			kv.mu[shardID].RLock()
			defer kv.mu[shardID].RUnlock()
			kvs, err := kv.shards[shardID].ScanKeysWithValues(prefix)
			if err != nil {
				errChan <- err
			} else {
				kvsChan <- kvs
			}
		}(i)
	}

	// Collect results
	for i := 0; i < kv.count; i++ {
		select {
		case err := <-errChan:
			return nil, err
		case kvs := <-kvsChan:
			mu.Lock()
			for k, v := range kvs {
				allKVs[k] = v
			}
			mu.Unlock()
		}
	}

	return allKVs, nil
}

// BatchSet stores multiple key-value pairs, distributed across shards
// Intelligently groups keys by shard and processes in parallel
func (kv *KVStorage) BatchSet(kvPairs map[string][]byte, ttl time.Duration) error {
	// Group keys by shard
	shardedPairs := make([]map[string][]byte, kv.count)
	for i := 0; i < kv.count; i++ {
		shardedPairs[i] = make(map[string][]byte)
	}

	for key, value := range kvPairs {
		shardID := kv.getShard(key)
		shardedPairs[shardID][key] = value
	}

	// Set batches in parallel across shards
	errChan := make(chan error, kv.count)
	for i := 0; i < kv.count; i++ {
		go func(shardID int) {
			if len(shardedPairs[shardID]) == 0 {
				errChan <- nil
				return
			}
			kv.mu[shardID].Lock()
			defer kv.mu[shardID].Unlock()
			errChan <- kv.shards[shardID].BatchSet(shardedPairs[shardID], ttl)
		}(i)
	}

	// Collect errors
	for i := 0; i < kv.count; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}
	return nil
}

// BatchGet retrieves multiple values, distributed across shards
// Intelligently groups keys by shard and fetches in parallel
func (kv *KVStorage) BatchGet(keys []string) (map[string][]byte, error) {
	// Group keys by shard
	shardedKeys := make([][]string, kv.count)
	for i := 0; i < kv.count; i++ {
		shardedKeys[i] = make([]string, 0)
	}

	for _, key := range keys {
		shardID := kv.getShard(key)
		shardedKeys[shardID] = append(shardedKeys[shardID], key)
	}

	// Get values in parallel across shards
	results := make(chan map[string][]byte, kv.count)
	errChan := make(chan error, kv.count)

	for i := 0; i < kv.count; i++ {
		go func(shardID int) {
			if len(shardedKeys[shardID]) == 0 {
				results <- make(map[string][]byte)
				errChan <- nil
				return
			}
			kv.mu[shardID].RLock()
			defer kv.mu[shardID].RUnlock()
			kvs, err := kv.shards[shardID].BatchGet(shardedKeys[shardID])
			results <- kvs
			errChan <- err
		}(i)
	}

	// Collect results
	allResults := make(map[string][]byte)
	for i := 0; i < kv.count; i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
		for k, v := range <-results {
			allResults[k] = v
		}
	}

	return allResults, nil
}

// BatchDelete removes multiple keys, distributed across shards
func (kv *KVStorage) BatchDelete(keys []string) error {
	// Group keys by shard
	shardedKeys := make([][]string, kv.count)
	for i := 0; i < kv.count; i++ {
		shardedKeys[i] = make([]string, 0)
	}

	for _, key := range keys {
		shardID := kv.getShard(key)
		shardedKeys[shardID] = append(shardedKeys[shardID], key)
	}

	// Delete batches in parallel across shards
	errChan := make(chan error, kv.count)
	for i := 0; i < kv.count; i++ {
		go func(shardID int) {
			if len(shardedKeys[shardID]) == 0 {
				errChan <- nil
				return
			}
			kv.mu[shardID].Lock()
			defer kv.mu[shardID].Unlock()
			errChan <- kv.shards[shardID].BatchDelete(shardedKeys[shardID])
		}(i)
	}

	// Collect errors
	for i := 0; i < kv.count; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// Storage (Single shard implementation)
// ============================================================================

// newStorage creates a new single storage instance (used internally by sharding)
func newStorage(path string, inMemory bool) (*Storage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable logging for cleaner output
	opts.InMemory = inMemory

	// Extreme performance optimizations (tuned for 1M+ ops/sec)
	opts.NumVersionsToKeep = 1        // Keep only latest version
	opts.NumLevelZeroTables = 10      // More L0 tables before compaction (doubled)
	opts.NumLevelZeroTablesStall = 20 // Higher stall threshold
	opts.ValueLogFileSize = 512 << 20 // 512MB value log files (doubled)
	opts.NumCompactors = 4            // More compactors for parallel work (doubled)
	opts.ValueThreshold = 1024        // Store values > 1KB in value log
	opts.BlockCacheSize = 2 << 30     // 2GB block cache (4x increase)
	opts.IndexCacheSize = 1 << 30     // 1GB index cache (2x increase)
	opts.SyncWrites = false           // Async writes for speed
	opts.DetectConflicts = false      // Disable conflict detection for speed
	opts.CompactL0OnClose = false     // Skip compaction on close
	opts.MemTableSize = 128 << 20     // 128MB memtable (larger for more buffering)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

// Close closes the BadgerDB connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// Set stores a key-value pair with optional TTL
func (s *Storage) Set(key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value)
		if ttl > 0 {
			entry = entry.WithTTL(ttl)
		}
		return txn.SetEntry(entry)
	})
}

// Get retrieves a value by key
func (s *Storage) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrKeyNotFound
			}
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	return value, err
}

// Incr increments a numeric value stored at key
func (s *Storage) Incr(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		var currentValue int64 = 0

		item, err := txn.Get([]byte(key))
		if err == nil {
			err = item.Value(func(val []byte) error {
				if len(val) == 8 {
					currentValue = int64(binary.BigEndian.Uint64(val))
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		currentValue++
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(currentValue))

		return txn.Set([]byte(key), buf[:])
	})
}

// Decr decrements a numeric value stored at key
func (s *Storage) Decr(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		var currentValue int64 = 0

		item, err := txn.Get([]byte(key))
		if err == nil {
			err = item.Value(func(val []byte) error {
				if len(val) == 8 {
					currentValue = int64(binary.BigEndian.Uint64(val))
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		currentValue--
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(currentValue))

		return txn.Set([]byte(key), buf[:])
	})
}

// Delete removes a key from the store
func (s *Storage) Delete(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exists checks if a key exists in the store
func (s *Storage) Exists(key string) (bool, error) {
	if key == "" {
		return false, ErrInvalidKey
	}

	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// Scan retrieves all values with keys matching the given prefix
func (s *Storage) Scan(prefix string) ([][]byte, error) {
	var values [][]byte

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				valueCopy := make([]byte, len(val))
				copy(valueCopy, val)
				values = append(values, valueCopy)
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return values, err
}

// ScanKeys retrieves all keys matching the given prefix
func (s *Storage) ScanKeys(prefix string) ([]string, error) {
	var keys []string

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			keys = append(keys, key)
		}

		return nil
	})

	return keys, err
}

// ScanKeysWithValues retrieves all keys and their values matching the given prefix
func (s *Storage) ScanKeysWithValues(prefix string) (map[string][]byte, error) {
	kvPairs := make(map[string][]byte)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			err := item.Value(func(val []byte) error {
				valueCopy := make([]byte, len(val))
				copy(valueCopy, val)
				kvPairs[key] = valueCopy
				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return kvPairs, err
}

// BatchSet stores multiple key-value pairs in a single transaction
func (s *Storage) BatchSet(kvPairs map[string][]byte, ttl time.Duration) error {
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for key, value := range kvPairs {
		if key == "" {
			continue
		}
		entry := badger.NewEntry([]byte(key), value)
		if ttl > 0 {
			entry = entry.WithTTL(ttl)
		}
		if err := wb.SetEntry(entry); err != nil {
			return err
		}
	}

	return wb.Flush()
}

// BatchGet retrieves multiple values by keys
func (s *Storage) BatchGet(keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(keys))

	err := s.db.View(func(txn *badger.Txn) error {
		for _, key := range keys {
			if key == "" {
				continue
			}
			item, err := txn.Get([]byte(key))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					continue
				}
				return err
			}

			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			result[key] = value
		}
		return nil
	})

	return result, err
}

// BatchDelete removes multiple keys in a single transaction
func (s *Storage) BatchDelete(keys []string) error {
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for _, key := range keys {
		if key == "" {
			continue
		}
		if err := wb.Delete([]byte(key)); err != nil {
			return err
		}
	}

	return wb.Flush()
}
