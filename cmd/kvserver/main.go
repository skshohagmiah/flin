package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/skshohagmiah/flin/internal/kv"
	"github.com/skshohagmiah/flin/internal/server"
)

func main() {
	// Create KV store
	tmpDir := "/tmp/flin-server"
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	
	store, err := kv.New(tmpDir)
	if err != nil {
		log.Fatalf("Failed to create KV store: %v", err)
	}
	defer store.Close()
	
	// Create NATS-style server
	srv, err := server.NewKVServer(store, ":6380")
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
		os.Exit(0)
	}()
	
	// Start server
	fmt.Println("🚀 Flin KV Server (NATS-style architecture)")
	fmt.Println("   - Per-connection goroutines (readLoop + writeLoop)")
	fmt.Println("   - Lock-free inline processing")
	fmt.Println("   - Buffered channels for async dispatch")
	fmt.Println()
	
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
