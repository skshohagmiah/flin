package db

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

// DocStorage provides low-level BadgerDB operations for document storage
type DocStorage struct {
	db *badger.DB
}

// NewDocStorage creates a new document storage instance
func NewStorage(path string) (*DocStorage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil

	// Optimize for document storage
	opts.NumVersionsToKeep = 1
	opts.NumLevelZeroTables = 10
	opts.NumLevelZeroTablesStall = 20
	opts.ValueLogFileSize = 512 << 20 // 512MB
	opts.NumCompactors = 4
	opts.ValueThreshold = 1024
	opts.BlockCacheSize = 2 << 30 // 2GB
	opts.IndexCacheSize = 1 << 30 // 1GB
	opts.SyncWrites = false
	opts.DetectConflicts = false
	opts.CompactL0OnClose = false
	opts.MemTableSize = 128 << 20 // 128MB

	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	return &DocStorage{db: badgerDB}, nil
}

// Close closes the BadgerDB instance
func (ds *DocStorage) Close() error {
	return ds.db.Close()
}

// Set stores a document with the given key
func (ds *DocStorage) Set(key string, data []byte) error {
	return ds.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// Get retrieves a document by key
func (ds *DocStorage) Get(key string) ([]byte, error) {
	var data []byte

	err := ds.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		data, err = item.ValueCopy(nil)
		return err
	})

	return data, err
}

// Delete removes a document by key
func (ds *DocStorage) Delete(key string) error {
	return ds.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exists checks if a key exists
func (ds *DocStorage) Exists(key string) (bool, error) {
	err := ds.db.View(func(txn *badger.Txn) error {
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

// Scan iterates over all keys with the given prefix
func (ds *DocStorage) Scan(prefix string, fn func(key string, value []byte) error) error {
	return ds.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			data, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			if err := fn(key, data); err != nil {
				return err
			}
		}
		return nil
	})
}

// Count returns the number of keys with the given prefix
func (ds *DocStorage) Count(prefix string) (int64, error) {
	var count int64

	err := ds.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
			count++
		}
		return nil
	})

	return count, err
}

// BatchSet stores multiple key-value pairs atomically
func (ds *DocStorage) BatchSet(items map[string][]byte) error {
	return ds.db.Update(func(txn *badger.Txn) error {
		for key, data := range items {
			if err := txn.Set([]byte(key), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchDelete removes multiple keys atomically
func (ds *DocStorage) BatchDelete(keys []string) error {
	return ds.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			if err := txn.Delete([]byte(key)); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetJSON stores a JSON-serializable value
func (ds *DocStorage) SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return ds.Set(key, data)
}

// GetJSON retrieves and unmarshals a JSON value
func (ds *DocStorage) GetJSON(key string, dest interface{}) error {
	data, err := ds.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}
