package main

import (
	"fmt"
	"log"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	fmt.Println("🚀 Flin Go SDK - Basic Example")
	fmt.Println("================================")

	// Create client (smart routing, single-node mode)
	opts := flin.DefaultOptions("localhost:7380")
	client, err := flin.NewClient(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Printf("Client mode: %s\n", map[bool]string{true: "Cluster", false: "Single-node"}[client.IsClusterMode()])

	// Set a value
	fmt.Println("\n📝 Setting key 'greeting'...")
	err = client.Set("greeting", []byte("Hello, Flin!"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Set successful")

	// Get a value
	fmt.Println("\n📖 Getting key 'greeting'...")
	value, err := client.Get("greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Value: %s\n", value)

	// Check if key exists
	fmt.Println("\n🔍 Checking if key exists...")
	exists, err := client.Exists("greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Exists: %v\n", exists)

	// Counter operations
	fmt.Println("\n🔢 Counter operations...")

	// Initialize counter (8 bytes for int64)
	err = client.Set("counter", []byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		log.Fatal(err)
	}

	// Increment
	count, err := client.Incr("counter")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ After increment: %d\n", count)

	// Increment again
	count, err = client.Incr("counter")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ After second increment: %d\n", count)

	// Decrement
	count, err = client.Decr("counter")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ After decrement: %d\n", count)

	// Batch operations
	fmt.Println("\n📦 Batch operations...")

	keys := []string{"user:1", "user:2", "user:3"}
	values := [][]byte{
		[]byte("Alice"),
		[]byte("Bob"),
		[]byte("Charlie"),
	}

	err = client.MSet(keys, values)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Batch set successful")

	results, err := client.MGet(keys)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Batch get results:")
	for i, result := range results {
		fmt.Printf("   %s: %s\n", keys[i], result)
	}

	// Delete
	fmt.Println("\n🗑️  Deleting keys...")
	err = client.MDelete(keys)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Batch delete successful")

	// Clean up
	client.Delete("greeting")
	client.Delete("counter")

	// // Show pool stats
	// stats := client.Stats()
	// fmt.Printf("\n📊 Connection Pool Stats:\n")
	// fmt.Printf("   Active: %d/%d\n", stats.ActiveCount, stats.MaxSize)
	// fmt.Printf("   Available: %d\n", stats.AvailableCount)

	fmt.Println("\n✨ Example completed!")
}
