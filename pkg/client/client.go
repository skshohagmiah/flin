package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client represents a Flin KV client
type Client struct {
	addr   string
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
	
	// Connection pool settings
	maxRetries int
	timeout    time.Duration
}

// Config holds client configuration
type Config struct {
	Address    string        // Server address (e.g., "localhost:6380")
	Timeout    time.Duration // Connection timeout
	MaxRetries int           // Max retry attempts
}

// DefaultConfig returns default client configuration
func DefaultConfig() *Config {
	return &Config{
		Address:    "localhost:6380",
		Timeout:    5 * time.Second,
		MaxRetries: 3,
	}
}

// New creates a new Flin client
func New(addr string) (*Client, error) {
	config := DefaultConfig()
	config.Address = addr
	return NewWithConfig(config)
}

// NewWithConfig creates a new Flin client with custom configuration
func NewWithConfig(config *Config) (*Client, error) {
	conn, err := net.DialTimeout("tcp", config.Address, config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Address, err)
	}
	
	client := &Client{
		addr:       config.Address,
		conn:       conn,
		reader:     bufio.NewReaderSize(conn, 32768),
		writer:     bufio.NewWriterSize(conn, 32768),
		maxRetries: config.MaxRetries,
		timeout:    config.Timeout,
	}
	
	return client, nil
}

// Set stores a key-value pair
func (c *Client) Set(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Send command: SET key value\r\n
	cmd := fmt.Sprintf("SET %s %s\r\n", key, value)
	if _, err := c.writer.WriteString(cmd); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	
	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("flush error: %w", err)
	}
	
	// Read response
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	
	if len(response) > 0 && response[0] == '-' {
		return fmt.Errorf("server error: %s", response[1:])
	}
	
	return nil
}

// Get retrieves a value by key
func (c *Client) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Send command: GET key\r\n
	cmd := fmt.Sprintf("GET %s\r\n", key)
	if _, err := c.writer.WriteString(cmd); err != nil {
		return nil, fmt.Errorf("write error: %w", err)
	}
	
	if err := c.writer.Flush(); err != nil {
		return nil, fmt.Errorf("flush error: %w", err)
	}
	
	// Read bulk string response: $length\r\ndata\r\n
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	
	if len(response) > 0 && response[0] == '-' {
		return nil, fmt.Errorf("server error: %s", response[1:])
	}
	
	// Parse bulk string length
	var length int
	if _, err := fmt.Sscanf(response, "$%d\r\n", &length); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	
	// Read value
	value := make([]byte, length)
	if _, err := c.reader.Read(value); err != nil {
		return nil, fmt.Errorf("read value error: %w", err)
	}
	
	// Read trailing \r\n
	c.reader.ReadString('\n')
	
	return value, nil
}

// Delete removes a key
func (c *Client) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cmd := fmt.Sprintf("DEL %s\r\n", key)
	if _, err := c.writer.WriteString(cmd); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	
	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("flush error: %w", err)
	}
	
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	
	if len(response) > 0 && response[0] == '-' {
		return fmt.Errorf("server error: %s", response[1:])
	}
	
	return nil
}

// Exists checks if a key exists
func (c *Client) Exists(key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cmd := fmt.Sprintf("EXISTS %s\r\n", key)
	if _, err := c.writer.WriteString(cmd); err != nil {
		return false, fmt.Errorf("write error: %w", err)
	}
	
	if err := c.writer.Flush(); err != nil {
		return false, fmt.Errorf("flush error: %w", err)
	}
	
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read error: %w", err)
	}
	
	if len(response) > 0 && response[0] == '-' {
		return false, fmt.Errorf("server error: %s", response[1:])
	}
	
	// Parse integer response: :1\r\n or :0\r\n
	var exists int
	if _, err := fmt.Sscanf(response, ":%d\r\n", &exists); err != nil {
		return false, fmt.Errorf("parse error: %w", err)
	}
	
	return exists == 1, nil
}

// Ping checks if the server is alive
func (c *Client) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cmd := "PING\r\n"
	if _, err := c.writer.WriteString(cmd); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	
	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("flush error: %w", err)
	}
	
	response, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	
	if response != "+PONG\r\n" {
		return fmt.Errorf("unexpected response: %s", response)
	}
	
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Reconnect attempts to reconnect to the server
func (c *Client) Reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.conn != nil {
		c.conn.Close()
	}
	
	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return fmt.Errorf("reconnect failed: %w", err)
	}
	
	c.conn = conn
	c.reader = bufio.NewReaderSize(conn, 32768)
	c.writer = bufio.NewWriterSize(conn, 32768)
	
	return nil
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	return c.conn != nil
}
