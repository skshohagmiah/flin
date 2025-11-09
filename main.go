package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/skshohagmiah/flin/internal/kv"
)

func main() {
	// Create a temporary directory for demo
	tmpDir := "./flin-data"
	defer os.RemoveAll(tmpDir)

	// Initialize KV store
	store, err := kv.New(tmpDir)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	fmt.Println("🚀 Flin KV Store Demo")
	fmt.Println("=====================")

	// Set a key-value pair
	fmt.Println("\n1. Setting key 'name' = 'Flin'")
	err = store.Set("name", []byte("Flin"), 0)
	if err != nil {
		log.Fatal(err)
	}

	// Get the value
	fmt.Println("2. Getting key 'name'")
	value, err := store.Get("name")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Value: %s\n", string(value))

	// Set with TTL
	fmt.Println("\n3. Setting key 'temp' with 5 second TTL")
	err = store.Set("temp", []byte("temporary"), 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	// Check if key exists
	fmt.Println("4. Checking if 'temp' exists")
	exists, err := store.Exists("temp")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Exists: %v\n", exists)

	// Increment counter
	fmt.Println("\n5. Incrementing counter 3 times")
	for i := 0; i < 3; i++ {
		err = store.Incr("counter")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Scan with prefix
	fmt.Println("\n6. Setting multiple keys with prefix 'user:'")
	store.Set("user:1", []byte("Alice"), 0)
	store.Set("user:2", []byte("Bob"), 0)
	store.Set("user:3", []byte("Charlie"), 0)

	fmt.Println("7. Scanning keys with prefix 'user:'")
	values, err := store.Scan("user:")
	if err != nil {
		log.Fatal(err)
	}
	for i, val := range values {
		fmt.Printf("   User %d: %s\n", i+1, string(val))
	}

	// Delete a key
	fmt.Println("\n8. Deleting key 'name'")
	err = store.Delete("name")
	if err != nil {
		log.Fatal(err)
	}

	exists, _ = store.Exists("name")
	fmt.Printf("   'name' exists after delete: %v\n", exists)

	fmt.Println("\n✅ Demo completed successfully!")
}
