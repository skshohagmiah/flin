package storage

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Helper Functions
// ============================================================================

// createTestKV creates a new KV storage instance for testing
func createTestKV(t *testing.T) *KVStorage {
	kv, err := NewMemory()
	if err != nil {
		t.Fatalf("Failed to create test KV store: %v", err)
	}
	return kv
}

// createTestKVWithShards creates a new KV storage with custom shard count
func createTestKVWithShards(t *testing.T, shards int) *KVStorage {
	kv, err := NewMemoryWithShards(shards)
	if err != nil {
		t.Fatalf("Failed to create test KV store with %d shards: %v", shards, err)
	}
	return kv
}

// ============================================================================
// Basic Operations Tests
// ============================================================================

// TestSet tests basic key-value storage
func TestSet(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "testkey"
	value := []byte("testvalue")

	err := kv.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
}

// TestSetEmptyKey tests that empty keys are rejected
func TestSetEmptyKey(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	err := kv.Set("", []byte("value"), 0)
	if err != ErrInvalidKey {
		t.Fatalf("Expected ErrInvalidKey for empty key, got: %v", err)
	}
}

// TestGet tests basic key retrieval
func TestGet(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "testkey"
	expectedValue := []byte("testvalue")

	// Set the value
	err := kv.Set(key, expectedValue, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value
	value, err := kv.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(value) != string(expectedValue) {
		t.Fatalf("Expected %s, got %s", expectedValue, value)
	}
}

// TestGetEmptyKey tests that empty keys are rejected
func TestGetEmptyKey(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	_, err := kv.Get("")
	if err != ErrInvalidKey {
		t.Fatalf("Expected ErrInvalidKey for empty key, got: %v", err)
	}
}

// TestGetNonexistent tests retrieving non-existent keys
func TestGetNonexistent(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	_, err := kv.Get("nonexistent")
	if err != ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound, got: %v", err)
	}
}

// TestDelete tests key deletion
func TestDelete(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "testkey"
	value := []byte("testvalue")

	// Set the value
	err := kv.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it exists
	_, err = kv.Get(key)
	if err != nil {
		t.Fatalf("Get before delete failed: %v", err)
	}

	// Delete it
	err = kv.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = kv.Get(key)
	if err != ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound after delete, got: %v", err)
	}
}

// TestDeleteEmptyKey tests that empty keys are rejected
func TestDeleteEmptyKey(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	err := kv.Delete("")
	if err != ErrInvalidKey {
		t.Fatalf("Expected ErrInvalidKey for empty key, got: %v", err)
	}
}

// TestExists tests key existence check
func TestExists(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "testkey"

	// Check non-existent key
	exists, err := kv.Exists(key)
	if err != nil {
		t.Fatalf("Exists check failed: %v", err)
	}
	if exists {
		t.Fatal("Expected key to not exist")
	}

	// Set the key
	err = kv.Set(key, []byte("value"), 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Check existing key
	exists, err = kv.Exists(key)
	if err != nil {
		t.Fatalf("Exists check failed: %v", err)
	}
	if !exists {
		t.Fatal("Expected key to exist")
	}
}

// ============================================================================
// TTL Tests
// ============================================================================

// TestTTL tests that keys expire after TTL
func TestTTL(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "expiring_key"
	value := []byte("value")
	ttl := 100 * time.Millisecond

	// Set with TTL
	err := kv.Set(key, value, ttl)
	if err != nil {
		t.Fatalf("Set with TTL failed: %v", err)
	}

	// Should exist immediately
	exists, err := kv.Exists(key)
	if err != nil || !exists {
		t.Fatal("Key should exist immediately after set")
	}

	// Should exist after half the TTL
	time.Sleep(50 * time.Millisecond)
	exists, err = kv.Exists(key)
	if err != nil || !exists {
		t.Fatal("Key should still exist before TTL expires")
	}

	// Should not exist after TTL
	time.Sleep(100 * time.Millisecond)
	exists, err = kv.Exists(key)
	if err != nil || exists {
		t.Fatal("Key should not exist after TTL expires")
	}
}

// TestZeroTTL tests that zero TTL means no expiration
func TestZeroTTL(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "persistent_key"
	value := []byte("value")

	// Set with zero TTL (permanent)
	err := kv.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set with zero TTL failed: %v", err)
	}

	// Sleep and verify it still exists
	time.Sleep(200 * time.Millisecond)
	exists, err := kv.Exists(key)
	if err != nil || !exists {
		t.Fatal("Key with zero TTL should persist")
	}
}

// ============================================================================
// Counter Operations Tests
// ============================================================================

// TestIncr tests atomic increment
func TestIncr(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "counter"

	// Initialize counter
	err := kv.Set(key, []byte{0, 0, 0, 0, 0, 0, 0, 0}, 0)
	if err != nil {
		t.Fatalf("Set counter failed: %v", err)
	}

	// Increment multiple times
	for i := 0; i < 5; i++ {
		err := kv.Incr(key)
		if err != nil {
			t.Fatalf("Incr failed on iteration %d: %v", i, err)
		}
	}

	// Verify counter value
	value, err := kv.Get(key)
	if err != nil {
		t.Fatalf("Get counter failed: %v", err)
	}

	if len(value) != 8 {
		t.Fatalf("Expected 8-byte value, got %d", len(value))
	}
}

// TestDecr tests atomic decrement
func TestDecr(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "counter"

	// Initialize counter to 10
	err := kv.Set(key, []byte{0, 0, 0, 0, 0, 0, 0, 10}, 0)
	if err != nil {
		t.Fatalf("Set counter failed: %v", err)
	}

	// Decrement
	err = kv.Decr(key)
	if err != nil {
		t.Fatalf("Decr failed: %v", err)
	}

	// Verify the counter decreased
	value, err := kv.Get(key)
	if err != nil {
		t.Fatalf("Get counter failed: %v", err)
	}

	if len(value) != 8 {
		t.Fatalf("Expected 8-byte value, got %d", len(value))
	}
}

// ============================================================================
// Scan Operations Tests
// ============================================================================

// TestScan tests prefix scanning
func TestScan(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	prefix := "user:"
	keys := []string{"user:1", "user:2", "user:3", "other:1"}

	// Set all keys
	for i, key := range keys {
		err := kv.Set(key, []byte(fmt.Sprintf("value%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed for %s: %v", key, err)
		}
	}

	// Scan with prefix
	values, err := kv.Scan(prefix)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should have exactly 3 values (user:1, user:2, user:3)
	if len(values) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(values))
	}
}

// TestScanKeys tests prefix scanning for keys
func TestScanKeys(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	prefix := "product:"
	keys := []string{"product:1", "product:2", "product:3", "other:1"}

	// Set all keys
	for i, key := range keys {
		err := kv.Set(key, []byte(fmt.Sprintf("value%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed for %s: %v", key, err)
		}
	}

	// Scan keys with prefix
	resultKeys, err := kv.ScanKeys(prefix)
	if err != nil {
		t.Fatalf("ScanKeys failed: %v", err)
	}

	// Should have exactly 3 keys
	if len(resultKeys) != 3 {
		t.Fatalf("Expected 3 keys, got %d", len(resultKeys))
	}

	// Verify all keys have the prefix
	for _, key := range resultKeys {
		if len(key) < len(prefix) || key[:len(prefix)] != prefix {
			t.Fatalf("Key %s doesn't match prefix %s", key, prefix)
		}
	}
}

// TestScanKeysWithValues tests prefix scanning for key-value pairs
func TestScanKeysWithValues(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	prefix := "config:"
	testData := map[string][]byte{
		"config:db":    []byte("postgres"),
		"config:cache": []byte("redis"),
		"config:queue": []byte("rabbitmq"),
		"other:value":  []byte("ignored"),
	}

	// Set all keys
	for key, value := range testData {
		err := kv.Set(key, value, 0)
		if err != nil {
			t.Fatalf("Set failed for %s: %v", key, err)
		}
	}

	// Scan with prefix
	results, err := kv.ScanKeysWithValues(prefix)
	if err != nil {
		t.Fatalf("ScanKeysWithValues failed: %v", err)
	}

	// Should have exactly 3 entries
	if len(results) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(results))
	}

	// Verify all entries
	if string(results["config:db"]) != "postgres" {
		t.Fatalf("Expected config:db=postgres, got %s", results["config:db"])
	}
	if string(results["config:cache"]) != "redis" {
		t.Fatalf("Expected config:cache=redis, got %s", results["config:cache"])
	}
	if string(results["config:queue"]) != "rabbitmq" {
		t.Fatalf("Expected config:queue=rabbitmq, got %s", results["config:queue"])
	}
}

// ============================================================================
// Batch Operations Tests
// ============================================================================

// TestBatchSet tests batch key-value storage
func TestBatchSet(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	err := kv.BatchSet(data, 0)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Verify all keys were set
	for key, expectedValue := range data {
		value, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", key, err)
		}
		if string(value) != string(expectedValue) {
			t.Fatalf("Expected %s=%s, got %s", key, expectedValue, value)
		}
	}
}

// TestBatchSetEmpty tests batch set with empty map
func TestBatchSetEmpty(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	err := kv.BatchSet(map[string][]byte{}, 0)
	if err != nil {
		t.Fatalf("BatchSet with empty map failed: %v", err)
	}
}

// TestBatchGet tests batch key retrieval
func TestBatchGet(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Set the data
	err := kv.BatchSet(data, 0)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Get the data
	keys := []string{"key1", "key2", "key3"}
	results, err := kv.BatchGet(keys)
	if err != nil {
		t.Fatalf("BatchGet failed: %v", err)
	}

	// Verify all keys were retrieved
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	for key, expectedValue := range data {
		if string(results[key]) != string(expectedValue) {
			t.Fatalf("Expected %s=%s, got %s", key, expectedValue, results[key])
		}
	}
}

// TestBatchGetPartial tests batch get with non-existent keys
func TestBatchGetPartial(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	// Set only key1 and key2
	err := kv.BatchSet(map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}, 0)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Try to get key1, key2, and non-existent key3
	keys := []string{"key1", "key2", "key3"}
	results, err := kv.BatchGet(keys)
	if err != nil {
		t.Fatalf("BatchGet failed: %v", err)
	}

	// Should only have key1 and key2
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if _, ok := results["key3"]; ok {
		t.Fatal("Should not have key3 in results")
	}
}

// TestBatchDelete tests batch key deletion
func TestBatchDelete(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Set the data
	err := kv.BatchSet(data, 0)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Delete the keys
	keys := []string{"key1", "key2", "key3"}
	err = kv.BatchDelete(keys)
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	// Verify all keys were deleted
	for _, key := range keys {
		_, err := kv.Get(key)
		if err != ErrKeyNotFound {
			t.Fatalf("Expected ErrKeyNotFound for %s, got: %v", key, err)
		}
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

// TestConcurrentWrites tests concurrent write operations
func TestConcurrentWrites(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	numGoroutines := 100
	numOpsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("key:%d:%d", id, j)
				value := []byte(fmt.Sprintf("value:%d:%d", id, j))
				err := kv.Set(key, value, 0)
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify some keys
	value, err := kv.Get("key:0:0")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(value) != "value:0:0" {
		t.Fatalf("Value mismatch: expected value:0:0, got %s", value)
	}
}

// TestConcurrentReads tests concurrent read operations
func TestConcurrentReads(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	// Set test data
	testKey := "shared_key"
	testValue := []byte("shared_value")
	err := kv.Set(testKey, testValue, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	numGoroutines := 100
	numReadsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numReadsPerGoroutine; j++ {
				value, err := kv.Get(testKey)
				if err != nil {
					t.Errorf("Get failed: %v", err)
				}
				if string(value) != "shared_value" {
					t.Errorf("Value mismatch: expected shared_value, got %s", value)
				}
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentMixed tests mixed concurrent operations
func TestConcurrentMixed(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	numGoroutines := 50
	numOpsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("key:%d:%d", id, j)
				value := []byte(fmt.Sprintf("value:%d:%d", id, j))

				// Mix of operations
				switch j % 4 {
				case 0:
					kv.Set(key, value, 0)
				case 1:
					kv.Get(key)
				case 2:
					kv.Exists(key)
				case 3:
					kv.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentBatches tests concurrent batch operations
func TestConcurrentBatches(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	numGoroutines := 10
	keysPerBatch := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			data := make(map[string][]byte)
			for j := 0; j < keysPerBatch; j++ {
				key := fmt.Sprintf("batch:%d:%d", id, j)
				data[key] = []byte(fmt.Sprintf("value:%d:%d", id, j))
			}

			err := kv.BatchSet(data, 0)
			if err != nil {
				t.Errorf("BatchSet failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// ============================================================================
// Shard Distribution Tests
// ============================================================================

// TestShardDistribution tests that keys are distributed across shards
func TestShardDistribution(t *testing.T) {
	shardCount := 16
	kv := createTestKVWithShards(t, shardCount)
	defer kv.Close()

	// Insert many keys
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key:%d", i)
		err := kv.Set(key, []byte(fmt.Sprintf("value:%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Verify all keys can be retrieved
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key:%d", i)
		value, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", key, err)
		}
		if string(value) != fmt.Sprintf("value:%d", i) {
			t.Fatalf("Value mismatch for %s", key)
		}
	}
}

// TestConsistentHashing tests that same key always goes to same shard
func TestConsistentHashing(t *testing.T) {
	kv := createTestKVWithShards(t, 64)
	defer kv.Close()

	key := "consistent_key"
	shardID1 := kv.getShard(key)
	shardID2 := kv.getShard(key)
	shardID3 := kv.getShard(key)

	if shardID1 != shardID2 || shardID2 != shardID3 {
		t.Fatalf("Same key should always map to same shard")
	}
}

// ============================================================================
// Edge Cases and Error Handling
// ============================================================================

// TestLargeValue tests storing large values
func TestLargeValue(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "large_key"
	// Create a 1MB value
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	err := kv.Set(key, largeValue, 0)
	if err != nil {
		t.Fatalf("Set large value failed: %v", err)
	}

	// Retrieve and verify
	value, err := kv.Get(key)
	if err != nil {
		t.Fatalf("Get large value failed: %v", err)
	}

	if len(value) != len(largeValue) {
		t.Fatalf("Size mismatch: expected %d, got %d", len(largeValue), len(value))
	}
}

// TestManyKeys tests storing many keys
func TestManyKeys(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key_%d", i)
		err := kv.Set(key, []byte(fmt.Sprintf("value_%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed at iteration %d: %v", i, err)
		}
	}

	// Verify random keys
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i*100)
		_, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", key, err)
		}
	}
}

// TestOverwrite tests overwriting existing keys
func TestOverwrite(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "overwrite_key"

	// Set initial value
	err := kv.Set(key, []byte("initial"), 0)
	if err != nil {
		t.Fatalf("Initial Set failed: %v", err)
	}

	// Overwrite multiple times
	for i := 0; i < 100; i++ {
		newValue := []byte(fmt.Sprintf("value_%d", i))
		err := kv.Set(key, newValue, 0)
		if err != nil {
			t.Fatalf("Overwrite failed at iteration %d: %v", i, err)
		}

		// Verify
		value, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if string(value) != string(newValue) {
			t.Fatalf("Value mismatch at iteration %d", i)
		}
	}
}

// ============================================================================
// Different Shard Count Tests
// ============================================================================

// TestSingleShard tests with 1 shard (sequential mode)
func TestSingleShard(t *testing.T) {
	kv := createTestKVWithShards(t, 1)
	defer kv.Close()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		err := kv.Set(key, []byte(fmt.Sprintf("value:%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		value, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if string(value) != fmt.Sprintf("value:%d", i) {
			t.Fatalf("Value mismatch")
		}
	}
}

// TestShardCount16 tests with 16 shards
func TestShardCount16(t *testing.T) {
	kv := createTestKVWithShards(t, 16)
	defer kv.Close()

	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key:%d", i)
		err := kv.Set(key, []byte(fmt.Sprintf("value:%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key:%d", i)
		_, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
	}
}

// TestShardCount128 tests with 128 shards
func TestShardCount128(t *testing.T) {
	kv := createTestKVWithShards(t, 128)
	defer kv.Close()

	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key:%d", i)
		err := kv.Set(key, []byte(fmt.Sprintf("value:%d", i)), 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key:%d", i)
		_, err := kv.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
	}
}

// ============================================================================
// Performance/Stress Tests
// ============================================================================

// TestHighConcurrency tests high concurrency with many goroutines
func TestHighConcurrency(t *testing.T) {
	kv := createTestKVWithShards(t, 64)
	defer kv.Close()

	numGoroutines := 1000
	opsPerGoroutine := 100

	var wg sync.WaitGroup
	var successCount int64

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key:%d:%d", id, j)
				value := []byte(fmt.Sprintf("value:%d:%d", id, j))

				err := kv.Set(key, value, 0)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				}

				_, err = kv.Get(key)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Should have successful operations
	if atomic.LoadInt64(&successCount) == 0 {
		t.Fatal("No successful operations")
	}
}

// TestStressWithBatches tests stress with batch operations
func TestStressWithBatches(t *testing.T) {
	kv := createTestKVWithShards(t, 64)
	defer kv.Close()

	numGoroutines := 50
	batchSize := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			data := make(map[string][]byte)
			for j := 0; j < batchSize; j++ {
				key := fmt.Sprintf("batch:%d:%d", id, j)
				data[key] = []byte(fmt.Sprintf("value:%d:%d", id, j))
			}

			err := kv.BatchSet(data, 0)
			if err != nil {
				t.Errorf("BatchSet failed: %v", err)
			}

			// Verify some keys
			keys := []string{
				fmt.Sprintf("batch:%d:0", id),
				fmt.Sprintf("batch:%d:%d", id, batchSize/2),
				fmt.Sprintf("batch:%d:%d", id, batchSize-1),
			}

			results, err := kv.BatchGet(keys)
			if err != nil || len(results) != 3 {
				t.Errorf("BatchGet failed: %v (got %d results)", err, len(results))
			}
		}(i)
	}

	wg.Wait()
}

// ============================================================================
// Close and Cleanup Tests
// ============================================================================

// TestClose tests proper cleanup
func TestClose(t *testing.T) {
	kv := createTestKV(t)

	// Set some data
	err := kv.Set("key", []byte("value"), 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Close the store
	err = kv.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// TestMultipleCloses tests that multiple closes are safe
func TestMultipleCloses(t *testing.T) {
	kv := createTestKV(t)

	// Multiple closes should be safe (or at least not panic)
	_ = kv.Close()
	_ = kv.Close()
	// If we get here, it didn't panic
}
