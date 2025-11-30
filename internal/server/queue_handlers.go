package server

import (
	"encoding/binary"
	"time"

	protocol "github.com/skshohagmiah/flin/internal/net"
)

// Queue operation handlers

func (c *Connection) processBinaryQPush(req *protocol.Request, startTime time.Time) {
	err := c.server.queue.Push(req.Key, req.Value)

	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinaryQPop(req *protocol.Request, startTime time.Time) {
	value, err := c.server.queue.Pop(req.Key)

	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeValueResponse(value), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinaryQPeek(req *protocol.Request, startTime time.Time) {
	value, err := c.server.queue.Peek(req.Key)

	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeValueResponse(value), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinaryQLen(req *protocol.Request, startTime time.Time) {
	length, err := c.server.queue.Len(req.Key)

	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// Encode length as 8-byte value
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, length)

	c.sendBinaryResponse(protocol.EncodeValueResponse(buf), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}

func (c *Connection) processBinaryQClear(req *protocol.Request, startTime time.Time) {
	err := c.server.queue.Clear(req.Key)

	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	c.sendBinaryResponse(protocol.EncodeOKResponse(), startTime)
	c.server.opsProcessed.Add(1)
	c.server.opsFastPath.Add(1)
}
