package server

import (
	"encoding/binary"
	"fmt"
	"time"

	protocol "github.com/skshohagmiah/flin/internal/net"
)

// KV operation handlers for binary protocol
func (c *Connection) processBinarySet(req *protocol.Request, startTime time.Time) {
	err := c.server.store.Set(req.Key, req.Value, 0)

	var response []byte
	if err != nil {
		response = protocol.EncodeErrorResponse(err)
	} else {
		response = protocol.EncodeOKResponse()
	}

	c.sendBinaryResponse(response, startTime)
}

func (c *Connection) processBinaryGet(req *protocol.Request, startTime time.Time) {
	val, err := c.server.store.Get(req.Key)

	var response []byte
	if err != nil {
		response = protocol.EncodeErrorResponse(err)
	} else {
		response = protocol.EncodeValueResponse(val)
	}

	c.sendBinaryResponse(response, startTime)
}

func (c *Connection) processBinaryDel(req *protocol.Request, startTime time.Time) {
	err := c.server.store.Delete(req.Key)

	var response []byte
	if err != nil {
		response = protocol.EncodeErrorResponse(err)
	} else {
		response = protocol.EncodeOKResponse()
	}

	c.sendBinaryResponse(response, startTime)
}

func (c *Connection) processBinaryMSet(req *protocol.Request, startTime time.Time) {
	// Use atomic batch set
	kvPairs := make(map[string][]byte, len(req.Keys))
	for i, key := range req.Keys {
		kvPairs[key] = req.Values[i]
	}

	err := c.server.store.BatchSet(kvPairs, 0)

	var response []byte
	if err != nil {
		response = protocol.EncodeErrorResponse(err)
	} else {
		response = protocol.EncodeOKResponse()
	}

	c.sendBinaryResponse(response, startTime)
}

func (c *Connection) processBinaryMGet(req *protocol.Request, startTime time.Time) {
	results, err := c.server.store.BatchGet(req.Keys)

	var response []byte
	if err != nil {
		response = protocol.EncodeErrorResponse(err)
	} else {
		// Convert map to ordered slice
		values := make([][]byte, 0, len(req.Keys))
		for _, key := range req.Keys {
			if val, ok := results[key]; ok {
				values = append(values, val)
			} else {
				values = append(values, []byte{})
			}
		}
		response = protocol.EncodeMultiValueResponse(values)
	}

	c.sendBinaryResponse(response, startTime)
}

func (c *Connection) processBinaryMDel(req *protocol.Request, startTime time.Time) {
	err := c.server.store.BatchDelete(req.Keys)

	var response []byte
	if err != nil {
		response = protocol.EncodeErrorResponse(err)
	} else {
		response = protocol.EncodeOKResponse()
	}

	c.sendBinaryResponse(response, startTime)
}

func (c *Connection) sendBinaryResponse(response []byte, startTime time.Time) {
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
		c.server.opsFastPath.Add(1)
		c.opsProcessed.Add(1)

		latency := time.Since(startTime)
		c.updateAvgLatency(latency)
	case <-c.ctx.Done():
		return
	default:
		c.server.opsErrors.Add(1)
	}
}

func (c *Connection) sendBinaryError(err error) {
	response := protocol.EncodeErrorResponse(err)
	select {
	case c.outQueue <- response:
	case <-c.ctx.Done():
	default:
	}
}

// shouldUseFastPath determines if operation should use fast path
func (c *Connection) shouldUseFastPath(cmd string) bool {
	// Simple operations go to fast path
	switch cmd {
	case "GET", "EXISTS", "MGET":
		// Read operations are typically fast (cache hits)
		return true
	case "SET", "DEL":
		// Check connection's average latency
		avgLatency := time.Duration(c.avgLatency.Load())
		if avgLatency > FastPathThreshold {
			// This connection is experiencing slow operations
			return false
		}
		return true
	case "INCR", "DECR":
		// Atomic operations, usually fast
		return true
	case "MSET", "MDEL":
		// Batch operations go to slow path (more work)
		return false
	default:
		// Unknown commands go to slow path
		return false
	}
}

// processFastPath handles operations inline (NATS-style)
func (c *Connection) processFastPath(cmd, key string, value []byte, startTime time.Time) {
	var response []byte

	switch cmd {
	case "SET":
		err := c.server.store.Set(key, value, 0)
		if err != nil {
			response = formatError(err)
		} else {
			response = []byte("+OK\r\n")
		}

	case "GET":
		val, err := c.server.store.Get(key)
		if err != nil {
			response = formatError(err)
		} else {
			response = formatBulkString(val)
		}

	case "DEL":
		err := c.server.store.Delete(key)
		if err != nil {
			response = formatError(err)
		} else {
			response = []byte("+OK\r\n")
		}

	case "EXISTS":
		exists, err := c.server.store.Exists(key)
		if err != nil {
			response = formatError(err)
		} else if exists {
			response = []byte(":1\r\n")
		} else {
			response = []byte(":0\r\n")
		}

	case "INCR":
		val, err := c.server.store.Incr(key)
		if err != nil {
			response = formatError(err)
		} else {
			// Return the new value as a bulk string
			var buf [8]byte
			binary.BigEndian.PutUint64(buf[:], uint64(val))
			response = formatBulkString(buf[:])
		}

	case "DECR":
		val, err := c.server.store.Decr(key)
		if err != nil {
			response = formatError(err)
		} else {
			// Return the new value as a bulk string
			var buf [8]byte
			binary.BigEndian.PutUint64(buf[:], uint64(val))
			response = formatBulkString(buf[:])
		}

	default:
		response = []byte("-ERR unknown command\r\n")
	}

	// Send response
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
		c.server.opsFastPath.Add(1)
		c.opsProcessed.Add(1)

		// Update average latency
		latency := time.Since(startTime)
		c.updateAvgLatency(latency)
	case <-c.ctx.Done():
		return
	default:
		// Queue full
		c.server.opsErrors.Add(1)
	}
}

// processSlowPath dispatches operation to worker pool
func (c *Connection) processSlowPath(cmd, key string, value []byte, startTime time.Time) {
	job := &Job{
		conn:      c,
		cmd:       cmd,
		key:       key,
		value:     value,
		startTime: startTime,
	}

	// Dispatch to worker pool
	select {
	case c.server.jobQueue <- job:
		c.server.opsProcessed.Add(1)
		c.server.opsSlowPath.Add(1)
		c.opsProcessed.Add(1)
	case <-c.ctx.Done():
		return
	default:
		// Queue full - apply backpressure
		c.sendError(fmt.Errorf("server busy"))
		c.server.opsErrors.Add(1)
	}
}

// processFastPathBatch handles batch operations inline (NATS-style)
func (c *Connection) processFastPathBatch(cmd string, keys []string, values [][]byte, kvPairs map[string][]byte, startTime time.Time) {
	var response []byte

	switch cmd {
	case "MSET":
		// Convert to map for BatchSet
		kvPairs := make(map[string][]byte, len(keys))
		for i, key := range keys {
			kvPairs[key] = values[i]
		}
		err := c.server.store.BatchSet(kvPairs, 0)
		if err != nil {
			c.sendError(err)
			return
		}
		response = []byte("+OK\r\n")

	case "MGET":
		results, err := c.server.store.BatchGet(keys)
		if err != nil {
			c.sendError(err)
			return
		}
		// Convert map to ordered slice
		var resultValues [][]byte
		for _, key := range keys {
			if val, ok := results[key]; ok {
				resultValues = append(resultValues, val)
			} else {
				resultValues = append(resultValues, []byte{})
			}
		}
		response = formatBulkStrings(resultValues)

	case "MDEL":
		err := c.server.store.BatchDelete(keys)
		if err != nil {
			c.sendError(err)
			return
		}
		response = []byte("+OK\r\n")

	default:
		c.sendError(fmt.Errorf("unknown command"))
		return
	}

	// Send response
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
		c.server.opsFastPath.Add(1)
		c.opsProcessed.Add(1)

		// Update average latency
		latency := time.Since(startTime)
		c.updateAvgLatency(latency)
	case <-c.ctx.Done():
		return
	default:
		// Queue full
		c.server.opsErrors.Add(1)
		c.sendError(fmt.Errorf("server busy"))
	}
}

// processSlowPathBatch dispatches batch operation to worker pool
func (c *Connection) processSlowPathBatch(cmd string, keys []string, values [][]byte, kvPairs map[string][]byte, startTime time.Time) {
	job := &Job{
		conn:      c,
		cmd:       cmd,
		keys:      keys,
		values:    values,
		kvPairs:   kvPairs,
		startTime: startTime,
	}

	// Dispatch to worker pool
	select {
	case c.server.jobQueue <- job:
		c.server.opsProcessed.Add(1)
		c.server.opsSlowPath.Add(1)
		c.opsProcessed.Add(1)
	case <-c.ctx.Done():
		return
	default:
		// Queue full
		c.server.opsErrors.Add(1)
		c.sendError(fmt.Errorf("server busy"))
	}
}

// updateAvgLatency updates the exponential moving average of latency
func (c *Connection) updateAvgLatency(latency time.Duration) {
	// Exponential moving average: new_avg = 0.9 * old_avg + 0.1 * new_value
	oldAvg := c.avgLatency.Load()
	newAvg := int64(float64(oldAvg)*0.9 + float64(latency.Nanoseconds())*0.1)
	c.avgLatency.Store(newAvg)
}
