package queue

import (
	"fmt"
	"time"

	"github.com/skshohagmiah/flin/internal/queue"
	"github.com/skshohagmiah/flin/internal/storage"
)

// Client provides a simple API for queue operations
type Client struct {
	manager *queue.QueueManager
}

// Message represents a queue message with acknowledgment
type Message struct {
	ID        string
	Body      []byte
	Headers   map[string]string
	Priority  int
	Timestamp time.Time
	consumer  *queue.Consumer
	acked     bool
}

// NewClient creates a new queue client
func NewClient(storagePath string) (*Client, error) {
	var queueStorage *storage.QueueStorage
	var err error

	if storagePath != "" {
		queueStorage, err = storage.NewQueueStorage(storagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}
	}

	// Create adapter for queue storage interface
	var adapter queue.QueueStorage
	if queueStorage != nil {
		adapter = &queueStorageAdapter{storage: queueStorage}
	}

	manager := queue.NewQueueManager(adapter)

	return &Client{
		manager: manager,
	}, nil
}

// Close closes the queue client and releases resources
func (c *Client) Close() error {
	// Nothing to close for now
	return nil
}

// queueStorageAdapter adapts storage.QueueStorage to queue.QueueStorage interface
type queueStorageAdapter struct {
	storage *storage.QueueStorage
}

func (a *queueStorageAdapter) SaveMessage(msg *queue.Message) error {
	qMsg := &storage.QueueMessage{
		ID:        msg.ID,
		QueueName: msg.QueueName,
		Body:      msg.Body,
		Headers:   msg.Headers,
		Priority:  msg.Priority,
		Timestamp: msg.Timestamp.Unix(),
		TTL:       int64(msg.TTL),
	}
	return a.storage.SaveMessage(qMsg)
}

func (a *queueStorageAdapter) LoadMessages(queueName string) ([]*queue.Message, error) {
	msgs, err := a.storage.LoadMessages(queueName)
	if err != nil {
		return nil, err
	}

	result := make([]*queue.Message, len(msgs))
	for i, m := range msgs {
		result[i] = &queue.Message{
			ID:        m.ID,
			QueueName: m.QueueName,
			Body:      m.Body,
			Headers:   m.Headers,
			Priority:  m.Priority,
			Timestamp: time.Unix(m.Timestamp, 0),
			TTL:       time.Duration(m.TTL),
		}
	}
	return result, nil
}

func (a *queueStorageAdapter) DeleteMessage(queueName, msgID string) error {
	return a.storage.DeleteMessage(queueName, msgID)
}

// Enqueue adds a message to a queue
func (c *Client) Enqueue(queueName string, data []byte) error {
	return c.EnqueueWithOptions(queueName, data, nil, 0, 0)
}

// EnqueueWithOptions adds a message with priority and TTL
func (c *Client) EnqueueWithOptions(queueName string, data []byte, headers map[string]string, priority int, ttl time.Duration) error {
	// Auto-create queue if it doesn't exist
	_, err := c.manager.DeclareQueue(queueName, false, false, false, 10000)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	return c.manager.Publish(queueName, data, headers, priority, ttl)
}

// Dequeue retrieves and removes a message from a queue
func (c *Client) Dequeue(queueName string) (*Message, error) {
	return c.DequeueWithTimeout(queueName, 5*time.Second)
}

// DequeueWithTimeout retrieves a message with custom timeout
func (c *Client) DequeueWithTimeout(queueName string, timeout time.Duration) (*Message, error) {
	// Get the queue
	queueObj := c.getQueue(queueName)
	if queueObj == nil {
		return nil, fmt.Errorf("queue not found: %s", queueName)
	}

	// Try immediate dequeue first
	msg, err := queueObj.Dequeue()
	if err == nil {
		return &Message{
			ID:        msg.ID,
			Body:      msg.Body,
			Headers:   msg.Headers,
			Priority:  msg.Priority,
			Timestamp: msg.Timestamp,
			consumer:  nil,
			acked:     true, // Direct dequeue is auto-ack
		}, nil
	}

	// If empty and timeout > 0, wait with backoff
	if timeout > 0 {
		startTime := time.Now()
		backoff := 100 * time.Microsecond
		
		for time.Since(startTime) < timeout {
			time.Sleep(backoff)
			
			msg, err := queueObj.Dequeue()
			if err == nil {
				return &Message{
					ID:        msg.ID,
					Body:      msg.Body,
					Headers:   msg.Headers,
					Priority:  msg.Priority,
					Timestamp: msg.Timestamp,
					consumer:  nil,
					acked:     true,
				}, nil
			}
			
			// Exponential backoff up to 10ms
			if backoff < 10*time.Millisecond {
				backoff *= 2
			}
		}
	}

	return nil, fmt.Errorf("timeout: no messages available")
}

// getQueue gets the internal queue object
func (c *Client) getQueue(name string) *queue.Queue {
	// Access the manager's queues map
	// This is a hack but necessary for direct dequeue
	return c.manager.GetQueue(name)
}

// Consume creates a consumer that continuously receives messages
func (c *Client) Consume(queueName string, handler func(*Message)) error {
	return c.ConsumeWithOptions(queueName, handler, false)
}

// ConsumeWithOptions creates a consumer with auto-ack option
func (c *Client) ConsumeWithOptions(queueName string, handler func(*Message), autoAck bool) error {
	// Auto-create queue if it doesn't exist
	_, err := c.manager.DeclareQueue(queueName, false, false, false, 10000)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	consumer, err := c.manager.Consume(queueName, "consumer", autoAck)
	if err != nil {
		return fmt.Errorf("failed to consume: %w", err)
	}

	go func() {
		for msg := range consumer.Messages {
			wrappedMsg := &Message{
				ID:        msg.ID,
				Body:      msg.Body,
				Headers:   msg.Headers,
				Priority:  msg.Priority,
				Timestamp: msg.Timestamp,
				consumer:  consumer,
				acked:     autoAck,
			}
			handler(wrappedMsg)
		}
	}()

	return nil
}

// CreateQueue explicitly creates a queue with options
func (c *Client) CreateQueue(name string, durable bool, maxSize int) error {
	_, err := c.manager.DeclareQueue(name, durable, false, false, maxSize)
	return err
}

// DeleteQueue removes a queue
func (c *Client) DeleteQueue(name string) error {
	return c.manager.DeleteQueue(name, false, false)
}

// QueueStats returns queue statistics
func (c *Client) QueueStats(name string) (map[string]interface{}, error) {
	// This would need to be implemented in the manager
	// For now, return basic info
	return map[string]interface{}{
		"name": name,
	}, nil
}

// Message methods

// Ack acknowledges the message (confirms processing)
func (m *Message) Ack() error {
	if m.acked {
		return fmt.Errorf("message already acknowledged")
	}

	if m.consumer != nil {
		m.consumer.AckMessage(m.ID)
		m.acked = true
	}

	return nil
}

// Nack rejects the message (requeue or send to dead letter)
func (m *Message) Nack(requeue bool) error {
	if m.acked {
		return fmt.Errorf("message already acknowledged")
	}

	// If requeue is false, message goes to dead letter queue
	// Implementation would depend on queue configuration
	m.acked = true

	return nil
}

// IsAcked returns whether the message has been acknowledged
func (m *Message) IsAcked() bool {
	return m.acked
}
