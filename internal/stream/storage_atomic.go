package stream

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"
)

// getPartitionOffset returns the atomic offset counter for a topic:partition
func (s *StreamStorage) getPartitionOffset(topic string, partition int) *atomic.Int64 {
	key := fmt.Sprintf("%s:%d", topic, partition)
	s.offsetsMu.RLock()
	offset, exists := s.partitionOffsets[key]
	s.offsetsMu.RUnlock()

	if exists {
		return offset
	}

	// Create new atomic counter
	s.offsetsMu.Lock()
	defer s.offsetsMu.Unlock()

	// Double-check after acquiring write lock
	offset, exists = s.partitionOffsets[key]
	if exists {
		return offset
	}

	// Load current offset from storage
	var currentOffset int64 = -1
	s.db.View(func(txn *badger.Txn) error {
		offsetKey := makeOffsetKey(topic, partition)
		item, err := txn.Get([]byte(offsetKey))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			currentOffset = int64(binary.BigEndian.Uint64(val))
			return nil
		})
	})

	offset = &atomic.Int64{}
	offset.Store(currentOffset)
	s.partitionOffsets[key] = offset

	return offset
}

// flushWorker periodically flushes buffered writes to BadgerDB
func (s *StreamStorage) flushWorker() {
	for {
		select {
		case <-s.stopFlush:
			// Final flush before shutdown
			s.flushWrites()
			return
		case <-s.flushTicker.C:
			s.flushWrites()
		case <-s.flushCh:
			s.flushWrites()
		}
	}
}

// flushWrites writes buffered messages to BadgerDB in batch
func (s *StreamStorage) flushWrites() error {
	s.writeBufferMu.Lock()
	if len(s.writeBuffer) == 0 {
		s.writeBufferMu.Unlock()
		return nil
	}

	// Copy buffer and clear
	buffer := s.writeBuffer
	s.writeBuffer = make(map[string][]*Message)
	s.writeBufferMu.Unlock()

	// Batch write to BadgerDB
	return s.db.Update(func(txn *badger.Txn) error {
		for _, messages := range buffer {
			for _, msg := range messages {
				msgKey := makeMessageKey(msg.Topic, msg.Partition, msg.Offset)
				msgData := encodeMessage(msg)
				if err := txn.Set([]byte(msgKey), msgData); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
