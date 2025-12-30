package db

import (
	"testing"
	"time"
)

// Helper function to create a test DB
func createTestDB(t *testing.T) *Store {
	tmpDir := t.TempDir()
	db, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	return db
}

// TestInsert tests basic document insertion
func TestInsert(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	doc := Document{
		"name":  "John Doe",
		"age":   30,
		"email": "john@example.com",
	}

	id, err := db.Insert("users", doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	if id == "" {
		t.Fatal("Expected non-empty ID")
	}
}

// TestGet tests document retrieval
func TestGet(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert
	originalDoc := Document{
		"name": "John Doe",
		"age":  30,
	}

	id, err := db.Insert("users", originalDoc)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Retrieve
	retrievedDoc, err := db.Get("users", id)
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	if name, ok := retrievedDoc["name"].(string); !ok || name != "John Doe" {
		t.Errorf("Name mismatch: expected 'John Doe', got '%v'", retrievedDoc["name"])
	}

	if age, ok := retrievedDoc["age"].(float64); !ok || age != 30 {
		t.Errorf("Age mismatch: expected 30, got %v", retrievedDoc["age"])
	}
}

// TestFind tests document finding
func TestFind(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert multiple documents
	for i := 0; i < 5; i++ {
		doc := Document{
			"name": "User " + string(rune('A'+i)),
			"age":  20 + i,
		}
		db.Insert("users", doc)
	}

	// Find all
	results, err := db.Find("users", FindOptions{})
	if err != nil {
		t.Fatalf("Failed to find documents: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 documents, got %d", len(results))
	}

	// Find with filter
	opts := FindOptions{
		Filters: []Query{
			{Field: "age", Operator: "gt", Value: 22},
		},
	}
	results, err = db.Find("users", opts)
	if err != nil {
		t.Fatalf("Failed to find with filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 documents with age > 22, got %d", len(results))
	}
}

// TestUpdate tests document updates
func TestUpdate(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert
	doc := Document{
		"name": "John",
		"age":  30,
	}
	id, err := db.Insert("users", doc)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Update
	updateOpts := UpdateOptions{
		Set:   Document{"age": 31},
		Merge: true,
	}
	err = db.Update("users", id, updateOpts)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify
	updated, err := db.Get("users", id)
	if err != nil {
		t.Fatalf("Failed to get updated document: %v", err)
	}

	age, ok := updated["age"].(float64)
	if !ok || age != 31 {
		t.Errorf("Expected age 31, got %v", updated["age"])
	}

	// Name should still be there (merge)
	if updated["name"] != "John" {
		t.Errorf("Name was lost during update")
	}
}

// TestDelete tests document deletion
func TestDelete(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert
	doc := Document{"name": "John"}
	id, err := db.Insert("users", doc)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify it exists
	_, err = db.Get("users", id)
	if err != nil {
		t.Fatalf("Document should exist after insert")
	}

	// Delete
	err = db.Delete("users", id)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify it's gone
	_, err = db.Get("users", id)
	if err != ErrDocumentNotFound {
		t.Errorf("Expected ErrDocumentNotFound, got %v", err)
	}
}

// TestIndex tests index creation and usage
func TestIndex(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert documents with email field
	for i := 0; i < 10; i++ {
		doc := Document{
			"name":  "User " + string(rune('A'+i)),
			"email": "user" + string(rune('a'+i)) + "@example.com",
			"age":   20 + i,
		}
		db.Insert("users", doc)
	}

	// Create index on email
	err := db.CreateIndex("users", "email")
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Verify index exists
	indexes := db.ListIndexes("users")
	found := false
	for _, idx := range indexes {
		if idx == "email" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Index 'email' not found after creation")
	}
}

// TestCount tests counting documents
func TestCount(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert 3 documents
	for i := 0; i < 3; i++ {
		db.Insert("users", Document{"name": "User " + string(rune('A'+i))})
	}

	count, err := db.Count("users")
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

// TestDeleteMany tests deleting multiple documents
func TestDeleteMany(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert documents with different ages
	for i := 0; i < 5; i++ {
		doc := Document{
			"name": "User " + string(rune('A'+i)),
			"age":  20 + i,
		}
		db.Insert("users", doc)
	}

	// Delete all with age > 22
	opts := FindOptions{
		Filters: []Query{
			{Field: "age", Operator: "gt", Value: 22},
		},
	}

	deleted, err := db.DeleteMany("users", opts)
	if err != nil {
		t.Fatalf("Failed to delete many: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected to delete 2 documents, deleted %d", deleted)
	}

	// Verify remaining count
	count, err := db.Count("users")
	if err != nil {
		t.Fatalf("Failed to count after delete: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 remaining documents, got %d", count)
	}
}

// TestFindOne tests finding a single document
func TestFindOne(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert documents
	for i := 0; i < 3; i++ {
		db.Insert("users", Document{
			"name": "User " + string(rune('A'+i)),
			"age":  20 + i,
		})
	}

	// Find one with age = 21
	opts := FindOptions{
		Filters: []Query{
			{Field: "age", Operator: "eq", Value: 21},
		},
	}

	doc, err := db.FindOne("users", opts)
	if err != nil {
		t.Fatalf("Failed to find one: %v", err)
	}

	if age, ok := doc["age"].(float64); !ok || age != 21 {
		t.Errorf("Expected age 21, got %v", doc["age"])
	}
}

// TestTimestamps tests that documents have creation and update timestamps
func TestTimestamps(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert
	doc := Document{"name": "John"}
	id, err := db.Insert("users", doc)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Get and check timestamps
	retrieved, err := db.Get("users", id)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	createdAt, ok := retrieved["_created_at"].(float64)
	if !ok || createdAt == 0 {
		t.Errorf("_created_at missing or invalid: %v", retrieved["_created_at"])
	}

	updatedAt, ok := retrieved["_updated_at"].(float64)
	if !ok || updatedAt == 0 {
		t.Errorf("_updated_at missing or invalid: %v", retrieved["_updated_at"])
	}

	// Sleep and update to test timestamp change
	time.Sleep(10 * time.Millisecond)
	err = db.Update("users", id, UpdateOptions{Set: Document{"age": 25}, Merge: true})
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Check timestamps again
	// Small delay to ensure timestamp differs
	time.Sleep(10 * time.Millisecond)

	updated, err := db.Get("users", id)
	if err != nil {
		t.Fatalf("Failed to get updated: %v", err)
	}

	newUpdatedAt, ok := updated["_updated_at"].(float64)
	if !ok {
		t.Errorf("_updated_at missing after update: %v", updated["_updated_at"])
	}

	if newUpdatedAt <= updatedAt {
		t.Errorf("_updated_at should increase after update: %v -> %v", updatedAt, newUpdatedAt)
	}
}

// TestSorting tests document sorting
func TestSorting(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert documents in random order
	ages := []int{25, 30, 20, 35, 22}
	for _, age := range ages {
		db.Insert("users", Document{
			"name": "User",
			"age":  age,
		})
	}

	// Find with sort ascending
	opts := FindOptions{
		Sort: &SortOption{
			Field:     "age",
			Direction: "asc",
		},
	}
	results, err := db.Find("users", opts)
	if err != nil {
		t.Fatalf("Failed to find with sort: %v", err)
	}

	// Verify order
	for i := 0; i < len(results)-1; i++ {
		age1, _ := results[i]["age"].(float64)
		age2, _ := results[i+1]["age"].(float64)
		if age1 > age2 {
			t.Errorf("Sort order incorrect at position %d: %v > %v", i, age1, age2)
		}
	}

	// Find with sort descending
	opts.Sort.Direction = "desc"
	results, err = db.Find("users", opts)
	if err != nil {
		t.Fatalf("Failed to find with desc sort: %v", err)
	}

	// Verify reverse order
	for i := 0; i < len(results)-1; i++ {
		age1, _ := results[i]["age"].(float64)
		age2, _ := results[i+1]["age"].(float64)
		if age1 < age2 {
			t.Errorf("Reverse sort order incorrect at position %d: %v < %v", i, age1, age2)
		}
	}
}

// TestPagination tests skip and limit
func TestPagination(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert 10 documents
	for i := 0; i < 10; i++ {
		db.Insert("users", Document{
			"name":  "User " + string(rune('A'+i)),
			"order": i,
		})
	}

	// Skip 5, take 3
	opts := FindOptions{
		Skip:  5,
		Limit: 3,
	}
	results, err := db.Find("users", opts)
	if err != nil {
		t.Fatalf("Failed to find with pagination: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

// BenchmarkInsert benchmarks document insertion
func BenchmarkInsert(b *testing.B) {
	db := createTestDB(&testing.T{})
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Insert("users", Document{
			"name": "User",
			"age":  30,
		})
	}
}

// BenchmarkFind benchmarks finding documents
func BenchmarkFind(b *testing.B) {
	db := createTestDB(&testing.T{})
	defer db.Close()

	// Insert 1000 documents
	for i := 0; i < 1000; i++ {
		db.Insert("users", Document{
			"name": "User",
			"age":  20 + i%20,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Find("users", FindOptions{
			Filters: []Query{
				{Field: "age", Operator: "gt", Value: 25},
			},
		})
	}
}

// BenchmarkUpdate benchmarks document updates
func BenchmarkUpdate(b *testing.B) {
	db := createTestDB(&testing.T{})
	defer db.Close()

	// Insert 1 document
	id, _ := db.Insert("users", Document{"name": "User", "age": 30})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Update("users", id, UpdateOptions{
			Set:   Document{"age": 30 + i},
			Merge: true,
		})
	}
}
