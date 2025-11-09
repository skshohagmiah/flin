package storage

import (
	"encoding/json"
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
)

// QueueStorage implements queue storage using BadgerDB v4
type QueueStorage struct {
	db *badger.DB
}

// QueueMessage represents a persisted queue message
type QueueMessage struct {
	ID         string            `json:"id"`
	QueueName  string            `json:"queue_name"`
	Body       []byte            `json:"body"`
	Headers    map[string]string `json:"headers"`
	Priority   int               `json:"priority"`
	Timestamp  int64             `json:"timestamp"`
	TTL        int64             `json:"ttl"`
	DeadLetter string            `json:"dead_letter"`
}

// NewQueueStorage creates a new BadgerDB storage for queues
func NewQueueStorage(path string) (*QueueStorage, error) {
	opts := badger.DefaultOptions(path)
	
	// Optimize for queue workload
	opts.Logger = nil
	opts.SyncWrites = false
	opts.NumVersionsToKeep = 1
	opts.CompactL0OnClose = true
	
	// Memory settings
	opts.MemTableSize = 64 << 20    // 64MB
	opts.BlockCacheSize = 256 << 20 // 256MB
	opts.IndexCacheSize = 128 << 20 // 128MB
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open queue storage: %w", err)
	}

	return &QueueStorage{db: db}, nil
}

// SaveMessage persists a message
func (s *QueueStorage) SaveMessage(msg *QueueMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	key := []byte(fmt.Sprintf("queue:%s:msg:%s", msg.QueueName, msg.ID))

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
}

// LoadMessages loads all messages for a queue
func (s *QueueStorage) LoadMessages(queueName string) ([]*QueueMessage, error) {
	var messages []*QueueMessage

	prefix := []byte(fmt.Sprintf("queue:%s:msg:", queueName))

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			
			err := item.Value(func(val []byte) error {
				var msg QueueMessage
				if err := json.Unmarshal(val, &msg); err != nil {
					return err
				}
				messages = append(messages, &msg)
				return nil
			})
			
			if err != nil {
				return err
			}
		}

		return nil
	})

	return messages, err
}

// DeleteMessage removes a message
func (s *QueueStorage) DeleteMessage(queueName, msgID string) error {
	key := []byte(fmt.Sprintf("queue:%s:msg:%s", queueName, msgID))
	
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// Close closes the database
func (s *QueueStorage) Close() error {
	return s.db.Close()
}
