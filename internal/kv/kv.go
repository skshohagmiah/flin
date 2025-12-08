package kv

import (
	"time"
)

// StorageBackend defines the storage interface
type StorageBackend interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Exists(key string) (bool, error)
	Incr(key string) (int64, error) // ✅ Now returns int64
	Decr(key string) (int64, error) // ✅ Now returns int64
	Scan(prefix string, limit int) ([][]byte, error)
	ScanKeys(prefix string, limit int) ([]string, error)
	ScanKeysWithValues(prefix string, limit int) (map[string][]byte, error)
	BatchSet(kvPairs map[string][]byte, ttl time.Duration) error
	BatchSetNoTTL(kvPairs map[string][]byte) error
	BatchGet(keys []string) (map[string][]byte, error)
	BatchDelete(keys []string) error
	Close() error
}

// KVStore is the developer-facing API for key-value operations
type KVStore struct {
	storage StorageBackend
}

// New creates a new KV store using BadgerDB backend
// Uses a single BadgerDB instance for maximum performance and simplicity
func New(path string) (*KVStore, error) {
	store, err := NewKVStorage(path)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		storage: store,
	}, nil
}

// NewWithShards is deprecated - shardCount parameter is ignored
// Use New() instead. Kept for backward compatibility.
func NewWithShards(path string, shardCount int) (*KVStore, error) {
	return New(path)
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

// Incr increments a numeric value and returns the new value
func (k *KVStore) Incr(key string) (int64, error) {
	return k.storage.Incr(key)
}

// Decr decrements a numeric value and returns the new value
func (k *KVStore) Decr(key string) (int64, error) {
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
// limit <= 0 means no limit
func (k *KVStore) Scan(prefix string, limit int) ([][]byte, error) {
	return k.storage.Scan(prefix, limit)
}

// ScanKeys retrieves all keys matching the given prefix
// limit <= 0 means no limit
func (k *KVStore) ScanKeys(prefix string, limit int) ([]string, error) {
	return k.storage.ScanKeys(prefix, limit)
}

// ScanKeysWithValues retrieves all keys and values matching the given prefix
// limit <= 0 means no limit
func (k *KVStore) ScanKeysWithValues(prefix string, limit int) (map[string][]byte, error) {
	return k.storage.ScanKeysWithValues(prefix, limit)
}

// BatchSet stores multiple key-value pairs in a single transaction with TTL
func (k *KVStore) BatchSet(kvPairs map[string][]byte, ttl time.Duration) error {
	return k.storage.BatchSet(kvPairs, ttl)
}

// BatchSetNoTTL stores multiple key-value pairs without TTL (faster than BatchSet with ttl=0)
func (k *KVStore) BatchSetNoTTL(kvPairs map[string][]byte) error {
	return k.storage.BatchSetNoTTL(kvPairs)
}

// BatchGet retrieves multiple values by keys
func (k *KVStore) BatchGet(keys []string) (map[string][]byte, error) {
	return k.storage.BatchGet(keys)
}

// BatchDelete removes multiple keys in a single transaction
func (k *KVStore) BatchDelete(keys []string) error {
	return k.storage.BatchDelete(keys)
}
