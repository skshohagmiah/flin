package storage

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// StreamStorage provides persistent storage for stream messages using BadgerDB
// Key format:
//   - Messages: stream:msg:{topic}:{partition}:{offset}
//   - Offsets: stream:offset:{topic}:{partition}
//   - Consumer offsets: stream:consumer:{group}:{topic}:{partition}
//   - Topic metadata: stream:meta:{topic}
type StreamStorage struct {
	db *badger.DB

	// Per-partition locks to reduce contention
	// Key: "topic:partition"
	partitionLocks map[string]*sync.RWMutex
	partitionMu    sync.Mutex // Only lock for partition map access
}

// Message represents a stream message
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       string
	Value     []byte
	Timestamp int64
}

// TopicMetadata stores topic configuration
type TopicMetadata struct {
	Name           string
	Partitions     int
	RetentionMs    int64 // Retention time in milliseconds
	RetentionBytes int64 // Max bytes per partition
	CreatedAt      int64
}

// ConsumerOffset tracks consumer group progress
type ConsumerOffset struct {
	Group     string
	Topic     string
	Partition int
	Offset    int64
	UpdatedAt int64
}

// NewStreamStorage creates a new stream storage instance
func NewStreamStorage(path string) (*StreamStorage, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable badger logging

	// Optimize for high throughput scenarios
	opts.NumVersionsToKeep = 1   // Only keep 1 version (no MVCC overhead)
	opts.CompactL0OnClose = true // Compact L0 on close to improve reads
	opts.NumMemtables = 5        // More memtables for concurrent writes
	opts.MemTableSize = 64 << 20 // 64MB memtables

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger: %w", err)
	}

	return &StreamStorage{
		db:             db,
		partitionLocks: make(map[string]*sync.RWMutex),
	}, nil
}

// getPartitionLock returns the lock for a topic:partition pair
func (s *StreamStorage) getPartitionLock(topic string, partition int) *sync.RWMutex {
	key := fmt.Sprintf("%s:%d", topic, partition)
	s.partitionMu.Lock()
	defer s.partitionMu.Unlock()

	lock, exists := s.partitionLocks[key]
	if !exists {
		lock = &sync.RWMutex{}
		s.partitionLocks[key] = lock
	}
	return lock
}

// AppendMessage appends a message to a topic partition and returns the assigned offset
func (s *StreamStorage) AppendMessage(topic string, partition int, key string, value []byte) (int64, error) {
	// Use per-partition lock instead of global lock
	partLock := s.getPartitionLock(topic, partition)
	partLock.Lock()
	defer partLock.Unlock()

	var offset int64
	err := s.db.Update(func(txn *badger.Txn) error {
		// Get current offset for this partition
		offsetKey := makeOffsetKey(topic, partition)
		item, err := txn.Get([]byte(offsetKey))
		if err == badger.ErrKeyNotFound {
			offset = 0
		} else if err != nil {
			return err
		} else {
			err = item.Value(func(val []byte) error {
				offset = int64(binary.BigEndian.Uint64(val)) + 1
				return nil
			})
			if err != nil {
				return err
			}
		}

		// Store the message
		msgKey := makeMessageKey(topic, partition, offset)
		msg := &Message{
			Topic:     topic,
			Partition: partition,
			Offset:    offset,
			Key:       key,
			Value:     value,
			Timestamp: time.Now().UnixMilli(),
		}
		msgData := encodeMessage(msg)
		if err := txn.Set([]byte(msgKey), msgData); err != nil {
			return err
		}

		// Update offset
		offsetData := make([]byte, 8)
		binary.BigEndian.PutUint64(offsetData, uint64(offset))
		return txn.Set([]byte(offsetKey), offsetData)
	})

	return offset, err
}

// FetchMessages retrieves messages from a topic partition starting at the given offset
func (s *StreamStorage) FetchMessages(topic string, partition int, startOffset int64, maxCount int) ([]*Message, error) {
	// Use per-partition lock instead of global lock
	partLock := s.getPartitionLock(topic, partition)
	partLock.RLock()
	defer partLock.RUnlock()

	messages := make([]*Message, 0, maxCount)

	err := s.db.View(func(txn *badger.Txn) error {
		// Create iterator with prefix for this topic/partition
		prefix := makeMessagePrefix(topic, partition)
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		// Seek to start offset
		seekKey := makeMessageKey(topic, partition, startOffset)
		count := 0

		for it.Seek([]byte(seekKey)); it.ValidForPrefix([]byte(prefix)) && count < maxCount; it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				msg, err := decodeMessage(val)
				if err != nil {
					return err
				}
				messages = append(messages, msg)
				count++
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

// GetOffset returns the current offset for a topic partition
func (s *StreamStorage) GetOffset(topic string, partition int) (int64, error) {
	// Use per-partition lock instead of global lock
	partLock := s.getPartitionLock(topic, partition)
	partLock.RLock()
	defer partLock.RUnlock()

	var offset int64 = -1
	err := s.db.View(func(txn *badger.Txn) error {
		offsetKey := makeOffsetKey(topic, partition)
		item, err := txn.Get([]byte(offsetKey))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			offset = int64(binary.BigEndian.Uint64(val))
			return nil
		})
	})
	return offset, err
}

// CommitOffset stores the consumer group's offset for a topic partition
func (s *StreamStorage) CommitOffset(group, topic string, partition int, offset int64) error {
	// Use per-partition lock instead of global lock
	partLock := s.getPartitionLock(topic, partition)
	partLock.Lock()
	defer partLock.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		key := makeConsumerOffsetKey(group, topic, partition)
		data := make([]byte, 16) // 8 bytes offset + 8 bytes timestamp
		binary.BigEndian.PutUint64(data[0:8], uint64(offset))
		binary.BigEndian.PutUint64(data[8:16], uint64(time.Now().UnixMilli()))
		return txn.Set([]byte(key), data)
	})
}

// GetConsumerOffset retrieves the consumer group's offset for a topic partition
func (s *StreamStorage) GetConsumerOffset(group, topic string, partition int) (int64, error) {
	// Use per-partition lock instead of global lock
	partLock := s.getPartitionLock(topic, partition)
	partLock.RLock()
	defer partLock.RUnlock()

	var offset int64 = 0
	err := s.db.View(func(txn *badger.Txn) error {
		key := makeConsumerOffsetKey(group, topic, partition)
		item, err := txn.Get([]byte(key))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			offset = int64(binary.BigEndian.Uint64(val[0:8]))
			return nil
		})
	})
	return offset, err
}

// CreateTopic stores topic metadata
func (s *StreamStorage) CreateTopic(meta *TopicMetadata) error {
	// No partition-specific lock needed for topic metadata
	// (topic creation is infrequent)
	return s.db.Update(func(txn *badger.Txn) error {
		key := makeTopicMetaKey(meta.Name)
		data := encodeTopicMetadata(meta)
		return txn.Set([]byte(key), data)
	})
}

// GetTopicMetadata retrieves topic metadata
func (s *StreamStorage) GetTopicMetadata(topic string) (*TopicMetadata, error) {
	// No partition-specific lock needed for topic metadata
	// (reading metadata is cheap and infrequent)
	var meta *TopicMetadata
	err := s.db.View(func(txn *badger.Txn) error {
		key := makeTopicMetaKey(topic)
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			meta, err = decodeTopicMetadata(val)
			return err
		})
	})
	return meta, err
}

// DeleteOldMessages removes messages older than retention policy
func (s *StreamStorage) DeleteOldMessages(topic string, partition int, retentionMs int64) (int, error) {
	// Use per-partition lock instead of global lock
	partLock := s.getPartitionLock(topic, partition)
	partLock.Lock()
	defer partLock.Unlock()

	cutoffTime := time.Now().UnixMilli() - retentionMs
	deleted := 0

	err := s.db.Update(func(txn *badger.Txn) error {
		prefix := makeMessagePrefix(topic, partition)
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefix)
		it := txn.NewIterator(opts)
		defer it.Close()

		keysToDelete := make([][]byte, 0)

		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				msg, err := decodeMessage(val)
				if err != nil {
					return err
				}
				if msg.Timestamp < cutoffTime {
					keysToDelete = append(keysToDelete, item.KeyCopy(nil))
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		// Delete old messages
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})

	return deleted, err
}

// Close closes the storage
func (s *StreamStorage) Close() error {
	return s.db.Close()
}

// Key generation helpers
func makeMessageKey(topic string, partition int, offset int64) string {
	return fmt.Sprintf("stream:msg:%s:%d:%020d", topic, partition, offset)
}

func makeMessagePrefix(topic string, partition int) string {
	return fmt.Sprintf("stream:msg:%s:%d:", topic, partition)
}

func makeOffsetKey(topic string, partition int) string {
	return fmt.Sprintf("stream:offset:%s:%d", topic, partition)
}

func makeConsumerOffsetKey(group, topic string, partition int) string {
	return fmt.Sprintf("stream:consumer:%s:%s:%d", group, topic, partition)
}

func makeTopicMetaKey(topic string) string {
	return fmt.Sprintf("stream:meta:%s", topic)
}

// Encoding/Decoding helpers
func encodeMessage(msg *Message) []byte {
	keyLen := len(msg.Key)
	valueLen := len(msg.Value)

	// Format: [8:offset][8:timestamp][2:keyLen][key][4:valueLen][value]
	size := 8 + 8 + 2 + keyLen + 4 + valueLen
	data := make([]byte, size)
	pos := 0

	binary.BigEndian.PutUint64(data[pos:], uint64(msg.Offset))
	pos += 8
	binary.BigEndian.PutUint64(data[pos:], uint64(msg.Timestamp))
	pos += 8
	binary.BigEndian.PutUint16(data[pos:], uint16(keyLen))
	pos += 2
	copy(data[pos:], msg.Key)
	pos += keyLen
	binary.BigEndian.PutUint32(data[pos:], uint32(valueLen))
	pos += 4
	copy(data[pos:], msg.Value)

	return data
}

func decodeMessage(data []byte) (*Message, error) {
	if len(data) < 18 {
		return nil, fmt.Errorf("invalid message data")
	}

	msg := &Message{}
	pos := 0

	msg.Offset = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8
	msg.Timestamp = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8

	keyLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if len(data) < pos+keyLen+4 {
		return nil, fmt.Errorf("invalid message data")
	}
	msg.Key = string(data[pos : pos+keyLen])
	pos += keyLen

	valueLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4
	if len(data) < pos+valueLen {
		return nil, fmt.Errorf("invalid message data")
	}
	msg.Value = data[pos : pos+valueLen]

	return msg, nil
}

func encodeTopicMetadata(meta *TopicMetadata) []byte {
	nameLen := len(meta.Name)
	// Format: [2:nameLen][name][4:partitions][8:retentionMs][8:retentionBytes][8:createdAt]
	size := 2 + nameLen + 4 + 8 + 8 + 8
	data := make([]byte, size)
	pos := 0

	binary.BigEndian.PutUint16(data[pos:], uint16(nameLen))
	pos += 2
	copy(data[pos:], meta.Name)
	pos += nameLen
	binary.BigEndian.PutUint32(data[pos:], uint32(meta.Partitions))
	pos += 4
	binary.BigEndian.PutUint64(data[pos:], uint64(meta.RetentionMs))
	pos += 8
	binary.BigEndian.PutUint64(data[pos:], uint64(meta.RetentionBytes))
	pos += 8
	binary.BigEndian.PutUint64(data[pos:], uint64(meta.CreatedAt))

	return data
}

func decodeTopicMetadata(data []byte) (*TopicMetadata, error) {
	if len(data) < 30 {
		return nil, fmt.Errorf("invalid topic metadata")
	}

	meta := &TopicMetadata{}
	pos := 0

	nameLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	if len(data) < pos+nameLen+28 {
		return nil, fmt.Errorf("invalid topic metadata")
	}
	meta.Name = string(data[pos : pos+nameLen])
	pos += nameLen
	meta.Partitions = int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4
	meta.RetentionMs = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8
	meta.RetentionBytes = int64(binary.BigEndian.Uint64(data[pos:]))
	pos += 8
	meta.CreatedAt = int64(binary.BigEndian.Uint64(data[pos:]))

	return meta, nil
}
