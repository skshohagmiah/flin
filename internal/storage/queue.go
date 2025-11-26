package storage

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrQueueEmpty   = errors.New("queue is empty")
	ErrInvalidQueue = errors.New("invalid queue name")
)

// QueueStorage implements BadgerDB-backed queue storage
type QueueStorage struct {
	db *badger.DB
}

// QueueMetadata stores head and tail pointers for a queue
type QueueMetadata struct {
	Head uint64 // Next item to dequeue
	Tail uint64 // Next position to enqueue
}

// NewQueueStorage creates a new BadgerDB-backed queue storage
func NewQueueStorage(path string) (*QueueStorage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable logging for cleaner output

	// Performance optimizations for queue workloads
	opts.NumVersionsToKeep = 1
	opts.NumLevelZeroTables = 10
	opts.NumLevelZeroTablesStall = 20
	opts.ValueLogFileSize = 512 << 20
	opts.NumCompactors = 4
	opts.ValueThreshold = 1024
	opts.BlockCacheSize = 1 << 30   // 1GB block cache
	opts.IndexCacheSize = 512 << 20 // 512MB index cache
	opts.SyncWrites = false         // Async writes for speed
	opts.DetectConflicts = false
	opts.CompactL0OnClose = false
	opts.MemTableSize = 64 << 20 // 64MB memtable

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &QueueStorage{db: db}, nil
}

// Close closes the BadgerDB connection
func (q *QueueStorage) Close() error {
	return q.db.Close()
}

// metadataKey returns the key for storing queue metadata
func metadataKey(queueName string) []byte {
	return []byte(fmt.Sprintf("queue:meta:%s", queueName))
}

// dataKey returns the key for storing a queue item
func dataKey(queueName string, seqID uint64) []byte {
	return []byte(fmt.Sprintf("queue:data:%s:%020d", queueName, seqID))
}

// getMetadata retrieves the metadata for a queue
func (q *QueueStorage) getMetadata(txn *badger.Txn, queueName string) (*QueueMetadata, error) {
	key := metadataKey(queueName)

	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			// Queue doesn't exist yet, return empty metadata
			return &QueueMetadata{Head: 0, Tail: 0}, nil
		}
		return nil, err
	}

	var meta *QueueMetadata
	err = item.Value(func(val []byte) error {
		if len(val) != 16 {
			meta = &QueueMetadata{Head: 0, Tail: 0}
			return nil
		}

		meta = &QueueMetadata{
			Head: binary.BigEndian.Uint64(val[0:8]),
			Tail: binary.BigEndian.Uint64(val[8:16]),
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return meta, nil
}

// setMetadata stores the metadata for a queue
func (q *QueueStorage) setMetadata(txn *badger.Txn, queueName string, meta *QueueMetadata) error {
	key := metadataKey(queueName)
	data := make([]byte, 16)
	binary.BigEndian.PutUint64(data[0:8], meta.Head)
	binary.BigEndian.PutUint64(data[8:16], meta.Tail)

	return txn.Set(key, data)
}

// Push adds an item to the end of the queue
func (q *QueueStorage) Push(queueName string, value []byte) error {
	if queueName == "" {
		return ErrInvalidQueue
	}

	return q.db.Update(func(txn *badger.Txn) error {
		// Get current metadata
		meta, err := q.getMetadata(txn, queueName)
		if err != nil {
			return err
		}

		// Store the item
		itemKey := dataKey(queueName, meta.Tail)
		if err := txn.Set(itemKey, value); err != nil {
			return err
		}

		// Update metadata
		meta.Tail++
		return q.setMetadata(txn, queueName, meta)
	})
}

// Pop removes and returns the first item from the queue
func (q *QueueStorage) Pop(queueName string) ([]byte, error) {
	if queueName == "" {
		return nil, ErrInvalidQueue
	}

	var value []byte
	err := q.db.Update(func(txn *badger.Txn) error {
		// Get current metadata
		meta, err := q.getMetadata(txn, queueName)
		if err != nil {
			return err
		}

		// Check if queue is empty
		if meta.Head >= meta.Tail {
			return ErrQueueEmpty
		}

		// Get the item
		itemKey := dataKey(queueName, meta.Head)
		item, err := txn.Get(itemKey)
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		// Delete the item
		if err := txn.Delete(itemKey); err != nil {
			return err
		}

		// Update metadata
		meta.Head++
		return q.setMetadata(txn, queueName, meta)
	})

	return value, err
}

// Peek returns the first item without removing it
func (q *QueueStorage) Peek(queueName string) ([]byte, error) {
	if queueName == "" {
		return nil, ErrInvalidQueue
	}

	var value []byte
	err := q.db.View(func(txn *badger.Txn) error {
		// Get current metadata
		meta, err := q.getMetadata(txn, queueName)
		if err != nil {
			return err
		}

		// Check if queue is empty
		if meta.Head >= meta.Tail {
			return ErrQueueEmpty
		}

		// Get the item
		itemKey := dataKey(queueName, meta.Head)
		item, err := txn.Get(itemKey)
		if err != nil {
			return err
		}

		value, err = item.ValueCopy(nil)
		return err
	})

	return value, err
}

// Len returns the number of items in the queue
func (q *QueueStorage) Len(queueName string) (uint64, error) {
	if queueName == "" {
		return 0, ErrInvalidQueue
	}

	var length uint64
	err := q.db.View(func(txn *badger.Txn) error {
		meta, err := q.getMetadata(txn, queueName)
		if err != nil {
			return err
		}

		if meta.Tail > meta.Head {
			length = meta.Tail - meta.Head
		}
		return nil
	})

	return length, err
}

// Clear removes all items from the queue
func (q *QueueStorage) Clear(queueName string) error {
	if queueName == "" {
		return ErrInvalidQueue
	}

	return q.db.Update(func(txn *badger.Txn) error {
		// Get current metadata
		meta, err := q.getMetadata(txn, queueName)
		if err != nil {
			return err
		}

		// Delete all items
		for i := meta.Head; i < meta.Tail; i++ {
			itemKey := dataKey(queueName, i)
			if err := txn.Delete(itemKey); err != nil {
				// Continue even if delete fails
				continue
			}
		}

		// Reset metadata
		meta.Head = 0
		meta.Tail = 0
		return q.setMetadata(txn, queueName, meta)
	})
}
