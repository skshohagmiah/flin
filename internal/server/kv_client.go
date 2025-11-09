package server

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// KVClient is a high-performance client for the NATS-style KV server
type KVClient struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewKVClient creates a new client connection
func NewKVClient(addr string) (*KVClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	
	return &KVClient{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, 32768),
		writer: bufio.NewWriterSize(conn, 32768),
	}, nil
}

// Set stores a key-value pair
func (c *KVClient) Set(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Send command
	cmd := fmt.Sprintf("SET %s %s\r\n", key, value)
	_, err := c.writer.WriteString(cmd)
	if err != nil {
		return err
	}
	
	err = c.writer.Flush()
	if err != nil {
		return err
	}
	
	// Read response
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return err
	}
	
	if response[0] == '-' {
		return fmt.Errorf("server error: %s", response[1:])
	}
	
	return nil
}

// Get retrieves a value by key
func (c *KVClient) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Send command
	cmd := fmt.Sprintf("GET %s\r\n", key)
	_, err := c.writer.WriteString(cmd)
	if err != nil {
		return nil, err
	}
	
	err = c.writer.Flush()
	if err != nil {
		return nil, err
	}
	
	// Read bulk string response
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	
	if response[0] == '-' {
		return nil, fmt.Errorf("server error: %s", response[1:])
	}
	
	// Parse bulk string length
	var length int
	_, err = fmt.Sscanf(response, "$%d\r\n", &length)
	if err != nil {
		return nil, err
	}
	
	// Read value
	value := make([]byte, length)
	_, err = c.reader.Read(value)
	if err != nil {
		return nil, err
	}
	
	// Read trailing \r\n
	c.reader.ReadString('\n')
	
	return value, nil
}

// Delete removes a key
func (c *KVClient) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cmd := fmt.Sprintf("DEL %s\r\n", key)
	_, err := c.writer.WriteString(cmd)
	if err != nil {
		return err
	}
	
	err = c.writer.Flush()
	if err != nil {
		return err
	}
	
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return err
	}
	
	if response[0] == '-' {
		return fmt.Errorf("server error: %s", response[1:])
	}
	
	return nil
}

// Close closes the connection
func (c *KVClient) Close() error {
	return c.conn.Close()
}
