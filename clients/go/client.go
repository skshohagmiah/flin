package flin

import (
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

	// Queue client (optional)
	Queue *QueueClient
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

	// Queue server address (optional, default: KV address with port +1)
	QueueAddress string

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
		MinConnectionsPerNode: 128, // Increased from 5 to support high concurrency
		MaxConnectionsPerNode: 512, // Increased from 50 to support 256+ workers
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

	// Initialize Queue client if address provided
	if opts.QueueAddress != "" {
		queueClient, err := NewQueueClient(opts.QueueAddress, client.poolOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create queue client: %w", err)
		}
		client.Queue = queueClient
	} else if opts.Address != "" {
		// Unified server: Queue is on the same port as KV
		queueClient, err := NewQueueClient(opts.Address, client.poolOpts)
		if err == nil {
			client.Queue = queueClient
		}
		// Silently ignore queue client creation errors
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

	// Close queue client if exists
	if c.Queue != nil {
		c.Queue.Close()
	}

	return nil
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
