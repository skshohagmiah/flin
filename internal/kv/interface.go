package kv

import "time"

type KV interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Incr(key string) error
	Decr(key string) error
	Delete(key string) error
	Exists(key string) (bool, error)
	Scan(prefix string) ([][]byte, error)
	
	// Batch operations for better performance
	BatchSet(kvPairs map[string][]byte, ttl time.Duration) error
	BatchGet(keys []string) (map[string][]byte, error)
	BatchDelete(keys []string) error
}
