package kv

import (
	"time"

	"github.com/skshohagmiah/flin/internal/storage"
)

// KVStore is the developer-facing API for key-value operations
type KVStore struct {
	storage *storage.Storage
}

// New creates a new KV store with BadgerDB backend at the specified path
func New(path string) (*KVStore, error) {
	store, err := storage.NewKVStorage(path)
	if err != nil {
		return nil, err
	}

	return &KVStore{storage: store}, nil
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
