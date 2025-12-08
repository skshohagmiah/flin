package stream

import (
	"fmt"
	"time"
)

// AppendMessageBatch appends multiple messages atomically
func (s *StreamStorage) AppendMessageBatch(topic string, messages []*PublishMessage) ([]int64, error) {
	offsets := make([]int64, len(messages))

	// We need to group by partition but also keep track of original indices
	// to return offsets in the correct order.
	type msgWithIndex struct {
		msg *PublishMessage
		idx int
	}

	partitionMessages := make(map[int][]*msgWithIndex)
	for i, msg := range messages {
		partitionMessages[msg.Partition] = append(partitionMessages[msg.Partition], &msgWithIndex{msg, i})
	}

	// Process each partition
	for partition, msgs := range partitionMessages {
		// Atomic offset increment for each message
		for _, item := range msgs {
			offset := s.getPartitionOffset(topic, partition).Add(1)

			// Create message
			message := &Message{
				Topic:     topic,
				Partition: partition,
				Offset:    offset,
				Key:       item.msg.Key,
				Value:     item.msg.Value,
				Timestamp: time.Now().UnixMilli(),
			}

			// Buffer write
			partKey := fmt.Sprintf("%s:%d", topic, partition)
			s.writeBufferMu.Lock()
			s.writeBuffer[partKey] = append(s.writeBuffer[partKey], message)
			s.writeBufferMu.Unlock()

			offsets[item.idx] = offset
		}
	}

	// Flush if buffer is large
	s.writeBufferMu.Lock()
	totalBuffered := 0
	for _, msgs := range s.writeBuffer {
		totalBuffered += len(msgs)
	}
	s.writeBufferMu.Unlock()

	if totalBuffered >= 1000 {
		select {
		case s.flushCh <- true:
			// Triggered flush
		default:
			// Flush already in progress or channel full
		}
	}

	return offsets, nil
}
