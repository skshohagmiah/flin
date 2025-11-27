package storage

import (
	"fmt"
	"sync"
	"time"
)

// MemoryStorage implements pure in-memory KV storage (like Redis)
// No persistence - data is lost on restart
// Extremely fast - no disk I/O
type MemoryStorage struct {
	data map[string]*memoryEntry
	mu   sync.RWMutex
}

type memoryEntry struct {
	value      []byte
	expiration time.Time
	hasExpiry  bool
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() (*MemoryStorage, error) {
	ms := &MemoryStorage{
		data: make(map[string]*memoryEntry),
	}

	// Start background cleanup for expired keys
	go ms.cleanupExpired()

	return ms, nil
}

// Set stores a key-value pair in memory
func (m *MemoryStorage) Set(key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &memoryEntry{
		value: make([]byte, len(value)),
	}
	copy(entry.value, value)

	if ttl > 0 {
		entry.expiration = time.Now().Add(ttl)
		entry.hasExpiry = true
	}

	m.data[key] = entry
	return nil
}

// Get retrieves a value by key
func (m *MemoryStorage) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// Check expiration
	if entry.hasExpiry && time.Now().After(entry.expiration) {
		return nil, fmt.Errorf("key expired")
	}

	// Return copy to prevent external modification
	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	return result, nil
}

// Delete removes a key
func (m *MemoryStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

// Exists checks if a key exists
func (m *MemoryStorage) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// Check expiration
	if entry.hasExpiry && time.Now().After(entry.expiration) {
		return false, nil
	}

	return true, nil
}

// Incr increments a numeric value
func (m *MemoryStorage) Incr(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.data[key]
	if !exists {
		// Initialize to 1
		m.data[key] = &memoryEntry{
			value: []byte("1"),
		}
		return nil
	}

	// Parse current value
	var current int64
	_, err := fmt.Sscanf(string(entry.value), "%d", &current)
	if err != nil {
		return fmt.Errorf("value is not a number")
	}

	// Increment
	current++
	entry.value = []byte(fmt.Sprintf("%d", current))
	return nil
}

// Decr decrements a numeric value
func (m *MemoryStorage) Decr(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.data[key]
	if !exists {
		// Initialize to -1
		m.data[key] = &memoryEntry{
			value: []byte("-1"),
		}
		return nil
	}

	// Parse current value
	var current int64
	_, err := fmt.Sscanf(string(entry.value), "%d", &current)
	if err != nil {
		return fmt.Errorf("value is not a number")
	}

	// Decrement
	current--
	entry.value = []byte(fmt.Sprintf("%d", current))
	return nil
}

// Scan retrieves all values with keys matching the prefix
func (m *MemoryStorage) Scan(prefix string) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var values [][]byte
	for key, entry := range m.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Check expiration
			if entry.hasExpiry && time.Now().After(entry.expiration) {
				continue
			}

			valueCopy := make([]byte, len(entry.value))
			copy(valueCopy, entry.value)
			values = append(values, valueCopy)
		}
	}

	return values, nil
}

// ScanKeys retrieves all keys matching the prefix
func (m *MemoryStorage) ScanKeys(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for key, entry := range m.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Check expiration
			if entry.hasExpiry && time.Now().After(entry.expiration) {
				continue
			}
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// ScanKeysWithValues retrieves all keys and values matching the prefix
func (m *MemoryStorage) ScanKeysWithValues(prefix string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	kvPairs := make(map[string][]byte)
	for key, entry := range m.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Check expiration
			if entry.hasExpiry && time.Now().After(entry.expiration) {
				continue
			}

			valueCopy := make([]byte, len(entry.value))
			copy(valueCopy, entry.value)
			kvPairs[key] = valueCopy
		}
	}

	return kvPairs, nil
}

// BatchSet stores multiple key-value pairs atomically
func (m *MemoryStorage) BatchSet(kvPairs map[string][]byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// All operations are atomic in memory
	for key, value := range kvPairs {
		if key == "" {
			continue
		}

		entry := &memoryEntry{
			value: make([]byte, len(value)),
		}
		copy(entry.value, value)

		if ttl > 0 {
			entry.expiration = time.Now().Add(ttl)
			entry.hasExpiry = true
		}

		m.data[key] = entry
	}

	return nil
}

// BatchGet retrieves multiple values by keys
func (m *MemoryStorage) BatchGet(keys []string) (map[string][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]byte, len(keys))
	now := time.Now()

	for _, key := range keys {
		if key == "" {
			continue
		}

		entry, exists := m.data[key]
		if !exists {
			continue
		}

		// Check expiration
		if entry.hasExpiry && now.After(entry.expiration) {
			continue
		}

		valueCopy := make([]byte, len(entry.value))
		copy(valueCopy, entry.value)
		result[key] = valueCopy
	}

	return result, nil
}

// BatchDelete removes multiple keys atomically
func (m *MemoryStorage) BatchDelete(keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		if key == "" {
			continue
		}
		delete(m.data, key)
	}

	return nil
}

// Close closes the storage (no-op for memory storage)
func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all data
	m.data = make(map[string]*memoryEntry)
	return nil
}

// cleanupExpired removes expired keys periodically
func (m *MemoryStorage) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, entry := range m.data {
			if entry.hasExpiry && now.After(entry.expiration) {
				delete(m.data, key)
			}
		}
		m.mu.Unlock()
	}
}

// Stats returns storage statistics
func (m *MemoryStorage) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalSize := int64(0)
	expiredCount := 0
	now := time.Now()

	for _, entry := range m.data {
		totalSize += int64(len(entry.value))
		if entry.hasExpiry && now.After(entry.expiration) {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"type":           "memory",
		"total_keys":     len(m.data),
		"total_size":     totalSize,
		"expired_keys":   expiredCount,
		"memory_backend": true,
	}
}
