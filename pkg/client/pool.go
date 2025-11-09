package client

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrPoolClosed = errors.New("connection pool is closed")
	ErrPoolEmpty  = errors.New("no available connections")
)

// Pool represents a connection pool
type Pool struct {
	config   *Config
	conns    chan *Client
	mu       sync.Mutex
	closed   bool
	minConns int
	maxConns int
}

// PoolConfig holds pool configuration
type PoolConfig struct {
	Address     string
	MinConns    int
	MaxConns    int
	Timeout     time.Duration
	IdleTimeout time.Duration
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		Address:     "localhost:6380",
		MinConns:    2,
		MaxConns:    10,
		Timeout:     5 * time.Second,
		IdleTimeout: 5 * time.Minute,
	}
}

// NewPool creates a new connection pool
func NewPool(config *PoolConfig) (*Pool, error) {
	if config.MinConns > config.MaxConns {
		config.MinConns = config.MaxConns
	}
	
	pool := &Pool{
		config: &Config{
			Address:    config.Address,
			Timeout:    config.Timeout,
			MaxRetries: 3,
		},
		conns:    make(chan *Client, config.MaxConns),
		minConns: config.MinConns,
		maxConns: config.MaxConns,
	}
	
	// Create minimum connections
	for i := 0; i < config.MinConns; i++ {
		client, err := NewWithConfig(pool.config)
		if err != nil {
			pool.Close()
			return nil, err
		}
		pool.conns <- client
	}
	
	return pool, nil
}

// Get retrieves a connection from the pool
func (p *Pool) Get() (*Client, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}
	p.mu.Unlock()
	
	select {
	case client := <-p.conns:
		// Test connection
		if err := client.Ping(); err != nil {
			// Connection is dead, try to reconnect
			if err := client.Reconnect(); err != nil {
				// Failed to reconnect, create new connection
				newClient, err := NewWithConfig(p.config)
				if err != nil {
					return nil, err
				}
				return newClient, nil
			}
		}
		return client, nil
	default:
		// No available connections, create new one if under max
		p.mu.Lock()
		currentSize := len(p.conns)
		p.mu.Unlock()
		
		if currentSize < p.maxConns {
			return NewWithConfig(p.config)
		}
		
		// Wait for available connection
		select {
		case client := <-p.conns:
			return client, nil
		case <-time.After(p.config.Timeout):
			return nil, ErrPoolEmpty
		}
	}
}

// Put returns a connection to the pool
func (p *Pool) Put(client *Client) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return client.Close()
	}
	p.mu.Unlock()
	
	select {
	case p.conns <- client:
		return nil
	default:
		// Pool is full, close the connection
		return client.Close()
	}
}

// Close closes all connections in the pool
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil
	}
	
	p.closed = true
	close(p.conns)
	
	// Close all connections
	for client := range p.conns {
		client.Close()
	}
	
	return nil
}

// Stats returns pool statistics
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return map[string]interface{}{
		"available": len(p.conns),
		"max":       p.maxConns,
		"min":       p.minConns,
		"closed":    p.closed,
	}
}

// PooledClient wraps a pool with convenient methods
type PooledClient struct {
	pool *Pool
}

// NewPooledClient creates a new pooled client
func NewPooledClient(config *PoolConfig) (*PooledClient, error) {
	pool, err := NewPool(config)
	if err != nil {
		return nil, err
	}
	
	return &PooledClient{pool: pool}, nil
}

// Set stores a key-value pair using a pooled connection
func (pc *PooledClient) Set(key string, value []byte) error {
	client, err := pc.pool.Get()
	if err != nil {
		return err
	}
	defer pc.pool.Put(client)
	
	return client.Set(key, value)
}

// Get retrieves a value using a pooled connection
func (pc *PooledClient) Get(key string) ([]byte, error) {
	client, err := pc.pool.Get()
	if err != nil {
		return nil, err
	}
	defer pc.pool.Put(client)
	
	return client.Get(key)
}

// Delete removes a key using a pooled connection
func (pc *PooledClient) Delete(key string) error {
	client, err := pc.pool.Get()
	if err != nil {
		return err
	}
	defer pc.pool.Put(client)
	
	return client.Delete(key)
}

// Exists checks if a key exists using a pooled connection
func (pc *PooledClient) Exists(key string) (bool, error) {
	client, err := pc.pool.Get()
	if err != nil {
		return false, err
	}
	defer pc.pool.Put(client)
	
	return client.Exists(key)
}

// Close closes the pool
func (pc *PooledClient) Close() error {
	return pc.pool.Close()
}

// Stats returns pool statistics
func (pc *PooledClient) Stats() map[string]interface{} {
	return pc.pool.Stats()
}
