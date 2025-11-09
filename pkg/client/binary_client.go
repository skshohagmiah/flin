package client

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/skshohagmiah/flin/pkg/protocol"
)

// BinaryClient implements high-performance binary protocol client
type BinaryClient struct {
	conn   net.Conn
	writer *bufio.Writer
	reader *bufio.Reader
	mu     sync.Mutex
}

// NewBinaryClient creates a new binary protocol client
func NewBinaryClient(addr string) (*BinaryClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Apply TCP optimizations
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	return &BinaryClient{
		conn:   conn,
		writer: bufio.NewWriterSize(conn, 65536), // 64KB buffer
		reader: bufio.NewReaderSize(conn, 65536),
	}, nil
}

// Set stores a key-value pair
func (c *BinaryClient) Set(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Encode request
	request := protocol.EncodeSetRequest(key, value)

	// Send request
	if _, err := c.writer.Write(request); err != nil {
		return err
	}
	if err := c.writer.Flush(); err != nil {
		return err
	}

	// Read response header (5 bytes)
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return err
	}

	// For OK response with no payload, we're done
	if header[0] == protocol.StatusOK {
		return nil
	}

	return fmt.Errorf("set failed: status %d", header[0])
}

// Get retrieves a value by key
func (c *BinaryClient) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Encode request
	request := protocol.EncodeGetRequest(key)

	// Send request
	if _, err := c.writer.Write(request); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	// Read response header (5 bytes)
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return nil, err
	}

	// Check status
	if header[0] != protocol.StatusOK {
		return nil, fmt.Errorf("get failed")
	}

	// Read payload
	payloadLen := binary.BigEndian.Uint32(header[1:5])
	if payloadLen == 0 {
		return nil, nil
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

// MSet performs a batch set operation
func (c *BinaryClient) MSet(keys []string, values [][]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Encode request
	request := protocol.EncodeMSetRequest(keys, values)

	// Send request
	if _, err := c.writer.Write(request); err != nil {
		return err
	}
	if err := c.writer.Flush(); err != nil {
		return err
	}

	// Read response header (5 bytes) using io.ReadFull
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return err
	}

	// For OK response with no payload, we're done
	if header[0] == protocol.StatusOK {
		return nil
	}

	return fmt.Errorf("mset failed: status %d", header[0])
}

// MGet performs a batch get operation
func (c *BinaryClient) MGet(keys []string) ([][]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Encode request
	request := protocol.EncodeMGetRequest(keys)

	// Send request
	if _, err := c.writer.Write(request); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	// Read response header
	header := make([]byte, 5)
	if _, err := c.reader.Read(header); err != nil {
		return nil, err
	}

	if header[0] != protocol.StatusMultiValue {
		return nil, fmt.Errorf("mget failed")
	}

	// Read payload length
	payloadLen := int(header[1])<<24 | int(header[2])<<16 | int(header[3])<<8 | int(header[4])

	// Read full payload
	payload := make([]byte, payloadLen)
	if _, err := c.reader.Read(payload); err != nil {
		return nil, err
	}

	// Combine header and payload for decoding
	fullResp := append(header, payload...)
	resp, err := protocol.DecodeResponse(fullResp)
	if err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// Close closes the connection
func (c *BinaryClient) Close() error {
	return c.conn.Close()
}
