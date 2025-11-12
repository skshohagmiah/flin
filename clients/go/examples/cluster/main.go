package main

import (
	"fmt"
	"log"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	fmt.Println("🚀 Flin Go SDK - Cluster Example")
	fmt.Println("==================================")

	// Create cluster-aware client (smart routing with topology discovery)
	opts := flin.DefaultClusterOptions([]string{
		"localhost:7080",
		"localhost:7081",
		"localhost:7082",
	})

	client, err := flin.NewClient(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("✅ Connected to cluster")
	fmt.Printf("Client mode: %s\n", map[bool]string{true: "Cluster", false: "Single-node"}[client.IsClusterMode()])

	// Get topology info
	topology := client.GetTopology()
	fmt.Printf("\n📊 Cluster Topology:\n")
	fmt.Printf("   Nodes: %d\n", len(topology.Nodes))
	fmt.Printf("   Partitions: %d\n", len(topology.PartitionMap))

	// Set values (automatically routed to correct nodes)
	fmt.Println("\n📝 Setting values across cluster...")

	err = client.Set("user:1", []byte("Alice"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Set("user:2", []byte("Bob"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Set("user:3", []byte("Charlie"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ Values set across cluster")

	// Get values (automatically routed)
	fmt.Println("\n📖 Getting values from cluster...")

	value, err := client.Get("user:1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   user:1 = %s\n", value)

	// Batch operations (distributed across nodes)
	fmt.Println("\n📦 Batch operations across cluster...")

	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	values := [][]byte{
		[]byte("value1"),
		[]byte("value2"),
		[]byte("value3"),
		[]byte("value4"),
		[]byte("value5"),
	}

	err = client.MSet(keys, values)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Batch set distributed across nodes")

	results, err := client.MGet(keys)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Batch get from multiple nodes:")
	for i, result := range results {
		fmt.Printf("   %s: %s\n", keys[i], result)
	}

	// Counter across cluster
	fmt.Println("\n🔢 Counter operations...")

	// Initialize counter
	err = client.Set("global:counter", []byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err != nil {
		log.Fatal(err)
	}

	count, err := client.Incr("global:counter")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Counter incremented: %d\n", count)

	// Clean up
	fmt.Println("\n🗑️  Cleaning up...")
	client.Delete("user:1")
	client.Delete("user:2")
	client.Delete("user:3")
	client.MDelete(keys)
	client.Delete("global:counter")

	fmt.Println("\n✨ Cluster example completed!")
}
