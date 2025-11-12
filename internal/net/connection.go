package net

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Connection represents a single TCP connection with buffered I/O
type Connection struct {
	conn         net.Conn
	reader       *bufio.Reader
	writer       *bufio.Writer
	readTimeout  time.Duration
	writeTimeout time.Duration
	mu           sync.Mutex
	closed       bool
}

// ConnectionOptions for creating a new connection
type ConnectionOptions struct {
	Address      string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	BufferSize   int
}

// DefaultConnectionOptions returns default connection options
func DefaultConnectionOptions(address string) *ConnectionOptions {
	return &ConnectionOptions{
		Address:      address,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BufferSize:   65536, // 64KB
	}
}

// NewConnection creates a new TCP connection
func NewConnection(opts *ConnectionOptions) (*Connection, error) {
	if opts == nil {
		return nil, errors.New("options cannot be nil")
	}

	conn, err := net.DialTimeout("tcp", opts.Address, opts.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", opts.Address, err)
	}

	// Apply TCP optimizations
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	return &Connection{
		conn:         conn,
		reader:       bufio.NewReaderSize(conn, opts.BufferSize),
		writer:       bufio.NewWriterSize(conn, opts.BufferSize),
		readTimeout:  opts.ReadTimeout,
		writeTimeout: opts.WriteTimeout,
	}, nil
}

// Write writes data to the connection
func (c *Connection) Write(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("connection closed")
	}

	if c.writeTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	if _, err := c.writer.Write(data); err != nil {
		return err
	}

	return c.writer.Flush()
}

// Read reads exactly n bytes from the connection
func (c *Connection) Read(n int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errors.New("connection closed")
	}

	if c.readTimeout > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	data := make([]byte, n)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return nil, err
	}

	return data, nil
}

// ReadHeader reads a 5-byte protocol header
func (c *Connection) ReadHeader() (status byte, payloadLen uint32, err error) {
	header, err := c.Read(5)
	if err != nil {
		return 0, 0, err
	}

	status = header[0]
	payloadLen = binary.BigEndian.Uint32(header[1:5])
	return status, payloadLen, nil
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.conn.Close()
}

// IsConnected checks if the connection is still active
func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.closed
}

// RemoteAddr returns the remote address
func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// LocalAddr returns the local address
func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}
