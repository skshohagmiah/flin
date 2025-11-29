package db

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"github.com/skshohagmiah/flin/internal/storage"
)

// DocStore provides high-level document database operations
type DocStore struct {
	storage *storage.DocStorage
	mu      sync.RWMutex
	// Index tracking: collection -> field -> value -> document IDs
	indexes map[string]map[string]map[interface{}][]string
}

// New creates a new document store
func New(path string) (*DocStore, error) {
	store, err := storage.NewDocStorage(path)
	if err != nil {
		return nil, err
	}

	ds := &DocStore{
		storage: store,
		indexes: make(map[string]map[string]map[interface{}][]string),
	}

	// Load existing indexes from metadata
	if err := ds.loadIndexMetadata(); err != nil {
		return nil, fmt.Errorf("failed to load indexes: %w", err)
	}

	return ds, nil
}

// Close closes the document store
func (ds *DocStore) Close() error {
	return ds.storage.Close()
}

// Insert adds a new document to a collection
func (ds *DocStore) Insert(collection string, doc Document) (string, error) {
	if collection == "" {
		return "", ErrInvalidCollection
	}

	if doc == nil {
		doc = make(Document)
	}

	// Generate ID if not present
	id, ok := doc["_id"].(string)
	if !ok || id == "" {
		id = uuid.New().String()
		doc["_id"] = id
	}

	// Set timestamps
	now := time.Now()
	doc["_created_at"] = now.UnixMilli()
	doc["_updated_at"] = now.UnixMilli()

	// Marshal and store
	data, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	key := makeKey(collection, id)
	if err := ds.storage.Set(key, data); err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	// Update indexes
	ds.updateIndexes(collection, id, doc)

	return id, nil
}

// Get retrieves a document by ID
func (ds *DocStore) Get(collection, id string) (Document, error) {
	if collection == "" || id == "" {
		return nil, ErrInvalidDocument
	}

	key := makeKey(collection, id)
	data, err := ds.storage.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// tryUseIndexes attempts to use indexes to satisfy filters
// Returns document IDs that match the filter and a bool indicating if indexes were used
func (ds *DocStore) tryUseIndexes(collection string, filters []Query) ([]string, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Check if collection has indexes
	collIndexes, ok := ds.indexes[collection]
	if !ok || len(collIndexes) == 0 {
		return nil, false
	}

	// Try to find a filter that has an index and uses equality
	for _, filter := range filters {
		// Only use indexes for equality filters ("eq" operator)
		if filter.Operator != "eq" {
			continue
		}

		// Check if this field has an index
		if fieldIndex, hasIndex := collIndexes[filter.Field]; hasIndex {
			// Use the index for this field
			if docIDs, found := fieldIndex[filter.Value]; found {
				return docIDs, true
			}
			// Field is indexed but no documents match
			return []string{}, true
		}
	}

	// No suitable index found
	return nil, false
}

// Find searches for documents matching the query criteria
func (ds *DocStore) Find(collection string, opts FindOptions) ([]Document, error) {
	if collection == "" {
		return nil, ErrInvalidCollection
	}

	var results []Document
	var docIDs []string

	// Check if we can use indexes for optimization
	docIDs, canUseIndex := ds.tryUseIndexes(collection, opts.Filters)

	if canUseIndex {
		// Use index-based retrieval - much faster for filtered queries
		// (even if empty, it means the index confirmed no matches)

		// Apply pagination to index results first to avoid unnecessary deserialization
		startIdx := opts.Skip
		endIdx := opts.Skip + opts.Limit
		if opts.Limit < 0 {
			endIdx = len(docIDs)
		}

		// Bounds check
		if startIdx >= len(docIDs) {
			// Pagination skip goes past all results
			return results, nil
		}
		if endIdx > len(docIDs) {
			endIdx = len(docIDs)
		}

		// Only deserialize documents we actually need
		for i := startIdx; i < endIdx && i < len(docIDs); i++ {
			id := docIDs[i]
			key := makeKey(collection, id)
			data, err := ds.storage.Get(key)
			if err != nil {
				// Skip if document not found
				continue
			}

			var doc Document
			if err := json.Unmarshal(data, &doc); err != nil {
				continue
			}

			// Still need to verify all filters match (in case of multiple filters)
			if matchesFilters(doc, opts.Filters) {
				results = append(results, doc)
			}
		}
	} else {
		// Fall back to full table scan
		prefix := makeCollectionPrefix(collection)
		err := ds.storage.Scan(prefix, func(key string, data []byte) error {
			var doc Document
			if err := json.Unmarshal(data, &doc); err != nil {
				return err
			}

			// Apply filters
			if matchesFilters(doc, opts.Filters) {
				results = append(results, doc)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	// Apply sorting
	if opts.Sort != nil {
		sortResults(results, opts.Sort)
	}

	// Apply pagination
	if opts.Skip > 0 {
		if opts.Skip >= len(results) {
			results = []Document{}
		} else {
			results = results[opts.Skip:]
		}
	}

	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// FindOne returns the first document matching the query
func (ds *DocStore) FindOne(collection string, opts FindOptions) (Document, error) {
	opts.Limit = 1
	results, err := ds.Find(collection, opts)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, ErrDocumentNotFound
	}

	return results[0], nil
}

// Update modifies a document
func (ds *DocStore) Update(collection, id string, opts UpdateOptions) error {
	if collection == "" || id == "" {
		return ErrInvalidDocument
	}

	// Get existing document
	doc, err := ds.Get(collection, id)
	if err != nil {
		return err
	}

	// Save old doc for index updates
	oldDoc := make(Document)
	for k, v := range doc {
		oldDoc[k] = v
	}

	// Apply updates
	if opts.Merge {
		for k, v := range opts.Set {
			doc[k] = v
		}
	} else {
		doc = opts.Set
		doc["_id"] = id // Preserve ID
	}

	// Update timestamp
	doc["_updated_at"] = time.Now().UnixMilli()

	// Remove unset fields
	for _, field := range opts.Unset {
		delete(doc, field)
	}

	// Marshal and store
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	key := makeKey(collection, id)
	if err := ds.storage.Set(key, data); err != nil {
		return err
	}

	// Update indexes
	ds.removeIndexes(collection, id, oldDoc)
	ds.updateIndexes(collection, id, doc)

	return nil
}

// Delete removes a document
func (ds *DocStore) Delete(collection, id string) error {
	if collection == "" || id == "" {
		return ErrInvalidDocument
	}

	// Get document before deletion for index updates
	doc, err := ds.Get(collection, id)
	if err != nil {
		return err
	}

	key := makeKey(collection, id)
	if err := ds.storage.Delete(key); err != nil {
		return err
	}

	// Update indexes
	ds.removeIndexes(collection, id, doc)

	return nil
}

// DeleteMany removes all documents matching a query
func (ds *DocStore) DeleteMany(collection string, opts FindOptions) (int64, error) {
	results, err := ds.Find(collection, opts)
	if err != nil {
		return 0, err
	}

	var deleted int64
	for _, doc := range results {
		id, ok := doc["_id"].(string)
		if !ok {
			continue
		}
		if err := ds.Delete(collection, id); err == nil {
			deleted++
		}
	}

	return deleted, nil
}

// Count returns the number of documents in a collection
func (ds *DocStore) Count(collection string) (int64, error) {
	if collection == "" {
		return 0, ErrInvalidCollection
	}

	prefix := makeCollectionPrefix(collection)
	return ds.storage.Count(prefix)
}

// CreateIndex builds an index on a field
func (ds *DocStore) CreateIndex(collection, field string) error {
	if collection == "" || field == "" {
		return ErrInvalidDocument
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Initialize collection indexes if needed
	if _, ok := ds.indexes[collection]; !ok {
		ds.indexes[collection] = make(map[string]map[interface{}][]string)
	}

	// Initialize field index
	if _, ok := ds.indexes[collection][field]; !ok {
		ds.indexes[collection][field] = make(map[interface{}][]string)
	}

	// Scan all documents and build index
	prefix := makeCollectionPrefix(collection)
	err := ds.storage.Scan(prefix, func(key string, data []byte) error {
		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return err
		}

		id, ok := doc["_id"].(string)
		if !ok {
			return nil
		}

		// Add to index
		if val, ok := doc[field]; ok {
			if _, exists := ds.indexes[collection][field][val]; !exists {
				ds.indexes[collection][field][val] = []string{}
			}
			ds.indexes[collection][field][val] = append(ds.indexes[collection][field][val], id)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return ds.saveIndexMetadata()
}

// DropIndex removes an index
func (ds *DocStore) DropIndex(collection, field string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if coll, ok := ds.indexes[collection]; ok {
		delete(coll, field)
	}

	return ds.saveIndexMetadata()
}

// ListIndexes returns all indexes for a collection
func (ds *DocStore) ListIndexes(collection string) []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var indexes []string
	if coll, ok := ds.indexes[collection]; ok {
		for field := range coll {
			indexes = append(indexes, field)
		}
	}
	return indexes
}

// Query returns a query builder for the collection
func (ds *DocStore) Query(collection string) *QueryBuilder {
	return NewQueryBuilder(collection)
}

// Helper functions

func makeKey(collection, id string) string {
	return fmt.Sprintf("doc:%s:%s", collection, id)
}

func makeCollectionPrefix(collection string) string {
	return fmt.Sprintf("doc:%s:", collection)
}

func (ds *DocStore) updateIndexes(collection, id string, doc Document) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	coll, ok := ds.indexes[collection]
	if !ok {
		return
	}

	for field, fieldIndex := range coll {
		if val, exists := doc[field]; exists {
			if _, ok := fieldIndex[val]; !ok {
				fieldIndex[val] = []string{}
			}
			fieldIndex[val] = append(fieldIndex[val], id)
		}
	}
}

func (ds *DocStore) removeIndexes(collection, id string, doc Document) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	coll, ok := ds.indexes[collection]
	if !ok {
		return
	}

	for field, fieldIndex := range coll {
		if val, exists := doc[field]; exists {
			if ids, ok := fieldIndex[val]; ok {
				newIds := []string{}
				for _, existingId := range ids {
					if existingId != id {
						newIds = append(newIds, existingId)
					}
				}
				if len(newIds) == 0 {
					delete(fieldIndex, val)
				} else {
					fieldIndex[val] = newIds
				}
			}
		}
	}
}

func (ds *DocStore) loadIndexMetadata() error {
	metadataKey := "db:indexes_metadata"

	var metadata map[string][]string
	err := ds.storage.GetJSON(metadataKey, &metadata)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil // No metadata yet
		}
		return err
	}

	// Rebuild index structure
	for collection, fields := range metadata {
		if _, ok := ds.indexes[collection]; !ok {
			ds.indexes[collection] = make(map[string]map[interface{}][]string)
		}
		for _, field := range fields {
			ds.indexes[collection][field] = make(map[interface{}][]string)
		}
	}

	return nil
}

func (ds *DocStore) saveIndexMetadata() error {
	metadata := make(map[string][]string)
	for collection, coll := range ds.indexes {
		for field := range coll {
			metadata[collection] = append(metadata[collection], field)
		}
	}

	return ds.storage.SetJSON("db:indexes_metadata", metadata)
}
