package flin

import (
	"errors"
	"fmt"
	"time"

	"github.com/skshohagmiah/flin/internal/net"
)

// Client is the unified Flin client with namespaced APIs
type Client struct {
	// Namespaced APIs
	KV     *KVClient
	Queue  *QueueClient
	Stream *StreamClient
	DB     *DBClient

	// Internal connection management
	pool *net.ConnectionPool
	opts *ClientOptions
}

// ClientOptions for creating a new client
type ClientOptions struct {
	// Server address (unified port for all services)
	Address string

	// Connection pool settings
	MinConnections int
	MaxConnections int

	// Connection timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultOptions returns default client options
func DefaultOptions(address string) *ClientOptions {
	return &ClientOptions{
		Address:        address,
		MinConnections: 16,
		MaxConnections: 256,
		DialTimeout:    5 * time.Second,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
	}
}

// NewClient creates a new unified Flin client
func NewClient(opts *ClientOptions) (*Client, error) {
	if opts == nil {
		return nil, errors.New("options cannot be nil")
	}

	if opts.Address == "" {
		return nil, errors.New("address is required")
	}

	// Create shared connection pool
	poolOpts := &net.PoolOptions{
		Address:      opts.Address,
		MinSize:      opts.MinConnections,
		MaxSize:      opts.MaxConnections,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		MaxIdleTime:  5 * time.Minute,
		BufferSize:   65536,
	}

	pool, err := net.NewConnectionPool(poolOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	client := &Client{
		pool: pool,
		opts: opts,
	}

	// Initialize namespaced clients
	client.KV = &KVClient{pool: pool}
	client.Queue = &QueueClient{pool: pool}
	client.Stream = &StreamClient{pool: pool}
	client.DB = &DBClient{pool: pool}

	return client, nil
}

// BatchMessage represents a message in a batch
type BatchMessage struct {
	Partition int
	Key       string
	Value     []byte
}

// Publish publishes a message to a topic
func (c *Client) Publish(topic string, partition int, key string, value []byte) error {
	return c.Stream.Publish(topic, partition, key, value)
}

// PublishBatch publishes a batch of messages to a topic
func (c *Client) PublishBatch(topic string, messages []*BatchMessage) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	// Convert to internal net.BatchMessage
	netMessages := make([]*net.BatchMessage, len(messages))
	for i, msg := range messages {
		netMessages[i] = &net.BatchMessage{
			Partition: msg.Partition,
			Key:       msg.Key,
			Value:     msg.Value,
		}
	}

	req := net.EncodeSPublishBatchRequest(topic, netMessages)
	if err := conn.Write(req); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Close closes all connections
func (c *Client) Close() error {
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// Ping checks if the server is reachable
func (c *Client) Ping() error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)
	return nil
}
