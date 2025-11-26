package flin

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/skshohagmiah/flin/internal/protocol"
)

// KV Operations

// Set stores a key-value pair using smart routing with replication
func (c *Client) Set(key string, value []byte) error {
	partitionID := c.getPartitionForKey(key)

	c.topology.mu.RLock()
	partition, exists := c.topology.PartitionMap[partitionID]
	c.topology.mu.RUnlock()

	if !exists {
		return fmt.Errorf("partition %d not found in topology", partitionID)
	}

	// Fast path: single node (no replicas)
	if len(partition.ReplicaNodes) == 0 {
		c.mu.RLock()
		pool, exists := c.pools[partition.PrimaryNode]
		c.mu.RUnlock()

		if !exists {
			return fmt.Errorf("pool not found for node %s", partition.PrimaryNode)
		}

		conn, err := pool.Get()
		if err != nil {
			return err
		}
		defer pool.Put(conn)

		request := protocol.EncodeSetRequest(key, value)
		if err := conn.Write(request); err != nil {
			return err
		}

		return readOKResponse(conn)
	}

	// Slow path: replicated write (cluster mode)
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
