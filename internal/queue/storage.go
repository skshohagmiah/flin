package queue

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrQueueEmpty      = errors.New("queue is empty")
	ErrInvalidQueue    = errors.New("invalid queue name")
	ErrMessageNotFound = errors.New("message not found")
	ErrMessageExpired  = errors.New("message visibility timeout expired")
)

// MessageState represents the current state of a message
type MessageState int

const (
	StatePending MessageState = iota
	StateInFlight
	StateAcked
	StateNacked
	StateDeadLetter
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

// QueueMessage stores message data and metadata for RabbitMQ-like behavior
type QueueMessage struct {
	ID                string
	Value             []byte
	State             MessageState
	DeliveryCount     int
	ConsumerID        string
	EnqueuedAt        int64
	InFlightAt        int64
	VisibilityTimeout int64 // Duration in milliseconds
	MaxRetries        int
}

// QueueConfig stores configuration for a queue
type QueueConfig struct {
	Name                     string
	DeadLetterQueue          string
	MaxRetries               int
	DefaultVisibilityTimeout int64 // milliseconds
}

// NewStorage creates a new BadgerDB-backed queue storage
func NewStorage(path string) (*QueueStorage, error) {
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

// metadataKey returns the key for storing queue metadata (legacy, kept for backward compatibility)
func metadataKey(queueName string) []byte {
	return []byte(fmt.Sprintf("queue:meta:%s", queueName))
}

// dataKey returns the key for storing a queue item (legacy, kept for backward compatibility)
func dataKey(queueName string, seqID uint64) []byte {
	return []byte(fmt.Sprintf("queue:data:%s:%020d", queueName, seqID))
}

// messageKey returns the key for storing a message by ID
func messageKey(queueName, messageID string) []byte {
	return []byte(fmt.Sprintf("queue:msg:%s:%s", queueName, messageID))
}

// messageIndexKey returns the key for indexing messages by state
func messageIndexKey(queueName string, state MessageState, messageID string) []byte {
	return []byte(fmt.Sprintf("queue:idx:%s:%d:%s", queueName, state, messageID))
}

// queueConfigKey returns the key for storing queue configuration
func queueConfigKey(queueName string) []byte {
	return []byte(fmt.Sprintf("queue:config:%s", queueName))
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

// encodeQueueMessage serializes a QueueMessage to bytes
func encodeQueueMessage(msg *QueueMessage) []byte {
	idLen := len(msg.ID)
	valueLen := len(msg.Value)
	consumerIDLen := len(msg.ConsumerID)

	// Format: [2:idLen][id][4:valueLen][value][1:state][4:deliveryCount][2:consumerIDLen][consumerID][8:enqueuedAt][8:inFlightAt][8:visibilityTimeout][4:maxRetries]
	size := 2 + idLen + 4 + valueLen + 1 + 4 + 2 + consumerIDLen + 8 + 8 + 8 + 4
	data := make([]byte, size)
	pos := 0

	binary.BigEndian.PutUint16(data[pos:], uint16(idLen))
	pos += 2
	copy(data[pos:], msg.ID)
	pos += idLen

	binary.BigEndian.PutUint32(data[pos:], uint32(valueLen))
	pos += 4
	copy(data[pos:], msg.Value)
	pos += valueLen

	data[pos] = byte(msg.State)
	pos++

	binary.BigEndian.PutUint32(data[pos:], uint32(msg.DeliveryCount))
	pos += 4

	binary.BigEndian.PutUint16(data[pos:], uint16(consumerIDLen))
	pos += 2
	copy(data[pos:], msg.ConsumerID)
	pos += consumerIDLen

	binary.BigEndian.PutUint64(data[pos:], uint64(msg.EnqueuedAt))
	pos += 8
	binary.BigEndian.PutUint64(data[pos:], uint64(msg.InFlightAt))
	pos += 8
	binary.BigEndian.PutUint64(data[pos:], uint64(msg.VisibilityTimeout))
	pos += 8
	binary.BigEndian.PutUint32(data[pos:], uint32(msg.MaxRetries))

	return data
}

// decodeQueueMessage deserializes bytes to a QueueMessage
func decodeQueueMessage(data []byte) (*QueueMessage, error) {
	if len(data) < 39 { // Minimum size
		return nil, fmt.Errorf("invalid message data: too short")
	}

	msg := &QueueMessage{}
	pos := 0

	idLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if len(data) < pos+idLen {
		return nil, fmt.Errorf("invalid message data: id length overflow")
	}
	msg.ID = string(data[pos : pos+idLen])
	pos += idLen

	valueLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4
	if len(data) < pos+valueLen {
		return nil, fmt.Errorf("invalid message data: value length overflow")
	}
	msg.Value = make([]byte, valueLen)
	copy(msg.Value, data[pos:pos+valueLen])
	pos += valueLen

	msg.State = MessageState(data[pos])
	pos++

	msg.DeliveryCount = int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4

	consumerIDLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if len(data) < pos+consumerIDLen {
		return nil, fmt.Errorf("invalid message data: consumerID length overflow")
	}
	msg.ConsumerID = string(data[pos : pos+consumerIDLen])
	pos += consumerIDLen

	msg.EnqueuedAt = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8
	msg.InFlightAt = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8
	msg.VisibilityTimeout = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8
	msg.MaxRetries = int(binary.BigEndian.Uint32(data[pos:]))

	return msg, nil
}

// encodeQueueConfig serializes a QueueConfig to bytes
func encodeQueueConfig(cfg *QueueConfig) []byte {
	nameLen := len(cfg.Name)
	dlqLen := len(cfg.DeadLetterQueue)

	// Format: [2:nameLen][name][2:dlqLen][dlq][4:maxRetries][8:defaultVisibilityTimeout]
	size := 2 + nameLen + 2 + dlqLen + 4 + 8
	data := make([]byte, size)
	pos := 0

	binary.BigEndian.PutUint16(data[pos:], uint16(nameLen))
	pos += 2
	copy(data[pos:], cfg.Name)
	pos += nameLen

	binary.BigEndian.PutUint16(data[pos:], uint16(dlqLen))
	pos += 2
	copy(data[pos:], cfg.DeadLetterQueue)
	pos += dlqLen

	binary.BigEndian.PutUint32(data[pos:], uint32(cfg.MaxRetries))
	pos += 4
	binary.BigEndian.PutUint64(data[pos:], uint64(cfg.DefaultVisibilityTimeout))

	return data
}

// decodeQueueConfig deserializes bytes to a QueueConfig
func decodeQueueConfig(data []byte) (*QueueConfig, error) {
	if len(data) < 16 { // Minimum size
		return nil, fmt.Errorf("invalid config data: too short")
	}

	cfg := &QueueConfig{}
	pos := 0

	nameLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if len(data) < pos+nameLen {
		return nil, fmt.Errorf("invalid config data: name length overflow")
	}
	cfg.Name = string(data[pos : pos+nameLen])
	pos += nameLen

	dlqLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if len(data) < pos+dlqLen {
		return nil, fmt.Errorf("invalid config data: dlq length overflow")
	}
	cfg.DeadLetterQueue = string(data[pos : pos+dlqLen])
	pos += dlqLen

	cfg.MaxRetries = int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4
	cfg.DefaultVisibilityTimeout = int64(binary.BigEndian.Uint64(data[pos:]))

	return cfg, nil
}
