package flin

import (
	"encoding/binary"
	"fmt"

	"github.com/skshohagmiah/flin/internal/net"
	"github.com/skshohagmiah/flin/pkg/protocol"
)

// QueueClient provides queue operations
type QueueClient struct {
	address string
	pool    *net.ConnectionPool
}

// NewQueueClient creates a new queue client
func NewQueueClient(address string, poolOpts *net.PoolOptions) (*QueueClient, error) {
	opts := *poolOpts
	opts.Address = address

	pool, err := net.NewConnectionPool(&opts)
	if err != nil {
		return nil, err
	}

	return &QueueClient{
		address: address,
		pool:    pool,
	}, nil
}

// Push adds an item to the queue
func (qc *QueueClient) Push(queueName string, value []byte) error {
	conn, err := qc.pool.Get()
	if err != nil {
		return err
	}
	defer qc.pool.Put(conn)

	request := protocol.EncodeQPushRequest(queueName, value)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Pop removes and returns the first item from the queue
func (qc *QueueClient) Pop(queueName string) ([]byte, error) {
	conn, err := qc.pool.Get()
	if err != nil {
		return nil, err
	}
	defer qc.pool.Put(conn)

	request := protocol.EncodeQPopRequest(queueName)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readValueResponse(conn)
}

// Peek returns the first item without removing it
func (qc *QueueClient) Peek(queueName string) ([]byte, error) {
	conn, err := qc.pool.Get()
	if err != nil {
		return nil, err
	}
	defer qc.pool.Put(conn)

	request := protocol.EncodeQPeekRequest(queueName)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readValueResponse(conn)
}

// Len returns the number of items in the queue
func (qc *QueueClient) Len(queueName string) (uint64, error) {
	conn, err := qc.pool.Get()
	if err != nil {
		return 0, err
	}
	defer qc.pool.Put(conn)

	request := protocol.EncodeQLenRequest(queueName)
	if err := conn.Write(request); err != nil {
		return 0, err
	}

	value, err := readValueResponse(conn)
	if err != nil {
		return 0, err
	}

	if len(value) != 8 {
		return 0, fmt.Errorf("invalid length response")
	}

	return binary.BigEndian.Uint64(value), nil
}

// Clear removes all items from the queue
func (qc *QueueClient) Clear(queueName string) error {
	conn, err := qc.pool.Get()
	if err != nil {
		return err
	}
	defer qc.pool.Put(conn)

	request := protocol.EncodeQClearRequest(queueName)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Close closes the queue client
func (qc *QueueClient) Close() error {
	return qc.pool.Close()
}
