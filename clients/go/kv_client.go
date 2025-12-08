package flin

import (
	"encoding/binary"
	"errors"

	"github.com/skshohagmiah/flin/internal/net"
	protocol "github.com/skshohagmiah/flin/internal/net"
)

// KVClient handles Key-Value store operations
type KVClient struct {
	pool *net.ConnectionPool
}

// Set stores a key-value pair
func (c *KVClient) Set(key string, value []byte) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeSetRequest(key, value)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Get retrieves a value by key
func (c *KVClient) Get(key string) ([]byte, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeGetRequest(key)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readValueResponse(conn)
}

// Delete removes a key
func (c *KVClient) Delete(key string) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeDeleteRequest(key)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Exists checks if a key exists
func (c *KVClient) Exists(key string) (bool, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return false, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeExistsRequest(key)
	if err := conn.Write(request); err != nil {
		return false, err
	}

	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return false, err
	}

	if status == protocol.StatusOK && payloadLen > 0 {
		payload, err := conn.Read(int(payloadLen))
		if err != nil {
			return false, err
		}
		return payload[0] == 1, nil
	}

	return false, nil
}

// Incr increments a counter
func (c *KVClient) Incr(key string) (int64, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return 0, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeIncrRequest(key)
	if err := conn.Write(request); err != nil {
		return 0, err
	}

	value, err := readValueResponse(conn)
	if err != nil {
		return 0, err
	}

	if len(value) != 8 {
		return 0, errors.New("invalid counter value")
	}

	return int64(binary.BigEndian.Uint64(value)), nil
}

// Decr decrements a counter
func (c *KVClient) Decr(key string) (int64, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return 0, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeDecrRequest(key)
	if err := conn.Write(request); err != nil {
		return 0, err
	}

	value, err := readValueResponse(conn)
	if err != nil {
		return 0, err
	}

	if len(value) != 8 {
		return 0, errors.New("invalid counter value")
	}

	return int64(binary.BigEndian.Uint64(value)), nil
}

// MSet performs a batch set operation
func (c *KVClient) MSet(keys []string, values [][]byte) error {
	if len(keys) != len(values) {
		return errors.New("keys and values length mismatch")
	}

	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeMSetRequest(keys, values)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// MGet performs a batch get operation
func (c *KVClient) MGet(keys []string) ([][]byte, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeMGetRequest(keys)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readMultiValueResponse(conn)
}

// MDelete performs a batch delete operation
func (c *KVClient) MDelete(keys []string) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeMDeleteRequest(keys)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}
