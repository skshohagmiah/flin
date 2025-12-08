package queue

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

// SetQueueConfig stores queue configuration
func (q *QueueStorage) SetQueueConfig(cfg *QueueConfig) error {
	if cfg.Name == "" {
		return ErrInvalidQueue
	}

	return q.db.Update(func(txn *badger.Txn) error {
		key := queueConfigKey(cfg.Name)
		data := encodeQueueConfig(cfg)
		return txn.Set(key, data)
	})
}

// GetQueueConfig retrieves queue configuration
func (q *QueueStorage) GetQueueConfig(queueName string) (*QueueConfig, error) {
	if queueName == "" {
		return nil, ErrInvalidQueue
	}

	var cfg *QueueConfig
	err := q.db.View(func(txn *badger.Txn) error {
		key := queueConfigKey(queueName)
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			// Return default config
			cfg = &QueueConfig{
				Name:                     queueName,
				MaxRetries:               3,
				DefaultVisibilityTimeout: 30000, // 30 seconds
			}
			return nil
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			cfg, err = decodeQueueConfig(val)
			return err
		})
	})

	return cfg, err
}

// PushMessage adds a message to the queue with RabbitMQ-like behavior
func (q *QueueStorage) PushMessage(queueName string, value []byte) (string, error) {
	if queueName == "" {
		return "", ErrInvalidQueue
	}

	messageID := uuid.New().String()
	msg := &QueueMessage{
		ID:                messageID,
		Value:             value,
		State:             StatePending,
		DeliveryCount:     0,
		ConsumerID:        "",
		EnqueuedAt:        time.Now().UnixMilli(),
		InFlightAt:        0,
		VisibilityTimeout: 0,
		MaxRetries:        0,
	}

	return messageID, q.db.Update(func(txn *badger.Txn) error {
		// Store message
		msgKey := messageKey(queueName, messageID)
		msgData := encodeQueueMessage(msg)
		if err := txn.Set(msgKey, msgData); err != nil {
			return err
		}

		// Add to pending index
		idxKey := messageIndexKey(queueName, StatePending, messageID)
		return txn.Set(idxKey, []byte{1})
	})
}

// PopMessage retrieves a message and marks it as in-flight
func (q *QueueStorage) PopMessage(queueName, consumerID string, visibilityTimeout time.Duration) (*QueueMessage, error) {
	if queueName == "" {
		return nil, ErrInvalidQueue
	}

	var msg *QueueMessage

	err := q.db.Update(func(txn *badger.Txn) error {
		// Find first pending message
		prefix := []byte(fmt.Sprintf("queue:idx:%s:%d:", queueName, StatePending))
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Rewind()
		if !it.Valid() {
			return ErrQueueEmpty
		}

		// Extract message ID from index key
		indexKey := string(it.Item().Key())
		parts := []byte(indexKey)
		// Format: queue:idx:{queueName}:{state}:{messageID}
		messageID := string(parts[len(prefix):])

		// Get message
		msgKey := messageKey(queueName, messageID)
		item, err := txn.Get(msgKey)
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			msg, err = decodeQueueMessage(val)
			return err
		})
		if err != nil {
			return err
		}

		// Update message state
		msg.State = StateInFlight
		msg.ConsumerID = consumerID
		msg.InFlightAt = time.Now().UnixMilli()
		msg.VisibilityTimeout = visibilityTimeout.Milliseconds()
		msg.DeliveryCount++

		// Save updated message
		msgData := encodeQueueMessage(msg)
		if err := txn.Set(msgKey, msgData); err != nil {
			return err
		}

		// Update index: remove from pending, add to in-flight
		oldIdxKey := messageIndexKey(queueName, StatePending, messageID)
		if err := txn.Delete(oldIdxKey); err != nil {
			return err
		}

		newIdxKey := messageIndexKey(queueName, StateInFlight, messageID)
		return txn.Set(newIdxKey, []byte{1})
	})

	return msg, err
}

// AckMessage acknowledges and deletes a message
func (q *QueueStorage) AckMessage(queueName, messageID string) error {
	if queueName == "" || messageID == "" {
		return ErrInvalidQueue
	}

	return q.db.Update(func(txn *badger.Txn) error {
		// Get message to check state
		msgKey := messageKey(queueName, messageID)
		item, err := txn.Get(msgKey)
		if err == badger.ErrKeyNotFound {
			return ErrMessageNotFound
		}
		if err != nil {
			return err
		}

		var msg *QueueMessage
		err = item.Value(func(val []byte) error {
			msg, err = decodeQueueMessage(val)
			return err
		})
		if err != nil {
			return err
		}

		// Delete message
		if err := txn.Delete(msgKey); err != nil {
			return err
		}

		// Remove from index
		idxKey := messageIndexKey(queueName, msg.State, messageID)
		return txn.Delete(idxKey)
	})
}

// NackMessage rejects a message and requeues it or sends to DLQ
func (q *QueueStorage) NackMessage(queueName, messageID string, requeue bool) error {
	if queueName == "" || messageID == "" {
		return ErrInvalidQueue
	}

	return q.db.Update(func(txn *badger.Txn) error {
		// Get message
		msgKey := messageKey(queueName, messageID)
		item, err := txn.Get(msgKey)
		if err == badger.ErrKeyNotFound {
			return ErrMessageNotFound
		}
		if err != nil {
			return err
		}

		var msg *QueueMessage
		err = item.Value(func(val []byte) error {
			msg, err = decodeQueueMessage(val)
			return err
		})
		if err != nil {
			return err
		}

		// Get queue config to check max retries
		cfg, err := q.GetQueueConfig(queueName)
		if err != nil {
			return err
		}

		// Check if max retries exceeded
		if msg.DeliveryCount >= cfg.MaxRetries && cfg.MaxRetries > 0 {
			// Move to dead letter queue
			if cfg.DeadLetterQueue != "" {
				return q.moveToDeadLetter(txn, queueName, cfg.DeadLetterQueue, msg)
			}
			// No DLQ configured, just delete
			if err := txn.Delete(msgKey); err != nil {
				return err
			}
			idxKey := messageIndexKey(queueName, msg.State, messageID)
			return txn.Delete(idxKey)
		}

		if !requeue {
			// Delete message
			if err := txn.Delete(msgKey); err != nil {
				return err
			}
			idxKey := messageIndexKey(queueName, msg.State, messageID)
			return txn.Delete(idxKey)
		}

		// Requeue: change state back to pending
		oldState := msg.State
		msg.State = StatePending
		msg.ConsumerID = ""
		msg.InFlightAt = 0

		// Save updated message
		msgData := encodeQueueMessage(msg)
		if err := txn.Set(msgKey, msgData); err != nil {
			return err
		}

		// Update index
		oldIdxKey := messageIndexKey(queueName, oldState, messageID)
		if err := txn.Delete(oldIdxKey); err != nil {
			return err
		}

		newIdxKey := messageIndexKey(queueName, StatePending, messageID)
		return txn.Set(newIdxKey, []byte{1})
	})
}

// moveToDeadLetter moves a message to the dead letter queue
func (q *QueueStorage) moveToDeadLetter(txn *badger.Txn, sourceQueue, dlqName string, msg *QueueMessage) error {
	// Delete from source queue
	srcMsgKey := messageKey(sourceQueue, msg.ID)
	if err := txn.Delete(srcMsgKey); err != nil {
		return err
	}
	srcIdxKey := messageIndexKey(sourceQueue, msg.State, msg.ID)
	if err := txn.Delete(srcIdxKey); err != nil {
		return err
	}

	// Add to DLQ with new ID
	dlqMessageID := uuid.New().String()
	dlqMsg := &QueueMessage{
		ID:                dlqMessageID,
		Value:             msg.Value,
		State:             StatePending, // Use Pending so it can be popped
		DeliveryCount:     msg.DeliveryCount,
		ConsumerID:        "",
		EnqueuedAt:        time.Now().UnixMilli(),
		InFlightAt:        0,
		VisibilityTimeout: 0,
		MaxRetries:        0,
	}

	dlqMsgKey := messageKey(dlqName, dlqMessageID)
	dlqMsgData := encodeQueueMessage(dlqMsg)
	if err := txn.Set(dlqMsgKey, dlqMsgData); err != nil {
		return err
	}

	dlqIdxKey := messageIndexKey(dlqName, StatePending, dlqMessageID)
	return txn.Set(dlqIdxKey, []byte{1})
}

// RequeueExpiredMessages finds and requeues messages whose visibility timeout has expired
func (q *QueueStorage) RequeueExpiredMessages(queueName string) (int, error) {
	if queueName == "" {
		return 0, ErrInvalidQueue
	}

	count := 0
	now := time.Now().UnixMilli()

	err := q.db.Update(func(txn *badger.Txn) error {
		// Find all in-flight messages
		prefix := []byte(fmt.Sprintf("queue:idx:%s:%d:", queueName, StateInFlight))
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := txn.NewIterator(opts)
		defer it.Close()

		expiredMessages := []string{}

		for it.Rewind(); it.Valid(); it.Next() {
			indexKey := string(it.Item().Key())
			messageID := string([]byte(indexKey)[len(prefix):])

			// Get message
			msgKey := messageKey(queueName, messageID)
			item, err := txn.Get(msgKey)
			if err != nil {
				continue
			}

			var msg *QueueMessage
			err = item.Value(func(val []byte) error {
				msg, err = decodeQueueMessage(val)
				return err
			})
			if err != nil {
				continue
			}

			// Check if expired
			if msg.VisibilityTimeout > 0 && (msg.InFlightAt+msg.VisibilityTimeout) < now {
				expiredMessages = append(expiredMessages, messageID)
			}
		}

		// Requeue expired messages
		for _, messageID := range expiredMessages {
			msgKey := messageKey(queueName, messageID)
			item, err := txn.Get(msgKey)
			if err != nil {
				continue
			}

			var msg *QueueMessage
			err = item.Value(func(val []byte) error {
				msg, err = decodeQueueMessage(val)
				return err
			})
			if err != nil {
				continue
			}

			// Update state
			msg.State = StatePending
			msg.ConsumerID = ""
			msg.InFlightAt = 0

			// Save
			msgData := encodeQueueMessage(msg)
			if err := txn.Set(msgKey, msgData); err != nil {
				continue
			}

			// Update index
			oldIdxKey := messageIndexKey(queueName, StateInFlight, messageID)
			txn.Delete(oldIdxKey)

			newIdxKey := messageIndexKey(queueName, StatePending, messageID)
			if err := txn.Set(newIdxKey, []byte{1}); err == nil {
				count++
			}
		}

		return nil
	})

	return count, err
}
