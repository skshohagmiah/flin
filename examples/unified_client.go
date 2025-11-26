package main

import (
	"fmt"
	"log"

	flin "github.com/skshohagmiah/flin/clients/go"
)

func main() {
	// Create unified client with both KV and Queue support
	opts := flin.DefaultOptions("localhost:6380")
	client, err := flin.NewClient(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Use KV operations
	fmt.Println("=== KV Operations ===")
	client.Set("mykey", []byte("myvalue"))
	value, _ := client.Get("mykey")
	fmt.Printf("GET mykey: %s\n", value)

	// Use Queue operations (if queue server is running)
	if client.Queue != nil {
		fmt.Println("\n=== Queue Operations ===")
		client.Queue.Push("tasks", []byte("task1"))
		client.Queue.Push("tasks", []byte("task2"))

		length, _ := client.Queue.Len("tasks")
		fmt.Printf("Queue length: %d\n", length)

		item, _ := client.Queue.Pop("tasks")
		fmt.Printf("Popped: %s\n", item)
	} else {
		fmt.Println("\nQueue client not available (queue server not running)")
	}
}
