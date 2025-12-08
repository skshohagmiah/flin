package server

import (
	"encoding/binary"
	"fmt"
	"time"

	protocol "github.com/skshohagmiah/flin/internal/net"
	"github.com/skshohagmiah/flin/internal/stream"
)

// Stream operation handlers

func (c *Connection) processBinarySPublish(req *protocol.Request, startTime time.Time) {
	_, err := c.server.stream.Publish(req.Topic, req.Partition, req.Key, req.Value)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// For now, just return OK. Ideally we return the offset.
	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinarySConsume(req *protocol.Request, startTime time.Time) {
	msgs, err := c.server.stream.Consume(req.Topic, req.Group, req.Consumer, req.Count)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	values := make([][]byte, len(msgs))
	for i, msg := range msgs {
		// Encode message: [8:offset][8:timestamp][2:keyLen][key][4:valueLen][value]
		keyLen := len(msg.Key)
		valLen := len(msg.Value)
		size := 8 + 8 + 2 + keyLen + 4 + valLen
		buf := make([]byte, size)

		pos := 0
		binary.BigEndian.PutUint64(buf[pos:], uint64(msg.Offset))
		pos += 8
		binary.BigEndian.PutUint64(buf[pos:], uint64(msg.Timestamp))
		pos += 8
		binary.BigEndian.PutUint16(buf[pos:], uint16(keyLen))
		pos += 2
		copy(buf[pos:], msg.Key)
		pos += keyLen
		binary.BigEndian.PutUint32(buf[pos:], uint32(valLen))
		pos += 4
		copy(buf[pos:], msg.Value)

		values[i] = buf
	}

	c.sendBinaryResponse(protocol.EncodeMultiValueResponse(values), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinarySCommit(req *protocol.Request, startTime time.Time) {
	// req.RetentionMs holds the offset (hack from decoder)
	err := c.server.stream.Commit(req.Topic, req.Group, req.Partition, req.RetentionMs)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinarySCreateTopic(req *protocol.Request, startTime time.Time) {
	err := c.server.stream.CreateTopic(req.Topic, req.Partition, req.RetentionMs)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinarySSubscribe(req *protocol.Request, startTime time.Time) {
	err := c.server.stream.Subscribe(req.Topic, req.Group, req.Consumer)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinarySUnsubscribe(req *protocol.Request, startTime time.Time) {
	err := c.server.stream.Unsubscribe(req.Topic, req.Group, req.Consumer)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinarySPublishBatch(req *protocol.Request, startTime time.Time) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in processBinarySPublishBatch: %v\n", r)
			c.sendBinaryError(fmt.Errorf("internal server error"))
			c.server.opsErrors.Add(1)
		}
	}()

	// Convert protocol.BatchMessage to stream.PublishMessage
	messages := make([]*stream.PublishMessage, len(req.BatchMessages))
	for i, msg := range req.BatchMessages {
		messages[i] = &stream.PublishMessage{
			Partition: msg.Partition,
			Key:       msg.Key,
			Value:     msg.Value,
		}
	}

	offsets, err := c.server.stream.PublishBatch(req.Topic, messages)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// Encode response: [4:count][8:offset1][8:offset2]...
	// For now, just return OK to keep protocol simple as per handler_stream_batch.go
	// But ideally we should return offsets.
	// Let's stick to OK for now to match the client expectation if any,
	// or better, return the offsets.

	// Since we haven't defined a specific response format for batch publish in protocol yet,
	// and the user request didn't specify it, let's return OK for simplicity
	// and to avoid breaking changes if client expects standard response.
	// If we want to return offsets, we'd need a new response type or reuse MultiValue.

	// Let's return OK for now.
	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)

	// Prevent unused variable error
	_ = offsets
}
