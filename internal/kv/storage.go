package kv

import (
	"encoding/binary"
	"errors"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrInvalidKey  = errors.New("invalid key")
	ErrClosed      = errors.New("storage closed")
)

type Storage struct {
	db       *badger.DB
	closed   atomic.Bool
	gcTicker *time.Ticker
	gcDone   chan struct{}
}

// NewKVStorage creates a new BadgerDB-backed KV storage
func NewKVStorage(path string) (*Storage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable logging for cleaner output

	// Extreme performance optimizations (tuned for 1M+ ops/sec)
	opts.NumVersionsToKeep = 1        // Keep only latest version
	opts.NumLevelZeroTables = 8       // Good balance (was 10)
	opts.NumLevelZeroTablesStall = 16 // Good balance (was 20)
	opts.ValueLogFileSize = 256 << 20 // 256MB (512MB might be too large)
	opts.NumCompactors = 4            // Good for 4-8 cores
	opts.ValueThreshold = 1024        // Store values > 1KB in value log
	opts.BlockCacheSize = 1 << 30     // 1GB block cache (2GB might be too much)
	opts.IndexCacheSize = 512 << 20   // 512MB index cache (1GB might be too much)
	opts.SyncWrites = false           // Async writes for speed
	opts.DetectConflicts = false      // Disable conflict detection for speed
	opts.CompactL0OnClose = false     // Skip compaction on close
	opts.MemTableSize = 128 << 20     // 128MB memtable
	opts.NumMemtables = 5             // More memtables for better caching

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	s := &Storage{
		db:     db,
		gcDone: make(chan struct{}),
	}

	// Start background garbage collection
	s.startGC()

	return s, nil
}

// startGC starts background value log garbage collection
func (s *Storage) startGC() {
	s.gcTicker = time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-s.gcTicker.C:
				// Run GC if 50% or more can be discarded
				err := s.db.RunValueLogGC(0.5)
				if err != nil && err != badger.ErrNoRewrite {
					// Log error if you have logging
					_ = err
				}
			case <-s.gcDone:
				return
			}
		}
	}()
}

// Close closes the BadgerDB connection
func (s *Storage) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return ErrClosed
	}

	// Stop GC
	if s.gcTicker != nil {
		s.gcTicker.Stop()
		close(s.gcDone)
	}

	return s.db.Close()
}

// Set stores a key-value pair with optional TTL
func (s *Storage) Set(key string, value []byte, ttl time.Duration) error {
	if s.closed.Load() {
		return ErrClosed
	}
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
	if s.closed.Load() {
		return nil, ErrClosed
	}
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

// Incr increments a numeric value and returns the new value
func (s *Storage) Incr(key string) (int64, error) {
	if s.closed.Load() {
		return 0, ErrClosed
	}
	if key == "" {
		return 0, ErrInvalidKey
	}

	var newValue int64

	err := s.db.Update(func(txn *badger.Txn) error {
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

		// Increment and store
		currentValue++
		newValue = currentValue

		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(currentValue))

		return txn.Set([]byte(key), buf[:])
	})

	return newValue, err
}

// Decr decrements a numeric value and returns the new value
func (s *Storage) Decr(key string) (int64, error) {
	if s.closed.Load() {
		return 0, ErrClosed
	}
	if key == "" {
		return 0, ErrInvalidKey
	}

	var newValue int64

	err := s.db.Update(func(txn *badger.Txn) error {
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

		// Decrement and store
		currentValue--
		newValue = currentValue

		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(currentValue))

		return txn.Set([]byte(key), buf[:])
	})

	return newValue, err
}

// Delete removes a key from the store
func (s *Storage) Delete(key string) error {
	if s.closed.Load() {
		return ErrClosed
	}
	if key == "" {
		return ErrInvalidKey
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exists checks if a key exists in the store
func (s *Storage) Exists(key string) (bool, error) {
	if s.closed.Load() {
		return false, ErrClosed
	}
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

// Scan retrieves all values with keys matching the given prefix with pagination
func (s *Storage) Scan(prefix string, limit int) ([][]byte, error) {
	if s.closed.Load() {
		return nil, ErrClosed
	}

	var values [][]byte
	if limit > 0 {
		values = make([][]byte, 0, limit) // Pre-allocate capacity
	}

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		opts.PrefetchSize = 100 // Prefetch for better performance

		it := txn.NewIterator(opts)
		defer it.Close()

		count := 0
		for it.Rewind(); it.Valid() && (limit <= 0 || count < limit); it.Next() {
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
			count++
		}

		return nil
	})

	return values, err
}

// ScanKeys retrieves all keys matching the given prefix with pagination
func (s *Storage) ScanKeys(prefix string, limit int) ([]string, error) {
	if s.closed.Load() {
		return nil, ErrClosed
	}

	var keys []string
	if limit > 0 {
		keys = make([]string, 0, limit)
	}

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		opts.PrefetchValues = false // Don't load values for keys-only scan

		it := txn.NewIterator(opts)
		defer it.Close()

		count := 0
		for it.Rewind(); it.Valid() && (limit <= 0 || count < limit); it.Next() {
			item := it.Item()
			key := string(item.Key())
			keys = append(keys, key)
			count++
		}

		return nil
	})

	return keys, err
}

// ScanKeysWithValues retrieves all keys and their values matching the given prefix
func (s *Storage) ScanKeysWithValues(prefix string, limit int) (map[string][]byte, error) {
	if s.closed.Load() {
		return nil, ErrClosed
	}

	kvPairs := make(map[string][]byte)
	if limit > 0 {
		kvPairs = make(map[string][]byte, limit)
	}

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		opts.PrefetchSize = 100 // Prefetch for better performance

		it := txn.NewIterator(opts)
		defer it.Close()

		count := 0
		for it.Rewind(); it.Valid() && (limit <= 0 || count < limit); it.Next() {
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
			count++
		}

		return nil
	})

	return kvPairs, err
}

// BatchSet stores multiple key-value pairs in a single transaction
func (s *Storage) BatchSet(kvPairs map[string][]byte, ttl time.Duration) error {
	if s.closed.Load() {
		return ErrClosed
	}
	if len(kvPairs) == 0 {
		return nil // No error for empty batch
	}

	// Validate all keys first
	for key := range kvPairs {
		if key == "" {
			return ErrInvalidKey
		}
	}

	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for key, value := range kvPairs {
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

// BatchSetNoTTL stores multiple key-value pairs without TTL (slightly faster)
func (s *Storage) BatchSetNoTTL(kvPairs map[string][]byte) error {
	if s.closed.Load() {
		return ErrClosed
	}
	if len(kvPairs) == 0 {
		return nil
	}

	// Validate all keys first
	for key := range kvPairs {
		if key == "" {
			return ErrInvalidKey
		}
	}

	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for key, value := range kvPairs {
		if err := wb.Set([]byte(key), value); err != nil {
			return err
		}
	}

	return wb.Flush()
}

// BatchGet retrieves multiple values by keys
func (s *Storage) BatchGet(keys []string) (map[string][]byte, error) {
	if s.closed.Load() {
		return nil, ErrClosed
	}
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Validate all keys first
	for _, key := range keys {
		if key == "" {
			return nil, ErrInvalidKey
		}
	}

	result := make(map[string][]byte, len(keys))

	// For small batches, single transaction is faster
	if len(keys) <= 100 {
		err := s.db.View(func(txn *badger.Txn) error {
			for _, key := range keys {
				item, err := txn.Get([]byte(key))
				if err != nil {
					if err == badger.ErrKeyNotFound {
						continue // Skip missing keys
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

	// For large batches, split into chunks
	chunkSize := 100
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize
		if end > len(keys) {
			end = len(keys)
		}

		err := s.db.View(func(txn *badger.Txn) error {
			for _, key := range keys[i:end] {
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

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// BatchDelete removes multiple keys in a single transaction
func (s *Storage) BatchDelete(keys []string) error {
	if s.closed.Load() {
		return ErrClosed
	}
	if len(keys) == 0 {
		return nil
	}

	// Validate all keys first
	for _, key := range keys {
		if key == "" {
			return ErrInvalidKey
		}
	}

	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for _, key := range keys {
		if err := wb.Delete([]byte(key)); err != nil {
			return err
		}
	}

	return wb.Flush()
}

// Sync forces a sync of all writes to disk
func (s *Storage) Sync() error {
	if s.closed.Load() {
		return ErrClosed
	}
	return s.db.Sync()
}

// RunGC manually triggers value log garbage collection
func (s *Storage) RunGC() error {
	if s.closed.Load() {
		return ErrClosed
	}
	return s.db.RunValueLogGC(0.5)
}
