package client_test

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/skshohagmiah/flin/pkg/client"
)

// Example_basic demonstrates basic client usage
func Example_basic() {
	// Create client
	c, err := client.New("localhost:6380")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Set a key
	err = c.Set("mykey", []byte("hello world"))
	if err != nil {
		log.Fatal(err)
	}

	// Get a key
	value, err := c.Get("mykey")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(value))

	// Output: hello world
}

// Example_pool demonstrates connection pooling
func Example_pool() {
	// Create pooled client
	config := client.DefaultPoolConfig()
	config.Address = "localhost:6380"
	config.MinConns = 5
	config.MaxConns = 20

	pc, err := client.NewPooledClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	// Use like regular client
	pc.Set("key1", []byte("value1"))
	value, _ := pc.Get("key1")
	fmt.Println(string(value))

	// Check pool stats
	stats := pc.Stats()
	fmt.Printf("Available connections: %v\n", stats["available"])

	// Output:
	// value1
	// Available connections: 5
}

// Example_concurrent demonstrates concurrent operations
func Example_concurrent() {
	pc, err := client.NewPooledClient(client.DefaultPoolConfig())
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	var wg sync.WaitGroup
	numOps := 10

	// Concurrent writes
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key_%d", i)
			pc.Set(key, []byte(fmt.Sprintf("value_%d", i)))
		}(i)
	}

	wg.Wait()

	// Verify
	value, _ := pc.Get("key_0")
	fmt.Println(string(value))

	// Output: value_0
}

// Example_healthCheck demonstrates health checking
func Example_healthCheck() {
	c, err := client.New("localhost:6380")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Check if server is alive
	err = c.Ping()
	if err != nil {
		fmt.Println("Server is down")
	} else {
		fmt.Println("Server is alive")
	}

	// Output: Server is alive
}

// Example_reconnect demonstrates auto-reconnection
func Example_reconnect() {
	c, err := client.New("localhost:6380")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Simulate connection loss and reconnect
	if !c.IsConnected() {
		err = c.Reconnect()
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Reconnected successfully")
	// Output: Reconnected successfully
}

// Example_exists demonstrates checking key existence
func Example_exists() {
	c, err := client.New("localhost:6380")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Set a key
	c.Set("testkey", []byte("testvalue"))

	// Check if exists
	exists, _ := c.Exists("testkey")
	fmt.Printf("Key exists: %v\n", exists)

	// Delete and check again
	c.Delete("testkey")
	exists, _ = c.Exists("testkey")
	fmt.Printf("Key exists after delete: %v\n", exists)

	// Output:
	// Key exists: true
	// Key exists after delete: false
}

// Example_timeout demonstrates timeout handling
func Example_timeout() {
	config := &client.Config{
		Address:    "localhost:6380",
		Timeout:    1 * time.Second,
		MaxRetries: 3,
	}

	c, err := client.NewWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Operations will timeout after 1 second
	err = c.Set("key", []byte("value"))
	if err != nil {
		fmt.Println("Operation timed out")
	} else {
		fmt.Println("Operation succeeded")
	}

	// Output: Operation succeeded
}

// Example_poolStats demonstrates pool statistics
func Example_poolStats() {
	config := &client.PoolConfig{
		Address:  "localhost:6380",
		MinConns: 3,
		MaxConns: 10,
		Timeout:  5 * time.Second,
	}

	pc, err := client.NewPooledClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	// Get stats
	stats := pc.Stats()
	fmt.Printf("Min connections: %v\n", stats["min"])
	fmt.Printf("Max connections: %v\n", stats["max"])
	fmt.Printf("Available: %v\n", stats["available"])

	// Output:
	// Min connections: 3
	// Max connections: 10
	// Available: 3
}
