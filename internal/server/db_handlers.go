package server

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/skshohagmiah/flin/internal/db"
	protocol "github.com/skshohagmiah/flin/internal/net"
)

// Database handler methods on Connection

// processBinaryDocInsert handles document insert operations
func (c *Connection) processBinaryDocInsert(req *protocol.Request, startTime time.Time) {
	log.Printf("[DOC] INSERT start: collection=%s", req.Collection)

	docStore := c.server.db
	if docStore == nil {
		log.Printf("[DOC] INSERT error: document store not available")
		c.sendBinaryError(fmt.Errorf("document store not available"))
		c.server.opsErrors.Add(1)
		return
	}

	// Parse the document JSON
	var doc db.Document
	err := json.Unmarshal(req.Value, &doc)
	if err != nil {
		log.Printf("[DOC] INSERT error: invalid JSON: %v", err)
		c.sendBinaryError(fmt.Errorf("invalid JSON document: %w", err))
		c.server.opsErrors.Add(1)
		return
	}

	// Insert the document
	id, err := docStore.Insert(req.Collection, doc)
	if err != nil {
		log.Printf("[DOC] INSERT error: %v", err)
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[DOC] INSERT complete: collection=%s, id=%s, duration=%v", req.Collection, id, duration)

	// Send response with the generated ID
	response := protocol.EncodeValueResponse([]byte(id))
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
	case <-c.ctx.Done():
	}
}

// processBinaryDocFind handles document find operations
func (c *Connection) processBinaryDocFind(req *protocol.Request, startTime time.Time) {
	log.Printf("[DOC] FIND start: collection=%s", req.Collection)

	docStore := c.server.db
	if docStore == nil {
		log.Printf("[DOC] FIND error: document store not available")
		c.sendBinaryError(fmt.Errorf("document store not available"))
		c.server.opsErrors.Add(1)
		return
	}

	// Parse the query options
	var queryData map[string]interface{}
	err := json.Unmarshal(req.Value, &queryData)
	if err != nil {
		log.Printf("[DOC] FIND error: invalid query: %v", err)
		c.sendBinaryError(fmt.Errorf("invalid query format: %w", err))
		c.server.opsErrors.Add(1)
		return
	}

	// Build FindOptions from query data
	opts := buildFindOptions(queryData)

	// Execute find
	results, err := docStore.Find(req.Collection, opts)
	if err != nil {
		log.Printf("[DOC] FIND error: %v", err)
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[DOC] FIND complete: collection=%s, results=%d, duration=%v", req.Collection, len(results), duration)

	// Encode results as JSON array
	resultJSON, err := json.Marshal(results)
	if err != nil {
		log.Printf("[DOC] FIND error: marshal failed: %v", err)
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// Send response
	response := protocol.EncodeValueResponse(resultJSON)
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
	case <-c.ctx.Done():
	}
}

// processBinaryDocUpdate handles document update operations
func (c *Connection) processBinaryDocUpdate(req *protocol.Request, startTime time.Time) {
	docStore := c.server.db
	if docStore == nil {
		c.sendBinaryError(fmt.Errorf("document store not available"))
		c.server.opsErrors.Add(1)
		return
	}

	// Parse the options (contains both query and update)
	var optsData map[string]interface{}
	err := json.Unmarshal(req.Value, &optsData)
	if err != nil {
		c.sendBinaryError(fmt.Errorf("invalid options format: %w", err))
		c.server.opsErrors.Add(1)
		return
	}

	// Extract update data
	var updateData map[string]interface{}
	if setRaw, ok := optsData["set"]; ok {
		if setMap, ok := setRaw.(map[string]interface{}); ok {
			updateData = setMap
		}
	}

	if updateData == nil {
		c.sendBinaryError(fmt.Errorf("missing 'set' in update options"))
		c.server.opsErrors.Add(1)
		return
	}

	// Find document matching query and update
	// buildFindOptions expects a map with "filters" key, which optsData has
	findOpts := buildFindOptions(optsData)
	results, err := docStore.Find(req.Collection, findOpts)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	var updated int
	for _, doc := range results {
		id, ok := doc["_id"].(string)
		if !ok {
			continue
		}

		updateOpts := db.UpdateOptions{
			Set:   db.Document(updateData),
			Merge: true,
		}
		if err := docStore.Update(req.Collection, id, updateOpts); err == nil {
			updated++
		}
	}

	// Send response with count
	response := protocol.EncodeValueResponse([]byte(fmt.Sprintf(`{"updated":%d}`, updated)))
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
	case <-c.ctx.Done():
	}
}

// processBinaryDocDelete handles document delete operations
func (c *Connection) processBinaryDocDelete(req *protocol.Request, startTime time.Time) {
	docStore := c.server.db
	if docStore == nil {
		c.sendBinaryError(fmt.Errorf("document store not available"))
		c.server.opsErrors.Add(1)
		return
	}

	// Parse the query
	var queryData map[string]interface{}
	err := json.Unmarshal(req.Value, &queryData)
	if err != nil {
		c.sendBinaryError(fmt.Errorf("invalid query format: %w", err))
		c.server.opsErrors.Add(1)
		return
	}

	// Build FindOptions from query data
	opts := buildFindOptions(queryData)

	// Delete documents
	deleted, err := docStore.DeleteMany(req.Collection, opts)
	if err != nil {
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	// Send response with count
	response := protocol.EncodeValueResponse([]byte(fmt.Sprintf(`{"deleted":%d}`, deleted)))
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
	case <-c.ctx.Done():
	}
}

// Helper function to build FindOptions from query data
func buildFindOptions(queryData map[string]interface{}) db.FindOptions {
	opts := db.FindOptions{
		Filters: []db.Query{},
	}

	// Parse filters
	if filtersRaw, ok := queryData["filters"]; ok {
		if filtersArr, ok := filtersRaw.([]interface{}); ok {
			for _, f := range filtersArr {
				if filterMap, ok := f.(map[string]interface{}); ok {
					query := db.Query{
						Field:    toString(filterMap["field"]),
						Operator: toString(filterMap["operator"]),
						Value:    filterMap["value"],
					}
					opts.Filters = append(opts.Filters, query)
				}
			}
		}
	}

	// Parse sort
	if sortRaw, ok := queryData["sort"]; ok {
		if sortMap, ok := sortRaw.(map[string]interface{}); ok {
			opts.Sort = &db.SortOption{
				Field:     toString(sortMap["field"]),
				Direction: toString(sortMap["direction"]),
			}
		}
	}

	// Parse pagination
	if skip, ok := queryData["skip"].(float64); ok {
		opts.Skip = int(skip)
	}
	if limit, ok := queryData["limit"].(float64); ok {
		opts.Limit = int(limit)
	}

	return opts
}

// processBinaryDocIndex handles document index creation operations
func (c *Connection) processBinaryDocIndex(req *protocol.Request, startTime time.Time) {
	log.Printf("[DOC] INDEX start: collection=%s, field=%s", req.Collection, req.Key)

	docStore := c.server.db
	if docStore == nil {
		log.Printf("[DOC] INDEX error: document store not available")
		c.sendBinaryError(fmt.Errorf("document store not available"))
		c.server.opsErrors.Add(1)
		return
	}

	// Create index on the specified field
	err := docStore.CreateIndex(req.Collection, req.Key)
	if err != nil {
		log.Printf("[DOC] INDEX error: %v", err)
		c.sendBinaryError(err)
		c.server.opsErrors.Add(1)
		return
	}

	duration := time.Since(startTime)
	log.Printf("[DOC] INDEX complete: collection=%s, field=%s, duration=%v", req.Collection, req.Key, duration)

	// Send success response
	response := protocol.EncodeValueResponse([]byte("OK"))
	select {
	case c.outQueue <- response:
		c.server.opsProcessed.Add(1)
	case <-c.ctx.Done():
	}
}

func toString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
