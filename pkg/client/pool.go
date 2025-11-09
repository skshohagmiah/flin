package client

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	DefaultPartitionCount = 256  // Optimized for maximum throughput
	DefaultReconnectDelay = 1 * time.Second
	DefaultReadTimeout    = 30 * time.Second
	DefaultWriteTimeout   = 10 * time.Second
)

// PoolClient manages 64 persistent connections mapped to partitions
type PoolClient struct {
	connections    []*PoolConnection
	partitionCount int
	nodeAddrs      []string

	opsProcessed atomic.Uint64
	opsErrors    atomic.Uint64

	closed atomic.Bool
	wg     sync.WaitGroup
}

// PoolConnection represents a single persistent connection
type PoolConnection struct {
	id       int
	nodeAddr string
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer

	connected    atomic.Bool
	reconnecting atomic.Bool

	writeMu sync.Mutex

	ctx    *PoolClient
	closed atomic.Bool
}

// PoolConfig holds configuration
type PoolConfig struct {
	NodeAddrs      []string
	PartitionCount int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ReconnectDelay time.Duration
}

// DefaultPoolConfig returns default configuration
func DefaultPoolConfig(nodeAddrs []string) *PoolConfig {
	return &PoolConfig{
		NodeAddrs:      nodeAddrs,
		PartitionCount: DefaultPartitionCount,
		ReadTimeout:    DefaultReadTimeout,
		WriteTimeout:   DefaultWriteTimeout,
		ReconnectDelay: DefaultReconnectDelay,
	}
}

// NewPoolClient creates a new partition-aware connection pool
func NewPoolClient(config *PoolConfig) (*PoolClient, error) {
	if len(config.NodeAddrs) == 0 {
		return nil, fmt.Errorf("at least one node address required")
	}

	client := &PoolClient{
		connections:    make([]*PoolConnection, config.PartitionCount),
		partitionCount: config.PartitionCount,
		nodeAddrs:      config.NodeAddrs,
	}

	for i := 0; i < config.PartitionCount; i++ {
		nodeAddr := config.NodeAddrs[i%len(config.NodeAddrs)]

		conn := &PoolConnection{
			id:       i,
			nodeAddr: nodeAddr,
			ctx:      client,
		}

		client.connections[i] = conn

		client.wg.Add(1)
		go func(c *PoolConnection) {
			defer client.wg.Done()
			c.connect()
			c.maintainConnection(config.ReconnectDelay)
		}(conn)
	}

	return client, nil
}

func (c *PoolConnection) connect() error {
	if c.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	conn, err := net.DialTimeout("tcp", c.nodeAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.nodeAddr, err)
	}

	// Optimize TCP connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		// Disable Nagle's algorithm for low latency
		tcpConn.SetNoDelay(true)

		// Enable TCP keepalive
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)

		// Set large socket buffers for high throughput
		if rawConn, err := tcpConn.SyscallConn(); err == nil {
			rawConn.Control(func(fd uintptr) {
				// 4MB send/receive buffers
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 4*1024*1024)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
			})
		}
	}

	c.conn = conn
	c.reader = bufio.NewReaderSize(conn, 32768)
	c.writer = bufio.NewWriterSize(conn, 32768)
	c.connected.Store(true)

	return nil
}

func (c *PoolConnection) maintainConnection(reconnectDelay time.Duration) {
	for !c.closed.Load() {
		if !c.connected.Load() && !c.reconnecting.Load() {
			c.reconnecting.Store(true)

			if err := c.connect(); err != nil {
				time.Sleep(reconnectDelay)
			}

			c.reconnecting.Store(false)
		}

		time.Sleep(1 * time.Second)
	}
}

func (c *PoolClient) getPartition(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % uint32(c.partitionCount))
}

func (c *PoolClient) getConnection(key string) *PoolConnection {
	partition := c.getPartition(key)
	return c.connections[partition]
}

func (c *PoolClient) Set(key string, value []byte) error {
	if c.closed.Load() {
		return fmt.Errorf("client closed")
	}

	conn := c.getConnection(key)
	err := conn.set(key, value)

	if err != nil {
		c.opsErrors.Add(1)
		return err
	}

	c.opsProcessed.Add(1)
	return nil
}

func (c *PoolClient) Get(key string) ([]byte, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("client closed")
	}

	conn := c.getConnection(key)
	value, err := conn.get(key)

	if err != nil {
		c.opsErrors.Add(1)
		return nil, err
	}

	c.opsProcessed.Add(1)
	return value, nil
}

func (c *PoolClient) Delete(key string) error {
	if c.closed.Load() {
		return fmt.Errorf("client closed")
	}

	conn := c.getConnection(key)
	err := conn.delete(key)

	if err != nil {
		c.opsErrors.Add(1)
		return err
	}

	c.opsProcessed.Add(1)
	return nil
}

func (c *PoolClient) Exists(key string) (bool, error) {
	if c.closed.Load() {
		return false, fmt.Errorf("client closed")
	}

	conn := c.getConnection(key)
	exists, err := conn.exists(key)

	if err != nil {
		c.opsErrors.Add(1)
		return false, err
	}

	c.opsProcessed.Add(1)
	return exists, nil
}

func (c *PoolConnection) set(key string, value []byte) error {
	if !c.connected.Load() {
		return fmt.Errorf("connection not established")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	cmd := fmt.Sprintf("SET %s %s\r\n", key, string(value))

	c.conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	if _, err := c.writer.WriteString(cmd); err != nil {
		c.connected.Store(false)
		return fmt.Errorf("write error: %w", err)
	}

	if err := c.writer.Flush(); err != nil {
		c.connected.Store(false)
		return fmt.Errorf("flush error: %w", err)
	}

	c.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout))
	response, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected.Store(false)
		return fmt.Errorf("read error: %w", err)
	}

	if len(response) > 0 && response[0] == '-' {
		return fmt.Errorf("server error: %s", response[1:])
	}

	return nil
}

func (c *PoolConnection) get(key string) ([]byte, error) {
	if !c.connected.Load() {
		return nil, fmt.Errorf("connection not established")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	cmd := fmt.Sprintf("GET %s\r\n", key)

	c.conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	if _, err := c.writer.WriteString(cmd); err != nil {
		c.connected.Store(false)
		return nil, err
	}

	if err := c.writer.Flush(); err != nil {
		c.connected.Store(false)
		return nil, err
	}

	c.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout))
	response, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected.Store(false)
		return nil, err
	}

	if len(response) > 0 && response[0] == '-' {
		return nil, fmt.Errorf("key not found")
	}

	var length int
	if _, err := fmt.Sscanf(response, "$%d\r\n", &length); err != nil {
		return nil, fmt.Errorf("invalid response format")
	}

	data := make([]byte, length)
	if _, err := c.reader.Read(data); err != nil {
		c.connected.Store(false)
		return nil, err
	}

	c.reader.ReadString('\n')

	return data, nil
}

func (c *PoolConnection) delete(key string) error {
	if !c.connected.Load() {
		return fmt.Errorf("connection not established")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	cmd := fmt.Sprintf("DEL %s\r\n", key)

	c.conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	if _, err := c.writer.WriteString(cmd); err != nil {
		c.connected.Store(false)
		return err
	}

	if err := c.writer.Flush(); err != nil {
		c.connected.Store(false)
		return err
	}

	c.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout))
	response, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected.Store(false)
		return err
	}

	if len(response) > 0 && response[0] == '-' {
		return fmt.Errorf("server error: %s", response[1:])
	}

	return nil
}

func (c *PoolConnection) exists(key string) (bool, error) {
	if !c.connected.Load() {
		return false, fmt.Errorf("connection not established")
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	cmd := fmt.Sprintf("EXISTS %s\r\n", key)

	c.conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout))
	if _, err := c.writer.WriteString(cmd); err != nil {
		c.connected.Store(false)
		return false, err
	}

	if err := c.writer.Flush(); err != nil {
		c.connected.Store(false)
		return false, err
	}

	c.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout))
	response, err := c.reader.ReadString('\n')
	if err != nil {
		c.connected.Store(false)
		return false, err
	}

	var exists int
	if _, err := fmt.Sscanf(response, ":%d\r\n", &exists); err != nil {
		return false, fmt.Errorf("invalid response format")
	}

	return exists == 1, nil
}

func (c *PoolClient) Stats() map[string]interface{} {
	connectedCount := 0
	for _, conn := range c.connections {
		if conn.connected.Load() {
			connectedCount++
		}
	}

	return map[string]interface{}{
		"total_connections": len(c.connections),
		"connected":         connectedCount,
		"ops_processed":     c.opsProcessed.Load(),
		"ops_errors":        c.opsErrors.Load(),
		"partition_count":   c.partitionCount,
		"node_count":        len(c.nodeAddrs),
	}
}

func (c *PoolClient) Close() error {
	if c.closed.Swap(true) {
		return nil
	}

	for _, conn := range c.connections {
		conn.close()
	}

	c.wg.Wait()

	return nil
}

func (c *PoolConnection) close() error {
	if c.closed.Swap(true) {
		return nil
	}

	c.connected.Store(false)

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}
