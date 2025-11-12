package net

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ConnectionPool manages a pool of connections
type ConnectionPool struct {
	opts        *ConnectionOptions
	conns       chan *Connection
	mu          sync.Mutex
	closed      bool
	activeCount int
	minSize     int
	maxSize     int
	maxIdleTime time.Duration
}

// PoolOptions for creating a connection pool
type PoolOptions struct {
	Address      string
	MinSize      int
	MaxSize      int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MaxIdleTime  time.Duration
	BufferSize   int
}

// DefaultPoolOptions returns default pool options
func DefaultPoolOptions(address string) *PoolOptions {
	return &PoolOptions{
		Address:      address,
		MinSize:      5,
		MaxSize:      50,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		MaxIdleTime:  5 * time.Minute,
		BufferSize:   65536,
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(opts *PoolOptions) (*ConnectionPool, error) {
	if opts == nil {
		return nil, errors.New("options cannot be nil")
	}

	if opts.MinSize < 0 || opts.MaxSize < opts.MinSize {
		return nil, errors.New("invalid pool size configuration")
	}

	pool := &ConnectionPool{
		opts: &ConnectionOptions{
			Address:      opts.Address,
			DialTimeout:  opts.DialTimeout,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
			BufferSize:   opts.BufferSize,
		},
		conns:       make(chan *Connection, opts.MaxSize),
		minSize:     opts.MinSize,
		maxSize:     opts.MaxSize,
		maxIdleTime: opts.MaxIdleTime,
	}

	// Pre-create minimum connections
	for i := 0; i < opts.MinSize; i++ {
		conn, err := NewConnection(pool.opts)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		pool.conns <- conn
		pool.activeCount++
	}

	return pool, nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get() (*Connection, error) {
	// Fast path: try to get from pool
	select {
	case conn := <-p.conns:
		if conn.IsConnected() {
			return conn, nil
		}
		// Connection is dead, decrement count and try to create new one
		p.mu.Lock()
		p.activeCount--
		p.mu.Unlock()
	default:
		// No connection available
	}

	// Try to create new connection or wait
	return p.createNewOrWait()
}

// createNewOrWait creates a new connection if under max, otherwise waits
func (p *ConnectionPool) createNewOrWait() (*Connection, error) {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()
		return nil, errors.New("pool is closed")
	}

	// Try to create new connection if under limit
	if p.activeCount < p.maxSize {
		p.activeCount++
		p.mu.Unlock()

		conn, err := NewConnection(p.opts)
		if err != nil {
			// Failed to create, decrement count
			p.mu.Lock()
			p.activeCount--
			p.mu.Unlock()
			return nil, err
		}
		return conn, nil
	}
	p.mu.Unlock()

	// At max capacity, wait for an available connection
	conn := <-p.conns
	if !conn.IsConnected() {
		// Connection is dead, try again
		p.mu.Lock()
		p.activeCount--
		p.mu.Unlock()
		return p.createNewOrWait()
	}
	return conn, nil
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn *Connection) {
	if conn == nil {
		return
	}

	// Check if connection is still good
	if !conn.IsConnected() {
		p.mu.Lock()
		p.activeCount--
		p.mu.Unlock()
		return
	}

	// Fast path: try to return to pool
	select {
	case p.conns <- conn:
		// Successfully returned to pool
		return
	default:
		// Pool is full, close the connection
		conn.Close()
		p.mu.Lock()
		p.activeCount--
		p.mu.Unlock()
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.conns)
	for conn := range p.conns {
		conn.Close()
	}
	return nil
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	return PoolStats{
		ActiveCount:    p.activeCount,
		AvailableCount: len(p.conns),
		MaxSize:        p.maxSize,
		MinSize:        p.minSize,
	}
}

// PoolStats represents pool statistics
type PoolStats struct {
	ActiveCount    int
	AvailableCount int
	MaxSize        int
	MinSize        int
}
