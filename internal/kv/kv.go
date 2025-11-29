package kv

import (
	"time"

	"github.com/skshohagmiah/flin/internal/storage"
)

// StorageBackend defines the storage interface
type StorageBackend interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Exists(key string) (bool, error)
	Incr(key string) error
	Decr(key string) error
	Scan(prefix string) ([][]byte, error)
	ScanKeys(prefix string) ([]string, error)
	ScanKeysWithValues(prefix string) (map[string][]byte, error)
	BatchSet(kvPairs map[string][]byte, ttl time.Duration) error
	BatchGet(keys []string) (map[string][]byte, error)
	BatchDelete(keys []string) error
	Close() error
}

// KVStore is the developer-facing API for key-value operations
type KVStore struct {
	storage StorageBackend
}

// New creates a new KV store with 64-shard BadgerDB backend (default)
// This provides the best balance of performance and concurrency
// - 512K ops/sec throughput (2.6x vs non-sharded)
// - True parallelism across 64 independent shards
// - Hash-based key routing for even distribution
func New(path string) (*KVStore, error) {
	// Use 64-shard sharding by default for optimal performance
	store, err := storage.NewWithShards(path, 64)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		storage: store,
	}, nil
}

// NewWithShards creates a KV store with a custom shard count
// Use this to tune performance for your workload:
// - 1 shard: Sequential access, low memory (like single BadgerDB)
// - 16 shards: Balanced (347K ops/sec, 400MB)
// - 64 shards: High concurrency (512K ops/sec, 1.5GB) ← Recommended
// - 128+ shards: For very high concurrency workloads
func NewWithShards(path string, shardCount int) (*KVStore, error) {
	store, err := storage.NewWithShards(path, shardCount)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		storage: store,
	}, nil
}

// NewMemory creates an in-memory KV store using BadgerDB's in-memory mode
// Data is not persisted to disk - lost on shutdown
// Uses same performance optimization as disk version
// - Fast: No disk I/O
// - Sharded: Same 64-shard architecture
// - TTL Support: Automatic expiration of keys
func NewMemory() (*KVStore, error) {
	store, err := storage.NewMemoryWithShards(64)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		storage: store,
	}, nil
}

// NewMemoryWithShards creates an in-memory KV store with custom shard count
func NewMemoryWithShards(shardCount int) (*KVStore, error) {
	store, err := storage.NewMemoryWithShards(shardCount)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		storage: store,
	}, nil
}

// Close closes the underlying storage
func (k *KVStore) Close() error {
	return k.storage.Close()
}

// Set stores a key-value pair with optional TTL
func (k *KVStore) Set(key string, value []byte, ttl time.Duration) error {
	return k.storage.Set(key, value, ttl)
}

// Get retrieves a value by key
func (k *KVStore) Get(key string) ([]byte, error) {
	return k.storage.Get(key)
}

// Incr increments a numeric value stored at key
func (k *KVStore) Incr(key string) error {
	return k.storage.Incr(key)
}

// Decr decrements a numeric value stored at key
func (k *KVStore) Decr(key string) error {
	return k.storage.Decr(key)
}

// Delete removes a key from the store
func (k *KVStore) Delete(key string) error {
	return k.storage.Delete(key)
}

// Exists checks if a key exists in the store
func (k *KVStore) Exists(key string) (bool, error) {
	return k.storage.Exists(key)
}

// Scan retrieves all values with keys matching the given prefix
func (k *KVStore) Scan(prefix string) ([][]byte, error) {
	return k.storage.Scan(prefix)
}

// ScanKeys retrieves all keys matching the given prefix
func (k *KVStore) ScanKeys(prefix string) ([]string, error) {
	return k.storage.ScanKeys(prefix)
}

// ScanKeysWithValues retrieves all keys and values matching the given prefix
func (k *KVStore) ScanKeysWithValues(prefix string) (map[string][]byte, error) {
	return k.storage.ScanKeysWithValues(prefix)
}

// BatchSet stores multiple key-value pairs in a single transaction
func (k *KVStore) BatchSet(kvPairs map[string][]byte, ttl time.Duration) error {
	return k.storage.BatchSet(kvPairs, ttl)
}

// BatchGet retrieves multiple values by keys
func (k *KVStore) BatchGet(keys []string) (map[string][]byte, error) {
	return k.storage.BatchGet(keys)
}

// BatchDelete removes multiple keys in a single transaction
func (k *KVStore) BatchDelete(keys []string) error {
	return k.storage.BatchDelete(keys)
}
