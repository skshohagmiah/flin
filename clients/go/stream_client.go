package flin

import (
	"encoding/binary"
	"fmt"

	"github.com/skshohagmiah/flin/internal/net"
	protocol "github.com/skshohagmiah/flin/internal/net"
)

// StreamClient handles Stream Processing operations
type StreamClient struct {
	pool *net.ConnectionPool
}

// StreamMessage represents a message in a stream
type StreamMessage struct {
	ID        uint64
	Timestamp int64
	Key       string
	Value     []byte
	Partition int
	Offset    uint64
}

// CreateTopic creates a new topic with partitions and retention
func (c *StreamClient) CreateTopic(topic string, partitions int, retentionMs int64) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSCreateTopicRequest(topic, partitions, retentionMs)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Publish publishes a message to a topic
func (c *StreamClient) Publish(topic string, partition int, key string, value []byte) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSPublishRequest(topic, partition, key, value)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Subscribe subscribes a consumer group to a topic
func (c *StreamClient) Subscribe(topic, group, consumer string) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSSubscribeRequest(topic, group, consumer)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Consume consumes messages from a topic
func (c *StreamClient) Consume(topic, group, consumer string, count int) ([]StreamMessage, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSConsumeRequest(topic, group, consumer, count)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	// Read response
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return nil, err
	}

	if status != protocol.StatusOK && status != protocol.StatusMultiValue {
		return nil, fmt.Errorf("server error: status %d", status)
	}

	payload, err := conn.Read(int(payloadLen))
	if err != nil {
		return nil, err
	}

	var messages []StreamMessage

	if status == protocol.StatusMultiValue {
		if len(payload) < 2 {
			return nil, fmt.Errorf("invalid multi-value response")
		}
		count := binary.BigEndian.Uint16(payload[0:2])
		pos := 2

		for i := 0; i < int(count); i++ {
			// Read value length
			if len(payload) < pos+4 {
				return nil, fmt.Errorf("invalid response")
			}
			valLen := binary.BigEndian.Uint32(payload[pos:])
			pos += 4

			if len(payload) < pos+int(valLen) {
				return nil, fmt.Errorf("invalid response")
			}
			msgData := payload[pos : pos+int(valLen)]
			pos += int(valLen)

			// Decode message
			msg, err := decodeStreamMessage(msgData)
			if err != nil {
				return nil, err
			}
			// Set topic/partition from context if needed, but message has them?
			// The encoded message format in storage/stream.go is:
			// [8:offset][8:timestamp][2:keyLen][key][4:valueLen][value]
			// It does NOT contain topic/partition.
			// So we should set them from the request context if needed.
			msg.Partition = 0 // TODO: Server should return partition?
			// For now, we just return what we have.
			messages = append(messages, *msg)
		}
	}

	return messages, nil
}

func decodeStreamMessage(data []byte) (*StreamMessage, error) {
	if len(data) < 18 {
		return nil, fmt.Errorf("invalid message data")
	}
	msg := &StreamMessage{}
	pos := 0
	msg.Offset = binary.BigEndian.Uint64(data[pos:])
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
	valLen := int(binary.BigEndian.Uint32(data[pos:]))
	pos += 4
	if len(data) < pos+valLen {
		return nil, fmt.Errorf("invalid message data")
	}
	msg.Value = make([]byte, valLen)
	copy(msg.Value, data[pos:pos+valLen])
	return msg, nil
}

// Commit commits an offset for a consumer group
func (c *StreamClient) Commit(topic, group string, partition int, offset uint64) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSCommitRequest(topic, group, partition, offset)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Unsubscribe removes a consumer from a group
func (c *StreamClient) Unsubscribe(topic, group, consumer string) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSUnsubscribeRequest(topic, group, consumer)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}
