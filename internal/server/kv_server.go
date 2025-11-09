package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skshohagmiah/flin/internal/kv"
)

// KVServer implements NATS-style architecture for KV operations
// - One goroutine per connection (readLoop + writeLoop)
// - Lock-free message routing
// - Buffered channels for non-blocking dispatch
type KVServer struct {
	store       *kv.KVStore
	listener    net.Listener
	connections sync.Map // map[uint64]*Connection
	connCounter atomic.Uint64
	
	// Metrics
	opsProcessed atomic.Uint64
	activeConns  atomic.Int64
	
	ctx    context.Context
	cancel context.CancelFunc
}

// Connection represents a single client connection (NATS-style)
type Connection struct {
	id       uint64
	conn     net.Conn
	server   *KVServer
	
	// Channels for async communication
	outQueue chan []byte // Buffered channel for responses
	
	// Per-connection buffer pool
	readBuf  []byte
	writeBuf []byte
	
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewKVServer creates a new NATS-style KV server
func NewKVServer(store *kv.KVStore, addr string) (*KVServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &KVServer{
		store:    store,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Start begins accepting connections (NATS-style accept loop)
func (s *KVServer) Start() error {
	fmt.Printf("KV Server listening on %s\n", s.listener.Addr())
	
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				continue
			}
		}
		
		// Spawn connection handler (NATS pattern: 2 goroutines per connection)
		go s.handleConnection(conn)
	}
}

// handleConnection manages a single client connection
func (s *KVServer) handleConnection(netConn net.Conn) {
	connID := s.connCounter.Add(1)
	s.activeConns.Add(1)
	defer s.activeConns.Add(-1)
	
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	
	conn := &Connection{
		id:       connID,
		conn:     netConn,
		server:   s,
		outQueue: make(chan []byte, 1000), // Buffered for non-blocking sends
		readBuf:  make([]byte, 32768),     // 32KB read buffer
		writeBuf: make([]byte, 32768),     // 32KB write buffer
		ctx:      ctx,
		cancel:   cancel,
	}
	
	s.connections.Store(connID, conn)
	defer s.connections.Delete(connID)
	defer netConn.Close()
	
	// NATS pattern: spawn readLoop and writeLoop
	var wg sync.WaitGroup
	wg.Add(2)
	
	go func() {
		defer wg.Done()
		conn.readLoop()
	}()
	
	go func() {
		defer wg.Done()
		conn.writeLoop()
	}()
	
	wg.Wait()
}

// readLoop handles incoming requests (NATS-style: parse and dispatch in same goroutine)
func (c *Connection) readLoop() {
	defer c.cancel()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		
		// Set read deadline
		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		
		n, err := c.conn.Read(c.readBuf)
		if err != nil {
			return
		}
		
		// Parse and process in same goroutine (no spawning!)
		// This is the NATS way: fast, inline processing
		c.processRequest(c.readBuf[:n])
	}
}

// writeLoop handles outgoing responses (NATS-style: dedicated write goroutine)
func (c *Connection) writeLoop() {
	defer c.cancel()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.outQueue:
			// Set write deadline
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			
			_, err := c.conn.Write(msg)
			if err != nil {
				return
			}
		}
	}
}

// processRequest handles a single request inline (NATS pattern: no goroutine spawn)
func (c *Connection) processRequest(data []byte) {
	// Fast path: parse command inline
	cmd, key, value, err := c.parseCommand(data)
	if err != nil {
		c.sendError(err)
		return
	}
	
	// Execute command (inline, no spawning)
	var response []byte
	
	switch cmd {
	case "SET":
		err = c.server.store.Set(key, value, 0)
		if err != nil {
			response = c.formatError(err)
		} else {
			response = []byte("+OK\r\n")
		}
		
	case "GET":
		val, err := c.server.store.Get(key)
		if err != nil {
			response = c.formatError(err)
		} else {
			response = c.formatBulkString(val)
		}
		
	case "DEL":
		err = c.server.store.Delete(key)
		if err != nil {
			response = c.formatError(err)
		} else {
			response = []byte("+OK\r\n")
		}
		
	case "EXISTS":
		exists, err := c.server.store.Exists(key)
		if err != nil {
			response = c.formatError(err)
		} else if exists {
			response = []byte(":1\r\n")
		} else {
			response = []byte(":0\r\n")
		}
		
	default:
		response = []byte("-ERR unknown command\r\n")
	}
	
	// Non-blocking send to write queue
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
	case <-c.ctx.Done():
		return
	default:
		// Queue full, drop or handle backpressure
	}
}

// parseCommand parses Redis-style protocol (simple version)
func (c *Connection) parseCommand(data []byte) (cmd, key string, value []byte, err error) {
	// Simple parser for: SET key value\r\n or GET key\r\n
	// In production, use proper RESP protocol parser
	
	parts := make([][]byte, 0, 3)
	start := 0
	
	for i := 0; i < len(data); i++ {
		if data[i] == ' ' || data[i] == '\r' || data[i] == '\n' {
			if i > start {
				parts = append(parts, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		parts = append(parts, data[start:])
	}
	
	if len(parts) < 2 {
		return "", "", nil, fmt.Errorf("invalid command")
	}
	
	cmd = string(parts[0])
	key = string(parts[1])
	
	if len(parts) > 2 {
		value = parts[2]
	}
	
	return cmd, key, value, nil
}

// formatBulkString formats response as Redis bulk string
func (c *Connection) formatBulkString(data []byte) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(data), data))
}

// formatError formats error response
func (c *Connection) formatError(err error) []byte {
	return []byte(fmt.Sprintf("-ERR %s\r\n", err.Error()))
}

// sendError sends error to client
func (c *Connection) sendError(err error) {
	select {
	case c.outQueue <- c.formatError(err):
	case <-c.ctx.Done():
	default:
	}
}

// Stop gracefully shuts down the server
func (s *KVServer) Stop() error {
	s.cancel()
	
	// Close all connections
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			conn.cancel()
		}
		return true
	})
	
	return s.listener.Close()
}

// Stats returns server statistics
func (s *KVServer) Stats() map[string]interface{} {
	return map[string]interface{}{
		"active_connections": s.activeConns.Load(),
		"ops_processed":      s.opsProcessed.Load(),
	}
}
