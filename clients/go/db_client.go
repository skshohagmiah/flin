package flin

import (
	"encoding/json"
	"fmt"

	"github.com/skshohagmiah/flin/internal/net"
	protocol "github.com/skshohagmiah/flin/internal/net"
)

// DBClient handles Document Database operations
type DBClient struct {
	pool *net.ConnectionPool
}

// Insert inserts a document into a collection
func (c *DBClient) Insert(collection string, doc map[string]interface{}) (string, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return "", err
	}
	defer c.pool.Put(conn)

	docBytes, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	request := protocol.EncodeDocInsertRequest(collection, docBytes)
	if err := conn.Write(request); err != nil {
		return "", err
	}

	resp, err := readValueResponse(conn)
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

// Query returns a query builder for finding documents
func (c *DBClient) Query(collection string) *QueryBuilder {
	return &QueryBuilder{
		client:     c,
		collection: collection,
		filters:    make([]QueryFilter, 0),
		limit:      -1,
	}
}

// Update returns an update builder
func (c *DBClient) Update(collection string) *UpdateBuilder {
	return &UpdateBuilder{
		client:     c,
		collection: collection,
		filters:    make([]QueryFilter, 0),
		setFields:  make(map[string]interface{}),
		merge:      true,
	}
}

// Delete returns a delete builder
func (c *DBClient) Delete(collection string) *DeleteBuilder {
	return &DeleteBuilder{
		client:     c,
		collection: collection,
		filters:    make([]QueryFilter, 0),
	}
}

// CreateIndex creates an index on a field in a collection
func (c *DBClient) CreateIndex(collection string, field string) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	request := protocol.EncodeDocIndexRequest(collection, field)
	if err := conn.Write(request); err != nil {
		return err
	}

	resp, err := readValueResponse(conn)
	if err != nil {
		return err
	}

	// Check if response is "OK"
	if string(resp) != "OK" {
		return fmt.Errorf("index creation returned unexpected response: %s", string(resp))
	}

	return nil
}

// Internal execution methods used by builders

func (c *DBClient) executeFind(collection string, opts map[string]interface{}) ([]map[string]interface{}, error) {
	conn, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	defer c.pool.Put(conn)

	optsBytes, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query options: %w", err)
	}

	request := protocol.EncodeDocFindRequest(collection, optsBytes)
	if err := conn.Write(request); err != nil {
		return nil, err
	}

	resp, err := readValueResponse(conn)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(resp, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return results, nil
}

func (c *DBClient) executeUpdate(collection string, opts map[string]interface{}) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	optsBytes, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal update options: %w", err)
	}

	request := protocol.EncodeDocUpdateRequest(collection, optsBytes)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

func (c *DBClient) executeDelete(collection string, opts map[string]interface{}) error {
	conn, err := c.pool.Get()
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	optsBytes, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal delete options: %w", err)
	}

	request := protocol.EncodeDocDeleteRequest(collection, optsBytes)
	if err := conn.Write(request); err != nil {
		return err
	}

	return readOKResponse(conn)
}

// Query Builder Implementation

type QueryFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type QueryBuilder struct {
	client     *DBClient
	collection string
	filters    []QueryFilter
	sort       *SortOption
	skip       int
	limit      int
}

type SortOption struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

const (
	Eq   = "eq"
	Ne   = "ne"
	Gt   = "gt"
	Gte  = "gte"
	Lt   = "lt"
	Lte  = "lte"
	In   = "in"
	Asc  = "asc"
	Desc = "desc"
)

func (qb *QueryBuilder) Where(field, operator string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return qb
}

func (qb *QueryBuilder) OrderBy(field, direction string) *QueryBuilder {
	qb.sort = &SortOption{
		Field:     field,
		Direction: direction,
	}
	return qb
}

func (qb *QueryBuilder) Skip(n int) *QueryBuilder {
	qb.skip = n
	return qb
}

func (qb *QueryBuilder) Take(n int) *QueryBuilder {
	qb.limit = n
	return qb
}

func (qb *QueryBuilder) Exec() ([]map[string]interface{}, error) {
	opts := map[string]interface{}{
		"filters": qb.filters,
		"skip":    qb.skip,
		"limit":   qb.limit,
	}
	if qb.sort != nil {
		opts["sort"] = qb.sort
	}
	return qb.client.executeFind(qb.collection, opts)
}

// Update Builder

type UpdateBuilder struct {
	client     *DBClient
	collection string
	filters    []QueryFilter
	setFields  map[string]interface{}
	merge      bool
}

func (ub *UpdateBuilder) Where(field, operator string, value interface{}) *UpdateBuilder {
	ub.filters = append(ub.filters, QueryFilter{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return ub
}

func (ub *UpdateBuilder) Set(field string, value interface{}) *UpdateBuilder {
	ub.setFields[field] = value
	return ub
}

func (ub *UpdateBuilder) Exec() error {
	opts := map[string]interface{}{
		"filters": ub.filters,
		"set":     ub.setFields,
		"merge":   ub.merge,
	}
	return ub.client.executeUpdate(ub.collection, opts)
}

// Delete Builder

type DeleteBuilder struct {
	client     *DBClient
	collection string
	filters    []QueryFilter
}

func (db *DeleteBuilder) Where(field, operator string, value interface{}) *DeleteBuilder {
	db.filters = append(db.filters, QueryFilter{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return db
}

func (db *DeleteBuilder) Exec() error {
	opts := map[string]interface{}{
		"filters": db.filters,
	}
	return db.client.executeDelete(db.collection, opts)
}
