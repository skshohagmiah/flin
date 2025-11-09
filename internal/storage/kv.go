package storage

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrInvalidKey  = errors.New("invalid key")
)

// Buffer pool for zero-allocation operations
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 8192) // 8KB buffers
	},
}

type Storage struct {
	db *badger.DB
}

// NewKVStorage creates a new BadgerDB-backed KV storage
func NewKVStorage(path string) (*Storage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable logging for cleaner output
	
	// Performance optimizations (tuned for 100K+ ops/sec)
	opts.NumVersionsToKeep = 1           // Keep only latest version
	opts.NumLevelZeroTables = 5          // More L0 tables before compaction
	opts.NumLevelZeroTablesStall = 15    // Higher stall threshold
	opts.ValueLogFileSize = 256 << 20    // 256MB value log files
	opts.NumCompactors = 2               // Balanced compaction
	opts.ValueThreshold = 1024           // Store values > 1KB in value log
	opts.BlockCacheSize = 512 << 20      // 512MB block cache (increased)
	opts.IndexCacheSize = 512 << 20      // 512MB index cache (increased)
	opts.SyncWrites = false              // Async writes for speed (less durable)

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

// Incr increments a numeric value stored at key (optimized with stack allocation)
func (s *Storage) Incr(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		var currentValue int64 = 0

		item, err := txn.Get([]byte(key))
		if err == nil {
			// Key exists, get current value
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

		// Increment and store (use stack allocation for small buffer)
		currentValue++
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(currentValue))

		return txn.Set([]byte(key), buf[:])
	})
}

// Decr decrements a numeric value stored at key (optimized with stack allocation)
func (s *Storage) Decr(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		var currentValue int64 = 0

		item, err := txn.Get([]byte(key))
		if err == nil {
			// Key exists, get current value
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

		// Decrement and store (use stack allocation for small buffer)
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
				// Make a copy of the value
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

// BatchSet stores multiple key-value pairs in a single transaction (optimized for throughput)
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

// BatchGet retrieves multiple values by keys (optimized with pre-allocation)
func (s *Storage) BatchGet(keys []string) (map[string][]byte, error) {
	// Pre-allocate result map with capacity
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

// BatchDelete removes multiple keys in a single transaction (optimized for throughput)
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
