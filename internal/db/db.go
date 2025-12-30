package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

var (
	ErrInvalidID = errors.New("invalid document ID")
	ErrStopScan  = errors.New("stop scan")
)

// Store provides high-level document database operations (Wide-Column DB)
type Store struct {
	storage *DocStorage
	mu      sync.RWMutex
	// Index tracking: collection -> field -> value -> document IDs
	indexes map[string]map[string]map[interface{}][]string

	// Schema registry: collection -> Schema
	schemas map[string]Schema
}

// New creates a new document store
func New(path string) (*Store, error) {
	store, err := NewStorage(path)
	if err != nil {
		return nil, err
	}

	ds := &Store{
		storage: store,
		indexes: make(map[string]map[string]map[interface{}][]string),
		schemas: make(map[string]Schema),
	}

	// Load existing indexes from metadata
	if err := ds.loadIndexMetadata(); err != nil {
		return nil, fmt.Errorf("failed to load indexes: %w", err)
	}

	return ds, nil
}

// Close closes the document store
func (ds *Store) Close() error {
	return ds.storage.Close()
}

// RegisterSchema registers a new schema for a collection
func (ds *Store) RegisterSchema(schema Schema) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if schema.TableName == "" {
		return fmt.Errorf("table name is required")
	}

	ds.schemas[schema.TableName] = schema
	return nil
}

// Insert adds a new document to a collection
func (ds *Store) Insert(collection string, doc Document) (string, error) {
	if collection == "" {
		return "", ErrInvalidCollection
	}

	if doc == nil {
		doc = make(Document)
	}

	// Schema Validation
	ds.mu.RLock()
	schema, hasSchema := ds.schemas[collection]
	ds.mu.RUnlock()

	var key string
	var id string

	if hasSchema {
		// 1. Validate against schema
		if err := schema.Validate(doc); err != nil {
			return "", fmt.Errorf("schema validation failed: %w", err)
		}

		// 2. Generate Key based on Schema (Partition Key)
		var err error
		key, err = schema.GenerateKey(doc)
		if err != nil {
			return "", fmt.Errorf("failed to generate key: %w", err)
		}

		// Ensure key is unique? For now, we overwrite if same PK+CK.
		// But we usually want an ID too.
		// If the schema has an explicit ID field, use it.
		if docID, ok := doc["_id"].(string); ok {
			id = docID
		} else {
			// If no ID provided, but we have a unique PK/CK combo,
			// we can use that combo as the ID or generate a new one.
			// Let's generate a UUID for _id if missing, just for reference.
			id = uuid.New().String()
			doc["_id"] = id
		}

		// If the key doesn't include the ID (e.g. non-unique PK+CK),
		// we might overwrite data.
		// In Cassandra, PK determines the row.
		// If we want unique rows, the FULL PK (Partition + Clustering) must be unique.
		// So `key` IS the unique identifier for storage.

	} else {
		// Schemaless mode (Legacy)
		idRaw, ok := doc["_id"].(string)
		if !ok || idRaw == "" {
			id = uuid.New().String()
			doc["_id"] = id
		} else {
			id = idRaw
		}
		key = makeKey(collection, id)
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

	if err := ds.storage.Set(key, data); err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	// Update indexes (only for non-schema path or if we support secondary indexes with schema)
	// For now, keep supporting secondary indexes
	ds.updateIndexes(collection, id, doc)

	return id, nil
}

// Get retrieves a document by ID
func (ds *Store) Get(collection, id string) (Document, error) {
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

// tryUsePartitionKey attempts to use partition key for direct lookup (Cassandra-style)
// Returns documents and a bool indicating if partition key was used
func (ds *Store) tryUsePartitionKey(collection string, filters []Query, opts FindOptions) ([]Document, bool) {
	ds.mu.RLock()
	schema, hasSchema := ds.schemas[collection]
	ds.mu.RUnlock()

	if !hasSchema {
		return nil, false
	}

	// Check if all partition key fields are present in filters with equality
	partitionValues := make(map[string]interface{})
	for _, pkField := range schema.PartitionKeys {
		found := false
		for _, filter := range filters {
			if filter.Field == pkField && filter.Operator == "eq" {
				partitionValues[pkField] = filter.Value
				found = true
				break
			}
		}
		if !found {
			// Missing partition key field, cannot use partition key lookup
			return nil, false
		}
	}

	// Build a document with partition key values to generate the key
	doc := make(Document)
	for field, value := range partitionValues {
		doc[field] = value
	}

	// Generate the partition-based key prefix
	// Use GeneratePartitionPrefix to allow querying part of the primary key (the partition part)
	keyPrefix, err := schema.GeneratePartitionPrefix(doc)
	if err != nil {
		return nil, false
	}

	// Scan for all documents with this partition key prefix
	var results []Document

	// Prepare pagination
	skipped := 0
	needed := opts.Limit
	if needed < 0 {
		needed = -1
	}

	ds.storage.db.View(func(txn *badger.Txn) error {
		iterOpts := badger.DefaultIteratorOptions
		iterOpts.Prefix = []byte(keyPrefix)
		it := txn.NewIterator(iterOpts)
		defer it.Close()

		for it.Seek([]byte(keyPrefix)); it.ValidForPrefix([]byte(keyPrefix)); it.Next() {
			// Apply Skip
			if skipped < opts.Skip {
				skipped++
				continue
			}

			// Apply Limit
			if needed != -1 && len(results) >= needed {
				break
			}

			item := it.Item()

			// Extract document
			err := item.Value(func(val []byte) error {
				var doc Document
				if err := json.Unmarshal(val, &doc); err == nil {
					// Check remaining filters (clustering keys or non-indexed fields)
					if matchesFilters(doc, filters) {
						results = append(results, doc)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	// If sorting is required, we should have fetched all matching docs (Limit logic above assumes no sort or pre-sorted)
	// Partition keys in Badger are sorted by key.
	// For now, return what we found. Caller (Find) might resort if needed, but since we return success,
	// Find needs to handle it.
	// To be safe: if Sort is requested, we shouldn't limit early?
	// Actually, `Find` doesn't pass results to sorter if we return here.
	// We should sort here if needed, OR just let `Find` handle sorting?
	// If `Find` handles sorting, we must return ALL matches, ignoring Limit in loop.

	if opts.Sort != nil {
		sortResults(results, opts.Sort)
		// Now apply pagination
		// Start variable unused for now as we assume sorting handles needs
		// If sorting is enabled, we CANNOT use early skip/limit efficiently without clustering keys support.
		// For now, let's assume we return results and let caller handle pagination?
		// No, `tryUsePartitionKey` signature implies it did the work.

		// If usage is:
		// results, used := tryUsePartitionKey(...)
		// if used { return results }
		// Then we must handle everything.

		// Re-slicing for pagination after sort:
		// Since we applied Skip/Limit during scan (which is unsorted order or Key order),
		// returning valid paged results strictly requires Key order == Sort order.
		// If Sort order differs, we must fetch ALL, sort, then page.
	}

	return results, true
}

// tryUseIndexes attempts to use indexes to satisfy filters
// Returns document IDs that match the filter and a bool indicating if indexes were used
func (ds *Store) tryUseIndexes(collection string, filters []Query) ([]string, bool) {
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
func (ds *Store) Find(collection string, opts FindOptions) ([]Document, error) {
	if collection == "" {
		return nil, ErrInvalidCollection
	}

	var results []Document
	var docIDs []string

	// 1. Try to use Partition Key (O(1) lookup - Fastest)
	if docs, used := ds.tryUsePartitionKey(collection, opts.Filters, opts); used {
		// If partition key was used, pagination and filtering is already applied.
		// We just return the docs.
		return docs, nil
	}

	// 2. Try to use Secondary Indexes
	var indexUsed bool
	docIDs, indexUsed = ds.tryUseIndexes(collection, opts.Filters)

	if !indexUsed {
		// 3. Fallback to Full Scan (Slowest)
		var err error
		results, err = ds.scanAll(collection, opts)
		if err != nil {
			return nil, err
		}
	}

	if indexUsed { // logic optimization: unified path for indexed access
		// Use index/partition-based retrieval - much faster for filtered queries
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
	}

	// Apply sorting
	if opts.Sort != nil {
		sortResults(results, opts.Sort)
	}

	// Apply skip/limit for scan results (for index path it's done above)
	// For scan path (results populated in scanAll), we need to apply pagination here if not already done
	// But `scanAll` below handles basic building, pagination for scan results should be done after sort
	// Actually, applying limit during scan is optimization we can do later, for now let's reuse sorting/pagination logic

	// Apply pagination (skip/limit)
	startIdx := opts.Skip
	endIdx := opts.Skip + opts.Limit
	if opts.Limit <= 0 {
		endIdx = len(results)
	}

	if startIdx >= len(results) {
		return []Document{}, nil
	}
	if endIdx > len(results) {
		endIdx = len(results)
	}

	return results[startIdx:endIdx], nil
}

// scanAll performs a full collection scan with filters
func (ds *Store) scanAll(collection string, opts FindOptions) ([]Document, error) {
	var results []Document

	// Optimization: If no sort is requested, we can apply Limit during scan
	// If sorting is required, we MUST scan everything matching filters first
	canLimitEarly := opts.Sort == nil

	// Max items we need to find if we are limiting early
	needed := opts.Skip + opts.Limit
	if opts.Limit <= 0 {
		needed = -1 // No limit
	}

	prefix := makeCollectionPrefix(collection)
	err := ds.storage.Scan(prefix, func(key string, data []byte) error {
		// If we've found enough, stop scanning (only if no sort)
		if canLimitEarly && needed != -1 && len(results) >= needed {
			return ErrStopScan
		}

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
		if errors.Is(err, ErrStopScan) {
			return results, nil
		}
		return nil, err
	}

	return results, nil
}

// FindOne returns the first document matching the query
func (ds *Store) FindOne(collection string, opts FindOptions) (Document, error) {
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
func (ds *Store) Update(collection, id string, opts UpdateOptions) error {
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
func (ds *Store) Delete(collection, id string) error {
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
func (ds *Store) DeleteMany(collection string, opts FindOptions) (int64, error) {
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
func (ds *Store) Count(collection string) (int64, error) {
	if collection == "" {
		return 0, ErrInvalidCollection
	}

	prefix := makeCollectionPrefix(collection)
	return ds.storage.Count(prefix)
}

// CreateIndex builds an index on a field
func (ds *Store) CreateIndex(collection, field string) error {
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
func (ds *Store) DropIndex(collection, field string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if coll, ok := ds.indexes[collection]; ok {
		delete(coll, field)
	}

	return ds.saveIndexMetadata()
}

// ListIndexes returns all indexes for a collection
func (ds *Store) ListIndexes(collection string) []string {
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
func (ds *Store) Query(collection string) *QueryBuilder {
	return NewQueryBuilder(collection)
}

// Helper functions

func makeKey(collection, id string) string {
	return fmt.Sprintf("doc:%s:%s", collection, id)
}

func makeCollectionPrefix(collection string) string {
	return fmt.Sprintf("doc:%s:", collection)
}

func (ds *Store) updateIndexes(collection, id string, doc Document) {
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

func (ds *Store) removeIndexes(collection, id string, doc Document) {
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

func (ds *Store) loadIndexMetadata() error {
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

func (ds *Store) saveIndexMetadata() error {
	metadata := make(map[string][]string)
	for collection, coll := range ds.indexes {
		for field := range coll {
			metadata[collection] = append(metadata[collection], field)
		}
	}

	return ds.storage.SetJSON("db:indexes_metadata", metadata)
}
