package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/skshohagmiah/clusterkit"
	"github.com/skshohagmiah/flin/internal/db"
	"github.com/skshohagmiah/flin/internal/kv"
	protocol "github.com/skshohagmiah/flin/internal/net"
	"github.com/skshohagmiah/flin/internal/queue"
	"github.com/skshohagmiah/flin/internal/stream"
)

// Buffer pool for connection buffers (32KB each)
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32768)
	},
}

type Server struct {
	store       *kv.KVStore
	queue       *queue.Queue
	stream      *stream.Stream
	db          *db.DocStore
	ck          *clusterkit.ClusterKit
	listener    net.Listener
	connections sync.Map
	connCounter atomic.Uint64
	nodeID      string

	// Worker pool for slow operations
	workerPool *WorkerPool
	jobQueue   chan *Job

	// Metrics
	opsProcessed atomic.Uint64
	opsFastPath  atomic.Uint64
	opsSlowPath  atomic.Uint64
	opsErrors    atomic.Uint64
	activeConns  atomic.Int64

	ctx    context.Context
	cancel context.CancelFunc
}

// Connection represents a single client connection (NATS-style with hybrid processing)
type Connection struct {
	id     uint64
	conn   net.Conn
	server *Server

	// Channels for async communication
	outQueue chan []byte // Buffered channel for responses

	// Per-connection buffer pool
	readBuf  []byte
	writeBuf []byte

	// Metrics for adaptive routing
	opsProcessed atomic.Uint64
	avgLatency   atomic.Int64 // in nanoseconds

	ctx    context.Context
	cancel context.CancelFunc
}

// Job represents a work item for the worker pool
type Job struct {
	conn  *Connection
	cmd   string
	key   string
	value []byte
	// Batch operation fields
	keys      []string
	values    [][]byte
	kvPairs   map[string][]byte
	startTime time.Time
}

// WorkerPool manages a pool of worker goroutines
type WorkerPool struct {
	workers  int
	jobQueue chan *Job
	store    *kv.KVStore
	wg       sync.WaitGroup

	// Metrics
	jobsProcessed atomic.Uint64
	activeWorkers atomic.Int32
}

const (
	// Fast path threshold: operations faster than this are processed inline
	FastPathThreshold = 100 * time.Microsecond

	// Worker pool size (optimized for maximum throughput)
	DefaultWorkerPoolSize = 64

	// Job queue buffer size (increased for higher throughput)
	DefaultJobQueueSize = 50000
)

// NewServer creates a new distributed KV server with hybrid architecture
func NewServer(store *kv.KVStore, q *queue.Queue, docStore *db.DocStore, ck *clusterkit.ClusterKit, addr string, nodeID string) (*Server, error) {
	return NewServerWithWorkers(store, q, nil, docStore, ck, addr, nodeID, DefaultWorkerPoolSize)
}

// NewServerWithWorkers creates a server with custom worker count
func NewServerWithWorkers(store *kv.KVStore, q *queue.Queue, stream *stream.Stream, docStore *db.DocStore, ck *clusterkit.ClusterKit, addr string, nodeID string, workerCount int) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	jobQueue := make(chan *Job, DefaultJobQueueSize)

	srv := &Server{
		store:    store,
		queue:    q,
		stream:   stream,   // Initialize the new stream field
		db:       docStore, // Initialize the document store field
		ck:       ck,
		listener: listener,
		nodeID:   nodeID,
		jobQueue: jobQueue,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize worker pool with custom size
	srv.workerPool = NewWorkerPool(workerCount, jobQueue, store)

	// Register ClusterKit event hooks
	srv.registerHooks()

	log.Printf("[Hybrid] Server initialized with %d workers", workerCount)

	return srv, nil
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(numWorkers int, jobQueue chan *Job, store *kv.KVStore) *WorkerPool {
	wp := &WorkerPool{
		workers:  numWorkers,
		jobQueue: jobQueue,
		store:    store,
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	log.Printf("[WorkerPool] Started %d workers", numWorkers)

	return wp
}

// worker processes jobs from the queue
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for job := range wp.jobQueue {
		wp.activeWorkers.Add(1)

		// Process job
		response := wp.processJob(job)

		// Send response back to connection
		select {
		case job.conn.outQueue <- response:
			wp.jobsProcessed.Add(1)
		case <-job.conn.ctx.Done():
			// Connection closed
		default:
			// Queue full, drop response
		}

		wp.activeWorkers.Add(-1)
	}
}

// processJob executes a job and returns the response
func (wp *WorkerPool) processJob(job *Job) []byte {
	switch job.cmd {
	case "SET":
		err := wp.store.Set(job.key, job.value, 0)
		if err != nil {
			return formatError(err)
		}
		return []byte("+OK\r\n")

	case "GET":
		val, err := wp.store.Get(job.key)
		if err != nil {
			return formatError(err)
		}
		return formatBulkString(val)

	case "DEL":
		err := wp.store.Delete(job.key)
		if err != nil {
			return formatError(err)
		}
		return []byte("+OK\r\n")

	case "EXISTS":
		exists, err := wp.store.Exists(job.key)
		if err != nil {
			return formatError(err)
		}
		if exists {
			return []byte(":1\r\n")
		}
		return []byte(":0\r\n")

	case "INCR":
		val, err := wp.store.Incr(job.key)
		if err != nil {
			return formatError(err)
		}
		// Return the new value as a bulk string
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(val))
		return formatBulkString(buf[:])

	case "DECR":
		val, err := wp.store.Decr(job.key)
		if err != nil {
			return formatError(err)
		}
		// Return the new value as a bulk string
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(val))
		return formatBulkString(buf[:])

	// Batch operations
	case "MSET":
		// Use atomic batch set
		kvPairs := make(map[string][]byte, len(job.keys))
		for i, key := range job.keys {
			kvPairs[key] = job.values[i]
		}
		err := wp.store.BatchSet(kvPairs, 0)
		if err != nil {
			return formatError(err)
		}
		return []byte("+OK\r\n")

	case "MGET":
		// Use atomic batch get
		results, err := wp.store.BatchGet(job.keys)
		if err != nil {
			return formatError(err)
		}
		// Convert map to ordered slice
		values := make([][]byte, 0, len(job.keys))
		for _, key := range job.keys {
			if val, ok := results[key]; ok {
				values = append(values, val)
			} else {
				values = append(values, []byte(""))
			}
		}
		return formatBulkStrings(values)

	case "MDEL":
		// Use atomic batch delete
		err := wp.store.BatchDelete(job.keys)
		if err != nil {
			return formatError(err)
		}
		return []byte(fmt.Sprintf(":%d\r\n", len(job.keys)))

	default:
		return []byte("-ERR unknown command\r\n")
	}
}

// registerHooks sets up ClusterKit event handlers
func (s *Server) registerHooks() {
	s.ck.OnPartitionChange(func(event *clusterkit.PartitionChangeEvent) {
		if event.CopyToNode.ID != s.nodeID {
			return
		}
		log.Printf("[Cluster] 🔄 Partition %s assigned (reason: %s)",
			event.PartitionID, event.ChangeReason)
	})

	s.ck.OnNodeJoin(func(event *clusterkit.NodeJoinEvent) {
		log.Printf("[Cluster] 🎉 Node %s joined (cluster size: %d)",
			event.Node.ID, event.ClusterSize)
	})

	s.ck.OnNodeLeave(func(event *clusterkit.NodeLeaveEvent) {
		log.Printf("[Cluster] ❌ Node %s left (reason: %s)",
			event.Node.ID, event.Reason)
	})

	s.ck.OnRebalanceStart(func(event *clusterkit.RebalanceEvent) {
		log.Printf("[Cluster] ⚖️  Rebalance starting (trigger: %s)", event.Trigger)
	})

	s.ck.OnRebalanceComplete(func(event *clusterkit.RebalanceEvent, duration time.Duration) {
		log.Printf("[Cluster] ✅ Rebalance completed in %v", duration)
	})
}

// Start begins accepting connections (NATS-style accept loop)
func (s *Server) Start() error {
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

// optimizeTCPConnection applies TCP optimizations for high throughput
func optimizeTCPConnection(conn net.Conn) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil
	}

	// Disable Nagle's algorithm for low latency
	if err := tcpConn.SetNoDelay(true); err != nil {
		return err
	}

	// Enable TCP keepalive
	if err := tcpConn.SetKeepAlive(true); err != nil {
		return err
	}
	if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
		return err
	}

	// Set socket buffer sizes for high throughput
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	err = rawConn.Control(func(fd uintptr) {
		// Set send buffer to 4MB
		sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 4*1024*1024)
		if sockErr != nil {
			return
		}

		// Set receive buffer to 4MB
		sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
	})

	if err != nil {
		return err
	}
	return sockErr
}

// handleConnection manages a single client connection
func (s *Server) handleConnection(netConn net.Conn) {
	connID := s.connCounter.Add(1)
	s.activeConns.Add(1)
	defer s.activeConns.Add(-1)

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// Optimize TCP connection
	if err := optimizeTCPConnection(netConn); err != nil {
		log.Printf("[WARN] Failed to optimize TCP connection: %v", err)
	}

	// Get buffers from pool
	readBuf := bufferPool.Get().([]byte)
	writeBuf := bufferPool.Get().([]byte)

	conn := &Connection{
		id:       connID,
		conn:     netConn,
		server:   s,
		outQueue: make(chan []byte, 5000), // Increased for higher throughput
		readBuf:  readBuf,
		writeBuf: writeBuf,
		ctx:      ctx,
		cancel:   cancel,
	}

	s.connections.Store(connID, conn)
	defer s.connections.Delete(connID)
	defer netConn.Close()

	// Return buffers to pool when done
	defer func() {
		bufferPool.Put(readBuf)
		bufferPool.Put(writeBuf)
	}()

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

// readLoop handles incoming requests with hybrid processing
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

		// Process with hybrid approach: fast path inline, slow path to worker pool
		c.processRequestHybrid(c.readBuf[:n])
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

// processRequestHybrid implements hybrid processing: fast path inline, slow path to worker pool
// Auto-detects binary vs text protocol
func (c *Connection) processRequestHybrid(data []byte) {
	startTime := time.Now()

	// Detect protocol: binary starts with opcode 0x01-0x12 (KV) or 0x20-0x24 (Queue) or 0x30-0x36 (Stream) or 0x40-0x44 (Document), text starts with ASCII letters
	isBinary := len(data) > 0 && ((data[0] >= 0x01 && data[0] <= 0x12) || (data[0] >= 0x20 && data[0] <= 0x24) || (data[0] >= 0x30 && data[0] <= 0x36) || (data[0] >= 0x40 && data[0] <= 0x44))

	if len(data) > 0 && (data[0] == 0x40 || data[0] == 0x41 || data[0] == 0x42 || data[0] == 0x43) {
		log.Printf("[DEBUG] Got document opcode: 0x%02x, isBinary=%v", data[0], isBinary)
	}

	if isBinary {
		c.processRequestBinary(data, startTime)
	} else {
		c.processRequestText(data, startTime)
	}
}

// processRequestText handles text protocol (existing)
func (c *Connection) processRequestText(data []byte, startTime time.Time) {
	// Parse command (handles both single and batch operations)
	cmd, key, value, keys, values, kvPairs, err := c.parseCommandExtended(data)
	if err != nil {
		c.sendError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// Decide: fast path or slow path?
	if c.shouldUseFastPath(cmd) {
		// Fast path: process inline (NATS-style)
		if keys != nil || kvPairs != nil {
			c.processFastPathBatch(cmd, keys, values, kvPairs, startTime)
		} else {
			c.processFastPath(cmd, key, value, startTime)
		}
	} else {
		// Slow path: dispatch to worker pool
		if keys != nil || kvPairs != nil {
			c.processSlowPathBatch(cmd, keys, values, kvPairs, startTime)
		} else {
			c.processSlowPath(cmd, key, value, startTime)
		}
	}
}

// processRequestBinary handles binary protocol (high performance)
func (c *Connection) processRequestBinary(data []byte, startTime time.Time) {
	// Parse binary request
	req, err := protocol.DecodeRequest(data)
	if err != nil {
		log.Printf("[BINARY] Decode error: %v", err)
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// log.Printf("[BINARY] Opcode: 0x%02x", req.OpCode)

	// Process based on opcode
	switch req.OpCode {
	case protocol.OpSet:
		c.processBinarySet(req, startTime)
	case protocol.OpGet:
		c.processBinaryGet(req, startTime)
	case protocol.OpDel:
		c.processBinaryDel(req, startTime)
	case protocol.OpMSet:
		c.processBinaryMSet(req, startTime)
	case protocol.OpMGet:
		c.processBinaryMGet(req, startTime)
	case protocol.OpMDel:
		c.processBinaryMDel(req, startTime)
	case protocol.OpQPush:
		c.processBinaryQPush(req, startTime)
	case protocol.OpQPop:
		c.processBinaryQPop(req, startTime)
	case protocol.OpQPeek:
		c.processBinaryQPeek(req, startTime)
	case protocol.OpQLen:
		c.processBinaryQLen(req, startTime)
	case protocol.OpQClear:
		c.processBinaryQClear(req, startTime)
	case protocol.OpSPublish:
		c.processBinarySPublish(req, startTime)
	case protocol.OpSConsume:
		c.processBinarySConsume(req, startTime)
	case protocol.OpSCommit:
		c.processBinarySCommit(req, startTime)
	case protocol.OpSCreateTopic:
		c.processBinarySCreateTopic(req, startTime)
	case protocol.OpSPublishBatch:
		c.processBinarySPublishBatch(req, startTime)
	case protocol.OpSSubscribe:
		c.processBinarySSubscribe(req, startTime)
	case protocol.OpSUnsubscribe:
		c.processBinarySUnsubscribe(req, startTime)
	case protocol.OpDocInsert:
		// log.Printf("[BINARY] Routing to DocInsert handler")
		c.processBinaryDocInsert(req, startTime)
	case protocol.OpDocFind:
		// log.Printf("[BINARY] Routing to DocFind handler")
		c.processBinaryDocFind(req, startTime)
	case protocol.OpDocUpdate:
		// log.Printf("[BINARY] Routing to DocUpdate handler")
		c.processBinaryDocUpdate(req, startTime)
	case protocol.OpDocDelete:
		// log.Printf("[BINARY] Routing to DocDelete handler")
		c.processBinaryDocDelete(req, startTime)
	case protocol.OpDocIndex:
		// log.Printf("[BINARY] Routing to DocIndex handler")
		c.processBinaryDocIndex(req, startTime)
	default:
		log.Printf("[BINARY] Unknown opcode: 0x%02x", req.OpCode)
		c.sendBinaryError(fmt.Errorf("unknown opcode"))
		c.server.opsErrors.Add(1)
	}
}

// Binary operation handlers (fast path - inline processing)

// parseCommandExtended parses both single and batch operations
func (c *Connection) parseCommandExtended(data []byte) (cmd, key string, value []byte, keys []string, values [][]byte, kvPairs map[string][]byte, err error) {
	// Parse command parts
	parts := make([][]byte, 0, 10)
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

	if len(parts) < 1 {
		return "", "", nil, nil, nil, nil, fmt.Errorf("invalid command")
	}

	cmd = string(parts[0])

	// Handle batch operations
	switch cmd {
	case "MSET":
		// MSET key1 val1 key2 val2 ...
		if len(parts) < 3 || len(parts)%2 == 0 {
			return "", "", nil, nil, nil, nil, fmt.Errorf("MSET requires pairs of keys and values")
		}
		keys = make([]string, 0, (len(parts)-1)/2)
		values = make([][]byte, 0, (len(parts)-1)/2)
		for i := 1; i < len(parts); i += 2 {
			keys = append(keys, string(parts[i]))
			values = append(values, parts[i+1])
		}
		return cmd, "", nil, keys, values, nil, nil

	case "MGET", "MDEL":
		// MGET key1 key2 key3 ...
		if len(parts) < 2 {
			return "", "", nil, nil, nil, nil, fmt.Errorf("%s requires at least one key", cmd)
		}
		keys = make([]string, 0, len(parts)-1)
		for i := 1; i < len(parts); i++ {
			keys = append(keys, string(parts[i]))
		}
		return cmd, "", nil, keys, nil, nil, nil

	default:
		// Single operation
		if len(parts) < 2 {
			return "", "", nil, nil, nil, nil, fmt.Errorf("invalid command")
		}
		key = string(parts[1])
		if len(parts) > 2 {
			value = parts[2]
		}
		return cmd, key, value, nil, nil, nil, nil
	}
}

// parseCommand parses Redis-style protocol (simple version) - kept for compatibility
func (c *Connection) parseCommand(data []byte) (cmd, key string, value []byte, err error) {
	cmdRet, keyRet, valueRet, _, _, _, errRet := c.parseCommandExtended(data)
	return cmdRet, keyRet, valueRet, errRet
}

// formatBulkString formats response as Redis bulk string
func formatBulkString(data []byte) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(data), data))
}

// formatBulkStrings formats multiple responses as Redis array
func formatBulkStrings(dataList [][]byte) []byte {
	result := fmt.Sprintf("*%d\r\n", len(dataList))
	for _, data := range dataList {
		result += fmt.Sprintf("$%d\r\n%s\r\n", len(data), data)
	}
	return []byte(result)
}

// formatError formats error response
func formatError(err error) []byte {
	return []byte(fmt.Sprintf("-ERR %s\r\n", err.Error()))
}

// sendError sends error to client
func (c *Connection) sendError(err error) {
	select {
	case c.outQueue <- formatError(err):
	case <-c.ctx.Done():
	default:
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.cancel()

	// Close job queue
	close(s.jobQueue)

	// Wait for workers to finish
	s.workerPool.wg.Wait()

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
func (s *Server) Stats() map[string]interface{} {
	return map[string]interface{}{
		"active_connections": s.activeConns.Load(),
		"ops_processed":      s.opsProcessed.Load(),
		"ops_fast_path":      s.opsFastPath.Load(),
		"ops_slow_path":      s.opsSlowPath.Load(),
		"ops_errors":         s.opsErrors.Load(),
		"worker_pool_size":   s.workerPool.workers,
		"active_workers":     s.workerPool.activeWorkers.Load(),
		"jobs_processed":     s.workerPool.jobsProcessed.Load(),
		"job_queue_len":      len(s.jobQueue),
		"job_queue_cap":      cap(s.jobQueue),
	}
}

// GetKVStore returns the KV store instance
func (s *Server) GetKVStore() *kv.KVStore {
	return s.store
}

// GetQueue returns the queue instance
func (s *Server) GetQueue() *queue.Queue {
	return s.queue
}
