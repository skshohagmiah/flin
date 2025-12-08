package queue

import (
	"testing"
	"time"
)

func TestMessageAcknowledgment(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-ack-queue"

	// Push a message
	msgID, err := q.PushMessage(queueName, []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to push message: %v", err)
	}

	if msgID == "" {
		t.Fatal("Expected non-empty message ID")
	}

	// Pop the message
	msg, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop message: %v", err)
	}

	if string(msg.Value) != "test message" {
		t.Errorf("Expected 'test message', got '%s'", string(msg.Value))
	}

	if msg.DeliveryCount != 1 {
		t.Errorf("Expected delivery count 1, got %d", msg.DeliveryCount)
	}

	// Acknowledge the message
	err = msg.Ack()
	if err != nil {
		t.Fatalf("Failed to ack message: %v", err)
	}

	// Try to pop again - should be empty
	_, err = q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err == nil {
		t.Fatal("Expected queue to be empty after ack")
	}
}

func TestMessageNackRequeue(t *testing.T) {
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-nack-queue"

	// Push a message
	_, err = q.PushMessage(queueName, []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to push message: %v", err)
	}

	// Pop the message
	msg, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop message: %v", err)
	}

	// Nack with requeue
	err = msg.Nack(true)
	if err != nil {
		t.Fatalf("Failed to nack message: %v", err)
	}

	// Pop again - should get the same message
	msg2, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop requeued message: %v", err)
	}

	if string(msg2.Value) != "test message" {
		t.Errorf("Expected 'test message', got '%s'", string(msg2.Value))
	}

	if msg2.DeliveryCount != 2 {
		t.Errorf("Expected delivery count 2, got %d", msg2.DeliveryCount)
	}

	// Clean up
	msg2.Ack()
}

func TestMessageNackNoRequeue(t *testing.T) {
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-nack-no-requeue"

	// Push a message
	_, err = q.PushMessage(queueName, []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to push message: %v", err)
	}

	// Pop the message
	msg, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop message: %v", err)
	}

	// Nack without requeue (delete)
	err = msg.Nack(false)
	if err != nil {
		t.Fatalf("Failed to nack message: %v", err)
	}

	// Try to pop again - should be empty
	_, err = q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err == nil {
		t.Fatal("Expected queue to be empty after nack without requeue")
	}
}

func TestVisibilityTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-visibility-timeout"

	// Push a message
	_, err = q.PushMessage(queueName, []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to push message: %v", err)
	}

	// Pop with short visibility timeout
	_, err = q.PopMessage(queueName, "consumer-1", 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop message: %v", err)
	}

	// Wait for visibility timeout to expire
	time.Sleep(2 * time.Second)

	// Manually trigger requeue
	count, err := q.RequeueExpiredMessages(queueName)
	if err != nil {
		t.Fatalf("Failed to requeue expired messages: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 requeued message, got %d", count)
	}

	// Pop again - should get the same message
	msg2, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop requeued message: %v", err)
	}

	if string(msg2.Value) != "test message" {
		t.Errorf("Expected 'test message', got '%s'", string(msg2.Value))
	}

	// Clean up
	msg2.Ack()
}

func TestDeadLetterQueue(t *testing.T) {
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-main-queue"
	dlqName := "test-dlq"

	// Configure DLQ with max 3 retries
	err = q.SetDeadLetterQueue(queueName, dlqName, 3)
	if err != nil {
		t.Fatalf("Failed to set DLQ: %v", err)
	}

	// Push a message
	_, err = q.PushMessage(queueName, []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to push message: %v", err)
	}

	// Nack 3 times (should move to DLQ on 3rd nack)
	for i := 0; i < 3; i++ {
		msg, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
		if err != nil {
			t.Fatalf("Failed to pop message (attempt %d): %v", i+1, err)
		}

		err = msg.Nack(true)
		if err != nil {
			t.Fatalf("Failed to nack message (attempt %d): %v", i+1, err)
		}
	}

	// Main queue should be empty
	_, err = q.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err == nil {
		t.Fatal("Expected main queue to be empty")
	}

	// DLQ should have the message
	dlqMsg, err := q.PopMessage(dlqName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop from DLQ: %v", err)
	}

	if string(dlqMsg.Value) != "test message" {
		t.Errorf("Expected 'test message' in DLQ, got '%s'", string(dlqMsg.Value))
	}

	// Clean up
	dlqMsg.Ack()
}

func TestConcurrentConsumers(t *testing.T) {
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-concurrent-queue"

	// Push 10 messages
	for i := 0; i < 10; i++ {
		_, err = q.PushMessage(queueName, []byte("message"))
		if err != nil {
			t.Fatalf("Failed to push message %d: %v", i, err)
		}
	}

	// Process with 3 concurrent consumers
	done := make(chan bool, 3)
	processed := make(chan int, 10)

	for c := 0; c < 3; c++ {
		consumerID := string(rune('A' + c))
		go func(id string) {
			defer func() { done <- true }()

			for {
				msg, err := q.PopMessage(queueName, id, 30*time.Second)
				if err != nil {
					return // Queue empty
				}

				processed <- 1
				msg.Ack()
			}
		}(consumerID)
	}

	// Wait for all consumers to finish
	for i := 0; i < 3; i++ {
		<-done
	}

	close(processed)

	// Count processed messages
	count := 0
	for range processed {
		count++
	}

	if count != 10 {
		t.Errorf("Expected 10 processed messages, got %d", count)
	}
}

func TestLegacyMethods(t *testing.T) {
	tmpDir := t.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "test-legacy-queue"

	// Test legacy Push
	err = q.Push(queueName, []byte("legacy message"))
	if err != nil {
		t.Fatalf("Failed to push with legacy method: %v", err)
	}

	// Test legacy Pop
	value, err := q.Pop(queueName)
	if err != nil {
		t.Fatalf("Failed to pop with legacy method: %v", err)
	}

	if string(value) != "legacy message" {
		t.Errorf("Expected 'legacy message', got '%s'", string(value))
	}
}

func TestVisibilityTimeoutWorker(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	queueName := "test-worker-queue"

	// Create worker with short check interval
	worker := NewVisibilityTimeoutWorker(storage, 500*time.Millisecond)
	worker.AddQueue(queueName)
	worker.Start()
	defer worker.Stop()

	// Push and pop a message with short visibility timeout
	_, err = storage.PushMessage(queueName, []byte("test message"))
	if err != nil {
		t.Fatalf("Failed to push message: %v", err)
	}

	_, err = storage.PopMessage(queueName, "consumer-1", 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop message: %v", err)
	}

	// Wait for worker to requeue (should happen after ~1.5 seconds)
	time.Sleep(2 * time.Second)

	// Try to pop again - should get the requeued message
	msg2, err := storage.PopMessage(queueName, "consumer-1", 30*time.Second)
	if err != nil {
		t.Fatalf("Failed to pop requeued message: %v", err)
	}

	if string(msg2.Value) != "test message" {
		t.Errorf("Expected 'test message', got '%s'", string(msg2.Value))
	}

	// Clean up
	storage.AckMessage(queueName, msg2.ID)
}

func BenchmarkQueueWithAck(b *testing.B) {
	tmpDir := b.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "bench-queue"
	value := []byte("benchmark message")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Push
		_, err := q.PushMessage(queueName, value)
		if err != nil {
			b.Fatalf("Failed to push: %v", err)
		}

		// Pop
		msg, err := q.PopMessage(queueName, "consumer-1", 30*time.Second)
		if err != nil {
			b.Fatalf("Failed to pop: %v", err)
		}

		// Ack
		err = msg.Ack()
		if err != nil {
			b.Fatalf("Failed to ack: %v", err)
		}
	}
}

func BenchmarkQueueLegacy(b *testing.B) {
	tmpDir := b.TempDir()

	q, err := New(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create queue: %v", err)
	}
	defer q.Close()

	queueName := "bench-legacy-queue"
	value := []byte("benchmark message")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Push
		err := q.Push(queueName, value)
		if err != nil {
			b.Fatalf("Failed to push: %v", err)
		}

		// Pop
		_, err = q.Pop(queueName)
		if err != nil {
			b.Fatalf("Failed to pop: %v", err)
		}
	}
}
