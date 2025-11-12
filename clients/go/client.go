package flin

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/skshohagmiah/flin/internal/net"
	"github.com/skshohagmiah/flin/internal/protocol"
)

// Client is a smart, cluster-aware Flin KV client with automatic topology discovery
type Client struct {
	topology       *ClusterTopology
	pools          map[string]*net.ConnectionPool // nodeID -> pool
	httpAddrs      []string                       // HTTP addresses for topology discovery
	poolOpts       *net.PoolOptions
	mu             sync.RWMutex
	refreshTicker  *time.Ticker
	stopRefresh    chan struct{}
	partitionCount int
	clusterMode    bool // true if using cluster discovery
}

// ClusterTopology represents the cluster state
type ClusterTopology struct {
	Nodes        []Node
	PartitionMap map[int]*Partition
	mu           sync.RWMutex
	lastUpdate   time.Time
}

// Node represents a cluster node
type Node struct {
	ID      string
	Address string // KV server address (e.g., "localhost:7380")
}

// Partition represents a partition assignment
type Partition struct {
	ID           int
	PrimaryNode  string
	ReplicaNodes []string
}

// ClientOptions for creating a new client
type ClientOptions struct {
	// For single-node mode: direct KV server address
	Address string

	// For cluster mode: HTTP addresses for topology discovery
	HTTPAddresses []string

	// Connection pool settings per node
	MinConnectionsPerNode int
	MaxConnectionsPerNode int

	// Connection timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Topology refresh interval (only for cluster mode)
	RefreshInterval time.Duration

	// Number of partitions (default: 64)
	PartitionCount int
}

// DefaultOptions returns default client options for single-node
func DefaultOptions(address string) *ClientOptions {
	return &ClientOptions{
		Address:               address,
		MinConnectionsPerNode: 5,
		MaxConnectionsPerNode: 50,
		DialTimeout:           5 * time.Second,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		PartitionCount:        64,
	}
}

// DefaultClusterOptions returns default client options for cluster mode
func DefaultClusterOptions(httpAddresses []string) *ClientOptions {
	return &ClientOptions{
		HTTPAddresses:         httpAddresses,
		MinConnectionsPerNode: 2,
		MaxConnectionsPerNode: 10,
		DialTimeout:           5 * time.Second,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		RefreshInterval:       30 * time.Second,
		PartitionCount:        64,
	}
}

// NewClient creates a new smart Flin client
// If HTTPAddresses are provided, it runs in cluster-aware mode with topology discovery
// If only Address is provided, it connects to a single node
func NewClient(opts *ClientOptions) (*Client, error) {
	if opts == nil {
		return nil, errors.New("options cannot be nil")
	}

	client := &Client{
		topology: &ClusterTopology{
			PartitionMap: make(map[int]*Partition),
		},
		pools:          make(map[string]*net.ConnectionPool),
		stopRefresh:    make(chan struct{}),
		partitionCount: opts.PartitionCount,
		poolOpts: &net.PoolOptions{
			MinSize:      opts.MinConnectionsPerNode,
			MaxSize:      opts.MaxConnectionsPerNode,
			DialTimeout:  opts.DialTimeout,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
			MaxIdleTime:  5 * time.Minute,
			BufferSize:   65536,
		},
	}

	// Determine mode: cluster or single-node
	if len(opts.HTTPAddresses) > 0 {
		// Cluster mode with topology discovery
		client.clusterMode = true
		client.httpAddrs = opts.HTTPAddresses

		// Initial topology discovery
		if err := client.refreshTopology(); err != nil {
			return nil, fmt.Errorf("failed to discover cluster topology: %w", err)
		}

		// Start background topology refresh
		if opts.RefreshInterval > 0 {
			client.refreshTicker = time.NewTicker(opts.RefreshInterval)
			go client.topologyRefreshLoop()
		}
	} else if opts.Address != "" {
		// Single-node mode
		client.clusterMode = false
		client.poolOpts.Address = opts.Address

		// Create single pool
		pool, err := net.NewConnectionPool(client.poolOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection pool: %w", err)
		}

		// Store with a dummy node ID
		client.pools["single-node"] = pool

		// Create minimal topology for single node
		client.topology.Nodes = []Node{{ID: "single-node", Address: opts.Address}}
		for i := 0; i < opts.PartitionCount; i++ {
			client.topology.PartitionMap[i] = &Partition{
				ID:          i,
				PrimaryNode: "single-node",
			}
		}
	} else {
		return nil, errors.New("either Address or HTTPAddresses must be provided")
	}

	return client, nil
}

// topologyRefreshLoop periodically refreshes the cluster topology
func (c *Client) topologyRefreshLoop() {
	for {
		select {
		case <-c.refreshTicker.C:
			c.refreshTopology()
		case <-c.stopRefresh:
			return
		}
	}
}

// refreshTopology queries the cluster for current topology
func (c *Client) refreshTopology() error {
	var lastErr error

	// Try each HTTP address until one succeeds
	for _, httpAddr := range c.httpAddrs {
		url := fmt.Sprintf("http://%s/cluster", httpAddr)
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// Debug: Print raw response (first 500 chars)
		if len(body) > 0 && len(body) < 2000 {
			fmt.Printf("[DEBUG] Cluster response: %s\n", string(body))
		}

		// Parse cluster response - try multiple formats
		var apiResponse struct {
			Cluster struct {
				Nodes []struct {
					ID string `json:"id"`
					IP string `json:"ip"`
				} `json:"nodes"`
				// Try both possible partition formats
				Partitions map[string]struct {
					ID           string   `json:"id"`
					PrimaryNode  string   `json:"primary_node"`
					ReplicaNodes []string `json:"replica_nodes"`
				} `json:"partitions"`
				PartitionMap struct {
					Partitions map[string]struct {
						ID           string   `json:"id"`
						PrimaryNode  string   `json:"primary_node"`
						ReplicaNodes []string `json:"replica_nodes"`
					} `json:"partitions"`
				} `json:"partition_map"`
			} `json:"cluster"`
		}

		if err := json.Unmarshal(body, &apiResponse); err != nil {
			lastErr = err
			continue
		}

		// Update topology
		c.topology.mu.Lock()

		// Update nodes
		c.topology.Nodes = make([]Node, 0, len(apiResponse.Cluster.Nodes))
		for _, n := range apiResponse.Cluster.Nodes {
			// ClusterKit returns HTTP addresses (e.g., "node1:8080")
			// Convert to KV server address by replacing port
			// Convention: KV port = HTTP port - 2000
			// 8080 -> 6380, 8081 -> 6381, 8082 -> 6382
			kvAddr := n.IP
			kvAddr = strings.Replace(kvAddr, ":8080", ":6380", 1)
			kvAddr = strings.Replace(kvAddr, ":8081", ":6381", 1)
			kvAddr = strings.Replace(kvAddr, ":8082", ":6382", 1)

			c.topology.Nodes = append(c.topology.Nodes, Node{
				ID:      n.ID,
				Address: kvAddr,
			})
		}

		// Update partition map
		c.topology.PartitionMap = make(map[int]*Partition)

		// Try both possible partition locations
		partitions := apiResponse.Cluster.Partitions
		if len(partitions) == 0 {
			partitions = apiResponse.Cluster.PartitionMap.Partitions
		}

		fmt.Printf("[DEBUG] Found %d partitions in response\n", len(partitions))

		for partID, part := range partitions {
			var id int
			// Try multiple formats: "0", "partition-0", "partition_0"
			if _, err := fmt.Sscanf(partID, "%d", &id); err != nil {
				if _, err := fmt.Sscanf(partID, "partition-%d", &id); err != nil {
					fmt.Sscanf(partID, "partition_%d", &id)
				}
			}

			c.topology.PartitionMap[id] = &Partition{
				ID:           id,
				PrimaryNode:  part.PrimaryNode,
				ReplicaNodes: part.ReplicaNodes,
			}
		}

		fmt.Printf("[DEBUG] Parsed %d partitions into topology map\n", len(c.topology.PartitionMap))

		c.topology.lastUpdate = time.Now()
		c.topology.mu.Unlock()

		// Create connection pools for new nodes
		c.ensureConnectionPools()

		return nil
	}

	return fmt.Errorf("failed to refresh topology from all addresses: %w", lastErr)
}

// ensureConnectionPools creates connection pools for all nodes
func (c *Client) ensureConnectionPools() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.topology.mu.RLock()
	defer c.topology.mu.RUnlock()

	for _, node := range c.topology.Nodes {
		if _, exists := c.pools[node.ID]; !exists {
			opts := *c.poolOpts
			opts.Address = node.Address
			pool, err := net.NewConnectionPool(&opts)
			if err == nil {
				c.pools[node.ID] = pool
			}
		}
	}
}

// getPartitionForKey calculates which partition a key belongs to
func (c *Client) getPartitionForKey(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % uint32(c.partitionCount))
}

// getConnectionForKey returns a connection to the node owning the key's partition
func (c *Client) getConnectionForKey(key string) (*net.Connection, string, error) {
	partitionID := c.getPartitionForKey(key)

	c.topology.mu.RLock()
	partition, exists := c.topology.PartitionMap[partitionID]
	c.topology.mu.RUnlock()

	if !exists {
		return nil, "", fmt.Errorf("partition %d not found in topology", partitionID)
	}

	// Try primary node first
	c.mu.RLock()
	pool, exists := c.pools[partition.PrimaryNode]
	c.mu.RUnlock()

	if exists {
		conn, err := pool.Get()
		if err == nil {
			return conn, partition.PrimaryNode, nil
		}
	}

	// Fallback to replicas if primary fails
	for _, replicaNode := range partition.ReplicaNodes {
		c.mu.RLock()
		pool, exists := c.pools[replicaNode]
		c.mu.RUnlock()

		if exists {
			conn, err := pool.Get()
			if err == nil {
				return conn, replicaNode, nil
			}
		}
	}

	return nil, "", fmt.Errorf("no available node for partition %d", partitionID)
}

// releaseConnection returns a connection to its pool
func (c *Client) releaseConnection(nodeID string, conn *net.Connection) {
	c.mu.RLock()
	pool, exists := c.pools[nodeID]
	c.mu.RUnlock()

	if exists {
		pool.Put(conn)
	}
}

// Set stores a key-value pair using smart routing with replication
func (c *Client) Set(key string, value []byte) error {
	partitionID := c.getPartitionForKey(key)

	c.topology.mu.RLock()
	partition, exists := c.topology.PartitionMap[partitionID]
	c.topology.mu.RUnlock()

	if !exists {
		return fmt.Errorf("partition %d not found in topology", partitionID)
	}

	// Get all nodes (primary + replicas)
	nodes := []string{partition.PrimaryNode}
	nodes = append(nodes, partition.ReplicaNodes...)

	// Write to all nodes in parallel
	errChan := make(chan error, len(nodes))
	var wg sync.WaitGroup

	for _, nodeID := range nodes {
		wg.Add(1)
		go func(nid string) {
			defer wg.Done()

			c.mu.RLock()
			pool, exists := c.pools[nid]
			c.mu.RUnlock()

			if !exists {
				errChan <- fmt.Errorf("pool not found for node %s", nid)
				return
			}

			conn, err := pool.Get()
			if err != nil {
				errChan <- err
				return
			}
			defer pool.Put(conn)

			request := protocol.EncodeSetRequest(key, value)
			if err := conn.Write(request); err != nil {
				errChan <- err
				return
			}

			if err := readOKResponse(conn); err != nil {
				errChan <- err
			}
		}(nodeID)
	}

	wg.Wait()
	close(errChan)

	// Check for errors - succeed if at least primary succeeded
	var lastErr error
	errorCount := 0
	for err := range errChan {
		if err != nil {
			lastErr = err
			errorCount++
		}
	}

	// Require at least quorum (majority) to succeed
	successCount := len(nodes) - errorCount
	quorum := (len(nodes) / 2) + 1
	if successCount >= quorum {
		return nil
	}

	return fmt.Errorf("write failed: only %d/%d nodes succeeded: %v", successCount, len(nodes), lastErr)
}

// Get retrieves a value by key using smart routing
func (c *Client) Get(key string) ([]byte, error) {
	conn, nodeID, err := c.getConnectionForKey(key)
	if err != nil {
		return nil, err
	}
	defer c.releaseConnection(nodeID, conn)

	request := protocol.EncodeGetRequest(key)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	return readValueResponse(conn)
}

// Delete removes a key using smart routing with replication
func (c *Client) Delete(key string) error {
	partitionID := c.getPartitionForKey(key)

	c.topology.mu.RLock()
	partition, exists := c.topology.PartitionMap[partitionID]
	c.topology.mu.RUnlock()

	if !exists {
		return fmt.Errorf("partition %d not found in topology", partitionID)
	}

	// Get all nodes (primary + replicas)
	nodes := []string{partition.PrimaryNode}
	nodes = append(nodes, partition.ReplicaNodes...)

	// Delete from all nodes in parallel
	errChan := make(chan error, len(nodes))
	var wg sync.WaitGroup

	for _, nodeID := range nodes {
		wg.Add(1)
		go func(nid string) {
			defer wg.Done()

			c.mu.RLock()
			pool, exists := c.pools[nid]
			c.mu.RUnlock()

			if !exists {
				errChan <- fmt.Errorf("pool not found for node %s", nid)
				return
			}

			conn, err := pool.Get()
			if err != nil {
				errChan <- err
				return
			}
			defer pool.Put(conn)

			request := protocol.EncodeDeleteRequest(key)
			if err := conn.Write(request); err != nil {
				errChan <- err
				return
			}

			if err := readOKResponse(conn); err != nil {
				errChan <- err
			}
		}(nodeID)
	}

	wg.Wait()
	close(errChan)

	// Check for errors - succeed if at least quorum succeeded
	var lastErr error
	errorCount := 0
	for err := range errChan {
		if err != nil {
			lastErr = err
			errorCount++
		}
	}

	successCount := len(nodes) - errorCount
	quorum := (len(nodes) / 2) + 1
	if successCount >= quorum {
		return nil
	}

	return fmt.Errorf("delete failed: only %d/%d nodes succeeded: %v", successCount, len(nodes), lastErr)
}

// Exists checks if a key exists using smart routing
func (c *Client) Exists(key string) (bool, error) {
	conn, nodeID, err := c.getConnectionForKey(key)
	if err != nil {
		return false, err
	}
	defer c.releaseConnection(nodeID, conn)

	request := protocol.EncodeExistsRequest(key)
	if err := conn.Write(request); err != nil {
		return false, err
	}

	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return false, err
	}

	if status == protocol.StatusOK && payloadLen > 0 {
		payload, err := conn.Read(int(payloadLen))
		if err != nil {
			return false, err
		}
		return payload[0] == 1, nil
	}

	return false, nil
}

// Incr increments a counter using smart routing
func (c *Client) Incr(key string) (int64, error) {
	conn, nodeID, err := c.getConnectionForKey(key)
	if err != nil {
		return 0, err
	}
	defer c.releaseConnection(nodeID, conn)

	request := protocol.EncodeIncrRequest(key)
	if err := conn.Write(request); err != nil {
		return 0, err
	}

	value, err := readValueResponse(conn)
	if err != nil {
		return 0, err
	}

	if len(value) != 8 {
		return 0, errors.New("invalid counter value")
	}

	return int64(binary.BigEndian.Uint64(value)), nil
}

// Decr decrements a counter using smart routing
func (c *Client) Decr(key string) (int64, error) {
	conn, nodeID, err := c.getConnectionForKey(key)
	if err != nil {
		return 0, err
	}
	defer c.releaseConnection(nodeID, conn)

	request := protocol.EncodeDecrRequest(key)
	if err := conn.Write(request); err != nil {
		return 0, err
	}

	value, err := readValueResponse(conn)
	if err != nil {
		return 0, err
	}

	if len(value) != 8 {
		return 0, errors.New("invalid counter value")
	}

	return int64(binary.BigEndian.Uint64(value)), nil
}

// MSet performs a batch set operation (routes to multiple nodes in cluster mode)
func (c *Client) MSet(keys []string, values [][]byte) error {
	if len(keys) != len(values) {
		return fmt.Errorf("keys and values length mismatch")
	}

	// Group keys by node
	nodeKeys := make(map[string][]int) // nodeID -> indices
	for i, key := range keys {
		partitionID := c.getPartitionForKey(key)

		c.topology.mu.RLock()
		partition, exists := c.topology.PartitionMap[partitionID]
		c.topology.mu.RUnlock()

		if !exists {
			return fmt.Errorf("partition %d not found", partitionID)
		}

		nodeKeys[partition.PrimaryNode] = append(nodeKeys[partition.PrimaryNode], i)
	}

	// Send batch requests to each node in parallel
	errChan := make(chan error, len(nodeKeys))
	var wg sync.WaitGroup

	for nodeID, indices := range nodeKeys {
		wg.Add(1)
		go func(nid string, idxs []int) {
			defer wg.Done()

			batchKeys := make([]string, len(idxs))
			batchValues := make([][]byte, len(idxs))
			for i, idx := range idxs {
				batchKeys[i] = keys[idx]
				batchValues[i] = values[idx]
			}

			c.mu.RLock()
			pool, exists := c.pools[nid]
			c.mu.RUnlock()

			if !exists {
				errChan <- fmt.Errorf("pool not found for node %s", nid)
				return
			}

			conn, err := pool.Get()
			if err != nil {
				errChan <- err
				return
			}
			defer pool.Put(conn)

			request := protocol.EncodeMSetRequest(batchKeys, batchValues)
			if err := conn.Write(request); err != nil {
				errChan <- err
				return
			}

			if err := readOKResponse(conn); err != nil {
				errChan <- err
			}
		}(nodeID, indices)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// MGet performs a batch get operation (routes to multiple nodes in cluster mode)
func (c *Client) MGet(keys []string) ([][]byte, error) {
	// Group keys by node
	nodeKeys := make(map[string][]int) // nodeID -> indices
	for i, key := range keys {
		partitionID := c.getPartitionForKey(key)

		c.topology.mu.RLock()
		partition, exists := c.topology.PartitionMap[partitionID]
		c.topology.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("partition %d not found", partitionID)
		}

		nodeKeys[partition.PrimaryNode] = append(nodeKeys[partition.PrimaryNode], i)
	}

	// Result slice
	results := make([][]byte, len(keys))
	var mu sync.Mutex

	// Send batch requests to each node in parallel
	errChan := make(chan error, len(nodeKeys))
	var wg sync.WaitGroup

	for nodeID, indices := range nodeKeys {
		wg.Add(1)
		go func(nid string, idxs []int) {
			defer wg.Done()

			batchKeys := make([]string, len(idxs))
			for i, idx := range idxs {
				batchKeys[i] = keys[idx]
			}

			c.mu.RLock()
			pool, exists := c.pools[nid]
			c.mu.RUnlock()

			if !exists {
				errChan <- fmt.Errorf("pool not found for node %s", nid)
				return
			}

			conn, err := pool.Get()
			if err != nil {
				errChan <- err
				return
			}
			defer pool.Put(conn)

			request := protocol.EncodeMGetRequest(batchKeys)
			if err := conn.Write(request); err != nil {
				errChan <- err
				return
			}

			values, err := readMultiValueResponse(conn)
			if err != nil {
				errChan <- err
				return
			}

			// Store results in correct positions
			mu.Lock()
			for i, idx := range idxs {
				results[idx] = values[i]
			}
			mu.Unlock()
		}(nodeID, indices)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// MDelete performs a batch delete operation (routes to multiple nodes in cluster mode)
func (c *Client) MDelete(keys []string) error {
	// Group keys by node
	nodeKeys := make(map[string][]int) // nodeID -> indices
	for i, key := range keys {
		partitionID := c.getPartitionForKey(key)

		c.topology.mu.RLock()
		partition, exists := c.topology.PartitionMap[partitionID]
		c.topology.mu.RUnlock()

		if !exists {
			return fmt.Errorf("partition %d not found", partitionID)
		}

		nodeKeys[partition.PrimaryNode] = append(nodeKeys[partition.PrimaryNode], i)
	}

	// Send batch requests to each node in parallel
	errChan := make(chan error, len(nodeKeys))
	var wg sync.WaitGroup

	for nodeID, indices := range nodeKeys {
		wg.Add(1)
		go func(nid string, idxs []int) {
			defer wg.Done()

			batchKeys := make([]string, len(idxs))
			for i, idx := range idxs {
				batchKeys[i] = keys[idx]
			}

			c.mu.RLock()
			pool, exists := c.pools[nid]
			c.mu.RUnlock()

			if !exists {
				errChan <- fmt.Errorf("pool not found for node %s", nid)
				return
			}

			conn, err := pool.Get()
			if err != nil {
				errChan <- err
				return
			}
			defer pool.Put(conn)

			request := protocol.EncodeMDeleteRequest(batchKeys)
			if err := conn.Write(request); err != nil {
				errChan <- err
				return
			}

			if err := readOKResponse(conn); err != nil {
				errChan <- err
			}
		}(nodeID, indices)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTopology returns the current cluster topology
func (c *Client) GetTopology() *ClusterTopology {
	c.topology.mu.RLock()
	defer c.topology.mu.RUnlock()

	// Return a copy
	topology := &ClusterTopology{
		Nodes:        make([]Node, len(c.topology.Nodes)),
		PartitionMap: make(map[int]*Partition),
		lastUpdate:   c.topology.lastUpdate,
	}

	copy(topology.Nodes, c.topology.Nodes)
	for k, v := range c.topology.PartitionMap {
		topology.PartitionMap[k] = v
	}

	return topology
}

// IsClusterMode returns true if client is running in cluster mode
func (c *Client) IsClusterMode() bool {
	return c.clusterMode
}

// Close closes all connections and stops topology refresh
func (c *Client) Close() error {
	// Stop refresh loop if running
	if c.refreshTicker != nil {
		close(c.stopRefresh)
		c.refreshTicker.Stop()
	}

	// Close all connection pools
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, pool := range c.pools {
		pool.Close()
	}

	return nil
}

// Helper functions for reading responses

func readOKResponse(conn *net.Connection) error {
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return err
	}

	if status == protocol.StatusOK {
		return nil
	}

	if status == protocol.StatusError && payloadLen > 0 {
		errMsg, err := conn.Read(int(payloadLen))
		if err != nil {
			return err
		}
		return errors.New(string(errMsg))
	}

	return fmt.Errorf("operation failed: status %d", status)
}

func readValueResponse(conn *net.Connection) ([]byte, error) {
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return nil, err
	}

	if status != protocol.StatusOK {
		if status == protocol.StatusNotFound {
			return nil, errors.New("key not found")
		}
		if status == protocol.StatusError && payloadLen > 0 {
			errMsg, err := conn.Read(int(payloadLen))
			if err != nil {
				return nil, err
			}
			return nil, errors.New(string(errMsg))
		}
		return nil, fmt.Errorf("get failed: status %d", status)
	}

	if payloadLen == 0 {
		return nil, nil
	}

	return conn.Read(int(payloadLen))
}

func readMultiValueResponse(conn *net.Connection) ([][]byte, error) {
	status, payloadLen, err := conn.ReadHeader()
	if err != nil {
		return nil, err
	}

	if status != protocol.StatusMultiValue {
		return nil, fmt.Errorf("expected multi-value response, got status %d", status)
	}

	payload, err := conn.Read(int(payloadLen))
	if err != nil {
		return nil, err
	}

	// Parse multi-value response
	if len(payload) < 2 {
		return nil, errors.New("invalid multi-value response")
	}

	count := binary.BigEndian.Uint16(payload[0:2])
	values := make([][]byte, 0, count)
	pos := 2

	for i := 0; i < int(count); i++ {
		if len(payload) < pos+4 {
			return nil, errors.New("invalid multi-value response")
		}

		valueLen := binary.BigEndian.Uint32(payload[pos:])
		pos += 4

		if len(payload) < pos+int(valueLen) {
			return nil, errors.New("invalid multi-value response")
		}

		value := make([]byte, valueLen)
		copy(value, payload[pos:pos+int(valueLen)])
		pos += int(valueLen)

		values = append(values, value)
	}

	return values, nil
}
