package server

import (
	"encoding/binary"
	"time"

	protocol "github.com/skshohagmiah/flin/internal/net"
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
