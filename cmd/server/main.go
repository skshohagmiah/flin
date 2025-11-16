package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skshohagmiah/clusterkit"
	"github.com/skshohagmiah/flin/internal/kv"
	"github.com/skshohagmiah/flin/internal/server"
)

var (
	nodeID         = flag.String("node-id", "", "Node ID (required)")
	httpAddr       = flag.String("http", ":8080", "HTTP address for cluster coordination")
	raftAddr       = flag.String("raft", ":9080", "Raft address for cluster consensus")
	joinAddr       = flag.String("join", "", "Address of node to join (empty for bootstrap)")
	dataDir        = flag.String("data", "./data", "Data directory")
	kvPort         = flag.String("port", ":6380", " Server port")
	partitionCount = flag.Int("partitions", 64, "Number of partitions")
	workerCount    = flag.Int("workers", 256, "Number of worker goroutines")
	useMemory      = flag.Bool("memory", false, "Use in-memory storage (like Redis)")
)

func main() {
	flag.Parse()

	if *nodeID == "" {
		fmt.Println("Error: -node-id is required")
		fmt.Println("\nFlin is a distributed KV store. Usage:")
		fmt.Println("  ./server -node-id=node-1 -http=:8080 -raft=:9080 -port=:6380")
		fmt.Println("  ./server -node-id=node-2 -http=:8081 -raft=:9081 -port=:6381 -join=localhost:8080")
		os.Exit(1)
	}

	fmt.Println("🚀 Flin Distributed KV Store")
	fmt.Println("   - ClusterKit coordination")
	fmt.Println("   - Raft consensus")
	fmt.Println("   - Automatic partitioning & replication")
	fmt.Println()
	fmt.Printf("   Node ID:  %s\n", *nodeID)
	fmt.Printf("   HTTP:     %s\n", *httpAddr)
	fmt.Printf("   Raft:     %s\n", *raftAddr)
	fmt.Printf("   KV Port:  %s\n", *kvPort)

	if *useMemory {
		fmt.Printf("   Storage:  IN-MEMORY (like Redis)\n")
		fmt.Printf("   ⚠️  Data will be lost on restart!\n")
	} else {
		fmt.Printf("   Storage:  DISK (BadgerDB)\n")
		fmt.Printf("   Data Dir: %s\n", *dataDir)
	}

	if *joinAddr != "" {
		fmt.Printf("   Join:     %s\n", *joinAddr)
	} else {
		fmt.Printf("   Bootstrap: true (first node)\n")
	}
	fmt.Println()

	// Create local KV store (memory or disk)
	var store *kv.KVStore
	var err error

	if *useMemory {
		fmt.Println("📦 Creating in-memory KV store...")
		store, err = kv.NewMemory()
		if err != nil {
			log.Fatalf("Failed to create in-memory store: %v", err)
		}
	} else {
		kvDataDir := *dataDir + "/kv"
		fmt.Printf("📦 Creating disk-based KV store at %s...\n", kvDataDir)
		store, err = kv.New(kvDataDir)
		if err != nil {
			log.Fatalf("Failed to create KV store: %v", err)
		}
	}
	defer store.Close()

	// Create ClusterKit instance
	ckOptions := clusterkit.Options{
		NodeID:            *nodeID,
		HTTPAddr:          *httpAddr,
		RaftAddr:          *raftAddr,
		JoinAddr:          *joinAddr,
		Bootstrap:         *joinAddr == "", // Bootstrap if not joining
		DataDir:           *dataDir + "/cluster",
		PartitionCount:    *partitionCount,
		ReplicationFactor: 3,
		HealthCheck: clusterkit.HealthCheckConfig{
			Enabled:          true,
			Interval:         5 * time.Second,
			Timeout:          2 * time.Second,
			FailureThreshold: 3,
		},
	}

	ck, err := clusterkit.NewClusterKit(ckOptions)
	if err != nil {
		log.Fatalf("Failed to create ClusterKit: %v", err)
	}

	// Start ClusterKit
	if err := ck.Start(); err != nil {
		log.Fatalf("Failed to start ClusterKit: %v", err)
	}

	log.Printf("✅ ClusterKit started")

	// Create  server with custom worker count
	srv, err := server.NewKVServerWithWorkers(store, ck, *kvPort, *nodeID, *workerCount)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down server...")
		srv.Stop()
		ck.Stop()
		store.Close()
		os.Exit(0)
	}()

	// Start server
	log.Printf("🚀 Server listening on %s", *kvPort)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
