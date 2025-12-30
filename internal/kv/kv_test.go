package kv

import (
	"testing"
	"time"
)

// ============================================================================
// Helper Functions
// ============================================================================

func createTestKV(t *testing.T) *KVStore {
	kv, err := New("/tmp/testkvstore")
	if err != nil {
		t.Fatalf("Failed to create KVStore: %v", err)
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
		t.Fatalf("Expected value %s, got %s", expectedValue, value)
	}
}

// TestGetNonExistentKey tests retrieval of a non-existent key
func TestGetNonExistentKey(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	_, err := kv.Get("nonexistent")
	if err != ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound for non-existent key, got: %v", err)
	}
}

// TestDelete tests basic key deletion
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

	// Delete the key
	err = kv.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Try to get the deleted key
	_, err = kv.Get(key)
	if err != ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound for deleted key, got: %v", err)
	}
}

// TestDeleteNonExistentKey tests deletion of a non-existent key
func TestDeleteNonExistentKey(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	err := kv.Delete("nonexistent")
	if err != ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound for non-existent key deletion, got: %v", err)
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

// ============================================================================
// Expiration Tests
// ============================================================================

// TestSetWithExpiration tests setting a key with expiration
func TestSetWithExpiration(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "expiringkey"
	value := []byte("expiringvalue")
	expiration := 2 * time.Second

	err := kv.Set(key, value, expiration)
	if err != nil {
		t.Fatalf("Set with expiration failed: %v", err)
	}

	// Get the value before expiration
	retrievedValue, err := kv.Get(key)
	if err != nil {
		t.Fatalf("Get before expiration failed: %v", err)
	}
	if string(retrievedValue) != string(value) {
		t.Fatalf("Expected value %s, got %s", value, retrievedValue)
	}

	// Wait for expiration
	time.Sleep(expiration + 1*time.Second)

	// Try to get the value after expiration
	_, err = kv.Get(key)
	if err != ErrKeyNotFound {
		t.Fatalf("Expected ErrKeyNotFound after expiration, got: %v", err)
	}
}

// TestSetWithImmediateExpiration tests setting a key with immediate expiration
func TestSetWithImmediateExpiration(t *testing.T) {
	kv := createTestKV(t)
	defer kv.Close()

	key := "immediatekey"
	value := []byte("immediatevalue")

	err := kv.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set with immediate expiration failed: %v", err)
	}

	// Try to get the value immediately
	retrievedValue, err := kv.Get(key)
	if err != nil {
		t.Fatalf("Get after immediate expiration failed: %v", err)
	}
	if string(retrievedValue) != string(value) {
		t.Fatalf("Expected value %s, got %s", value, retrievedValue)
	}
}
