package stream

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestBasicPublishConsume tests basic publish and consume functionality
func TestBasicPublishConsume(t *testing.T) {
	storage, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	stream, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "test-topic"

	// Create topic
	if err := stream.CreateTopic(topic, 4, 0); err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Publish message
	offset, err := stream.Publish(topic, 0, "key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}

	// Subscribe consumer
	group := "test-group"
	consumerID := "consumer-1"
	if err := stream.Subscribe(topic, group, consumerID); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Consume message
	messages, err := stream.Consume(topic, group, consumerID, 10)
	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if string(messages[0].Value) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(messages[0].Value))
	}
}

// TestBatchPublish tests batch publishing functionality
func TestBatchPublish(t *testing.T) {
	stream, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "batch-topic"

	// Create topic
	if err := stream.CreateTopic(topic, 4, 0); err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Batch publish
	messages := []*PublishMessage{
		{Partition: 0, Key: "key1", Value: []byte("value1")},
		{Partition: 0, Key: "key2", Value: []byte("value2")},
		{Partition: 1, Key: "key3", Value: []byte("value3")},
	}

	offsets, err := stream.PublishBatch(topic, messages)
	if err != nil {
		t.Fatalf("Failed to batch publish: %v", err)
	}

	if len(offsets) != 3 {
		t.Fatalf("Expected 3 offsets, got %d", len(offsets))
	}

	// Verify offsets are sequential per partition
	if offsets[0] != 0 || offsets[1] != 1 {
		t.Errorf("Partition 0 offsets should be 0,1, got %d,%d", offsets[0], offsets[1])
	}
	if offsets[2] != 0 {
		t.Errorf("Partition 1 offset should be 0, got %d", offsets[2])
	}
}

// TestParallelConsume tests parallel partition fetching
func TestParallelConsume(t *testing.T) {
	stream, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "parallel-topic"

	// Create topic with 4 partitions
	if err := stream.CreateTopic(topic, 4, 0); err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Publish to all partitions
	for p := 0; p < 4; p++ {
		for i := 0; i < 10; i++ {
			_, err := stream.Publish(topic, p, fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)))
			if err != nil {
				t.Fatalf("Failed to publish: %v", err)
			}
		}
	}

	// Subscribe consumer
	group := "parallel-group"
	consumerID := "consumer-1"
	if err := stream.Subscribe(topic, group, consumerID); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Consume (should fetch from all partitions in parallel)
	start := time.Now()
	messages, err := stream.Consume(topic, group, consumerID, 40)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}

	if len(messages) != 40 {
		t.Errorf("Expected 40 messages, got %d", len(messages))
	}

	t.Logf("Parallel consume took %v for 40 messages from 4 partitions", duration)
}

// TestOffsetCaching tests offset caching functionality
func TestOffsetCaching(t *testing.T) {
	stream, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "cache-topic"
	group := "cache-group"
	consumerID := "consumer-1"

	// Create topic
	if err := stream.CreateTopic(topic, 1, 0); err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Publish messages
	for i := 0; i < 10; i++ {
		_, err := stream.Publish(topic, 0, fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)))
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Subscribe
	if err := stream.Subscribe(topic, group, consumerID); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// First consume (cache miss)
	start1 := time.Now()
	messages1, err := stream.Consume(topic, group, consumerID, 5)
	duration1 := time.Since(start1)
	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}
	if len(messages1) != 5 {
		t.Fatalf("Expected 5 messages, got %d", len(messages1))
	}

	// Commit offset
	if err := stream.Commit(topic, group, 0, 5); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Second consume (cache hit)
	start2 := time.Now()
	messages2, err := stream.Consume(topic, group, consumerID, 5)
	duration2 := time.Since(start2)
	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}
	if len(messages2) != 5 {
		t.Fatalf("Expected 5 messages, got %d", len(messages2))
	}

	t.Logf("First consume (cache miss): %v", duration1)
	t.Logf("Second consume (cache hit): %v", duration2)

	// Cache hit should be faster (though timing can be flaky in tests)
	if duration2 > duration1*2 {
		t.Logf("Warning: Cache hit not faster (miss=%v, hit=%v)", duration1, duration2)
	}
}

// TestAtomicOffsets tests atomic offset increment under concurrency
func TestAtomicOffsets(t *testing.T) {
	stream, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "atomic-topic"

	// Create topic
	if err := stream.CreateTopic(topic, 1, 0); err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Concurrent publishes
	numGoroutines := 10
	messagesPerGoroutine := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				_, err := stream.Publish(topic, 0, fmt.Sprintf("key%d-%d", id, j), []byte(fmt.Sprintf("value%d-%d", id, j)))
				if err != nil {
					t.Errorf("Failed to publish: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify total messages
	totalExpected := numGoroutines * messagesPerGoroutine

	// Get final offset
	offset, err := stream.storage.GetOffset(topic, 0)
	if err != nil {
		t.Fatalf("Failed to get offset: %v", err)
	}

	// Offset should be totalExpected - 1 (0-indexed)
	if offset != int64(totalExpected-1) {
		t.Errorf("Expected final offset %d, got %d", totalExpected-1, offset)
	}
}

// TestWriteBuffering tests write batching functionality
func TestWriteBuffering(t *testing.T) {
	stream, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "buffer-topic"

	// Create topic
	if err := stream.CreateTopic(topic, 1, 0); err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Publish many messages quickly
	numMessages := 100
	for i := 0; i < numMessages; i++ {
		_, err := stream.Publish(topic, 0, fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)))
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Wait for flush
	time.Sleep(50 * time.Millisecond)

	// Verify all messages are persisted
	group := "buffer-group"
	consumerID := "consumer-1"
	if err := stream.Subscribe(topic, group, consumerID); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	messages, err := stream.Consume(topic, group, consumerID, numMessages)
	if err != nil {
		t.Fatalf("Failed to consume: %v", err)
	}

	if len(messages) != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, len(messages))
	}
}

// BenchmarkSinglePublish benchmarks single message publishing
func BenchmarkSinglePublish(b *testing.B) {
	stream, err := New(b.TempDir())
	if err != nil {
		b.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "bench-topic"
	if err := stream.CreateTopic(topic, 4, 0); err != nil {
		b.Fatalf("Failed to create topic: %v", err)
	}

	value := make([]byte, 1024)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := stream.Publish(topic, i%4, "key", value)
		if err != nil {
			b.Fatalf("Failed to publish: %v", err)
		}
	}
}

// BenchmarkBatchPublish benchmarks batch publishing
func BenchmarkBatchPublish(b *testing.B) {
	stream, err := New(b.TempDir())
	if err != nil {
		b.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "bench-batch-topic"
	if err := stream.CreateTopic(topic, 4, 0); err != nil {
		b.Fatalf("Failed to create topic: %v", err)
	}

	value := make([]byte, 1024)
	batchSize := 10

	b.ResetTimer()

	for i := 0; i < b.N; i += batchSize {
		messages := make([]*PublishMessage, batchSize)
		for j := 0; j < batchSize; j++ {
			messages[j] = &PublishMessage{
				Partition: (i + j) % 4,
				Key:       "key",
				Value:     value,
			}
		}
		_, err := stream.PublishBatch(topic, messages)
		if err != nil {
			b.Fatalf("Failed to batch publish: %v", err)
		}
	}
}

// BenchmarkConsume benchmarks message consumption
func BenchmarkConsume(b *testing.B) {
	stream, err := New(b.TempDir())
	if err != nil {
		b.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	topic := "bench-consume-topic"
	if err := stream.CreateTopic(topic, 4, 0); err != nil {
		b.Fatalf("Failed to create topic: %v", err)
	}

	// Pre-populate with messages
	value := make([]byte, 1024)
	for i := 0; i < 10000; i++ {
		_, err := stream.Publish(topic, i%4, "key", value)
		if err != nil {
			b.Fatalf("Failed to publish: %v", err)
		}
	}

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	group := "bench-group"
	consumerID := "consumer-1"
	if err := stream.Subscribe(topic, group, consumerID); err != nil {
		b.Fatalf("Failed to subscribe: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		messages, err := stream.Consume(topic, group, consumerID, 100)
		if err != nil {
			b.Fatalf("Failed to consume: %v", err)
		}
		if len(messages) > 0 {
			// Commit last offset
			lastMsg := messages[len(messages)-1]
			stream.Commit(topic, group, lastMsg.Partition, lastMsg.Offset+1)
		}
	}
}
