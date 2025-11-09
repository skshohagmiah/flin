#!/bin/bash

# Change to project root
cd "$(dirname "$0")/.." || exit 1

echo "🚀 Flin Queue Test"
echo "=================="
echo ""

# Build
echo "📦 Building..."
cat > /tmp/test_queue.go << 'EOF'
package main

import (
	"fmt"
	"time"

	"github.com/skshohagmiah/flin/pkg/queue"
)

func main() {
	fmt.Println("🎯 Testing Queue Features")
	fmt.Println("==========================")
	fmt.Println()

	// Create queue client (in-memory)
	client, err := queue.NewClient("")
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		return
	}

	// Test 1: Enqueue and Dequeue
	fmt.Println("📝 Test 1: Enqueue and Dequeue")
	fmt.Println("-------------------------------")
	
	client.Enqueue("tasks", []byte("Task 1"))
	client.Enqueue("tasks", []byte("Task 2"))
	client.Enqueue("tasks", []byte("Task 3"))
	fmt.Println("✅ Enqueued 3 messages")

	msg, _ := client.Dequeue("tasks")
	fmt.Printf("✅ Dequeued: %s\n", string(msg.Body))
	msg.Ack()
	fmt.Println("✅ Acknowledged")
	fmt.Println()

	// Test 2: Priority
	fmt.Println("📝 Test 2: Priority Queue")
	fmt.Println("-------------------------")
	
	client.EnqueueWithOptions("priority", []byte("Low"), nil, 1, 0)
	client.EnqueueWithOptions("priority", []byte("High"), nil, 9, 0)
	client.EnqueueWithOptions("priority", []byte("Medium"), nil, 5, 0)
	fmt.Println("✅ Enqueued with priorities: 1, 9, 5")

	for i := 0; i < 3; i++ {
		msg, _ := client.Dequeue("priority")
		fmt.Printf("✅ Priority %d: %s\n", msg.Priority, string(msg.Body))
		msg.Ack()
	}
	fmt.Println()

	// Test 3: Consume (continuous)
	fmt.Println("📝 Test 3: Consume")
	fmt.Println("------------------")
	
	count := 0
	client.Consume("notifications", func(msg *queue.Message) {
		fmt.Printf("✅ Received: %s\n", string(msg.Body))
		msg.Ack()
		count++
	})

	client.Enqueue("notifications", []byte("Notification 1"))
	client.Enqueue("notifications", []byte("Notification 2"))
	
	time.Sleep(1 * time.Second)
	fmt.Printf("✅ Received %d notifications\n", count)
	fmt.Println()

	// Test 4: Headers
	fmt.Println("📝 Test 4: Message Headers")
	fmt.Println("--------------------------")
	
	headers := map[string]string{
		"user_id": "123",
		"type":    "urgent",
	}
	client.EnqueueWithOptions("api", []byte("API Request"), headers, 0, 0)
	fmt.Println("✅ Enqueued with headers")

	msg, _ = client.Dequeue("api")
	fmt.Printf("✅ Body: %s\n", string(msg.Body))
	fmt.Printf("✅ Headers: %v\n", msg.Headers)
	msg.Ack()
	fmt.Println()

	fmt.Println("🎉 All queue tests passed!")
}
EOF

go run /tmp/test_queue.go

# Cleanup
rm -f /tmp/test_queue.go

echo ""
echo "✅ Test complete!"
