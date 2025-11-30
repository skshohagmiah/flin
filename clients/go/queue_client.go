package flin

import (
	"encoding/binary"
	"errors"

	"github.com/skshohagmiah/flin/internal/net"
	protocol "github.com/skshohagmiah/flin/internal/net"
)

// QueueClient handles Message Queue operations
type QueueClient struct {
	pool *net.ConnectionPool
}

// Push adds an item to the queue
func (c *QueueClient) Push(queue string, item []byte) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeQPushRequest(queue, item)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Pop removes and returns an item from the queue
func (c *QueueClient) Pop(queue string) ([]byte, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeQPopRequest(queue)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readValueResponse(conn)
}

// Peek returns the next item without removing it
func (c *QueueClient) Peek(queue string) ([]byte, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeQPeekRequest(queue)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readValueResponse(conn)
}

// Len returns the number of items in the queue
func (c *QueueClient) Len(queue string) (int64, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return 0, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeQLenRequest(queue)
	if err := conn.Write(request); err != nil {
		return 0, err
	}

	value, err := readValueResponse(conn)
	if err != nil {
		return 0, err
	}

	if len(value) != 8 {
		return 0, errors.New("invalid length value")
	}

	return int64(binary.BigEndian.Uint64(value)), nil
}

// Clear removes all items from the queue
func (c *QueueClient) Clear(queue string) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeQClearRequest(queue)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}
