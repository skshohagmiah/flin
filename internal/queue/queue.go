package queue

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Message represents a queue message
type Message struct {
	ID        string                 `json:"id"`
	QueueName string                 `json:"queue_name"`
	Body      []byte                 `json:"body"`
	Headers   map[string]string      `json:"headers"`
	Priority  int                    `json:"priority"` // 0-9, higher = more priority
	Timestamp time.Time              `json:"timestamp"`
	Attempts  int                    `json:"attempts"`
	MaxRetries int                   `json:"max_retries"`
	TTL       time.Duration          `json:"ttl"` // Time to live
	DeadLetter string                `json:"dead_letter"` // Dead letter queue name
}

// Queue represents a message queue (like RabbitMQ queue)
type Queue struct {
	name       string
	messages   chan *Message
	consumers  map[string]*Consumer
	mu         sync.RWMutex
	maxSize    int
	durable    bool // Persist to disk
	autoDelete bool // Delete when no consumers
	exclusive  bool // Only one consumer allowed
	closed     bool
}

// Consumer represents a queue consumer
type Consumer struct {
	ID       string
	Tag      string
	Queue    *Queue
	Messages chan *Message
	Ack      chan string // For message acknowledgment
	Cancel   chan struct{}
}

// QueueManager manages all queues (like RabbitMQ broker)
type QueueManager struct {
	queues    map[string]*Queue
	exchanges map[string]*Exchange
	mu        sync.RWMutex
	storage   QueueStorage
}

// Exchange represents a message exchange (like RabbitMQ exchange)
type Exchange struct {
	Name       string
	Type       string // direct, fanout, topic, headers
	Durable    bool
	AutoDelete bool
	Bindings   map[string][]*Binding // routing_key -> queues
	mu         sync.RWMutex
}

// Binding represents queue-exchange binding
type Binding struct {
	QueueName  string
	RoutingKey string
	Arguments  map[string]interface{}
}

// QueueStorage interface for persistence
type QueueStorage interface {
	SaveMessage(msg *Message) error
	LoadMessages(queueName string) ([]*Message, error)
	DeleteMessage(queueName, msgID string) error
}

// NewQueueManager creates a new queue manager
func NewQueueManager(storage QueueStorage) *QueueManager {
	return &QueueManager{
		queues:    make(map[string]*Queue),
		exchanges: make(map[string]*Exchange),
		storage:   storage,
	}
}

// GetQueue returns a queue by name
func (qm *QueueManager) GetQueue(name string) *Queue {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return qm.queues[name]
}

// DeclareQueue creates or gets a queue
func (qm *QueueManager) DeclareQueue(name string, durable, autoDelete, exclusive bool, maxSize int) (*Queue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if q, exists := qm.queues[name]; exists {
		return q, nil
	}

	bufferSize := maxSize
	if bufferSize == 0 {
		bufferSize = 10000 // Default buffer
	}

	queue := &Queue{
		name:       name,
		messages:   make(chan *Message, bufferSize),
		consumers:  make(map[string]*Consumer),
		maxSize:    maxSize,
		durable:    durable,
		autoDelete: autoDelete,
		exclusive:  exclusive,
		closed:     false,
	}

	qm.queues[name] = queue
	return queue, nil
}

// DeleteQueue removes a queue
func (qm *QueueManager) DeleteQueue(name string, ifUnused, ifEmpty bool) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queue, exists := qm.queues[name]
	if !exists {
		return fmt.Errorf("queue not found: %s", name)
	}

	queue.mu.RLock()
	hasConsumers := len(queue.consumers) > 0
	hasMessages := len(queue.messages) > 0
	queue.mu.RUnlock()

	if ifUnused && hasConsumers {
		return fmt.Errorf("queue has consumers")
	}

	if ifEmpty && hasMessages {
		return fmt.Errorf("queue not empty")
	}

	delete(qm.queues, name)
	return nil
}

// Publish publishes a message to a queue
func (qm *QueueManager) Publish(queueName string, body []byte, headers map[string]string, priority int, ttl time.Duration) error {
	qm.mu.RLock()
	queue, exists := qm.queues[queueName]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("queue not found: %s", queueName)
	}

	msg := &Message{
		ID:        generateID(),
		QueueName: queueName,
		Body:      body,
		Headers:   headers,
		Priority:  priority,
		Timestamp: time.Now(),
		Attempts:  0,
		TTL:       ttl,
	}

	return queue.Enqueue(msg)
}

// PublishToExchange publishes to an exchange with routing
func (qm *QueueManager) PublishToExchange(exchangeName, routingKey string, body []byte, headers map[string]string) error {
	qm.mu.RLock()
	exchange, exists := qm.exchanges[exchangeName]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("exchange not found: %s", exchangeName)
	}

	return exchange.Route(qm, routingKey, body, headers)
}

// Consume starts consuming messages from a queue
func (qm *QueueManager) Consume(queueName, consumerTag string, autoAck bool) (*Consumer, error) {
	qm.mu.RLock()
	queue, exists := qm.queues[queueName]
	qm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("queue not found: %s", queueName)
	}

	return queue.AddConsumer(consumerTag, autoAck)
}

// DeclareExchange creates an exchange
func (qm *QueueManager) DeclareExchange(name, exchangeType string, durable, autoDelete bool) (*Exchange, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if ex, exists := qm.exchanges[name]; exists {
		return ex, nil
	}

	exchange := &Exchange{
		Name:       name,
		Type:       exchangeType,
		Durable:    durable,
		AutoDelete: autoDelete,
		Bindings:   make(map[string][]*Binding),
	}

	qm.exchanges[name] = exchange
	return exchange, nil
}

// BindQueue binds a queue to an exchange
func (qm *QueueManager) BindQueue(queueName, exchangeName, routingKey string, args map[string]interface{}) error {
	qm.mu.RLock()
	exchange, exists := qm.exchanges[exchangeName]
	qm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("exchange not found: %s", exchangeName)
	}

	binding := &Binding{
		QueueName:  queueName,
		RoutingKey: routingKey,
		Arguments:  args,
	}

	exchange.mu.Lock()
	exchange.Bindings[routingKey] = append(exchange.Bindings[routingKey], binding)
	exchange.mu.Unlock()

	return nil
}

// Queue methods

// Enqueue adds a message to the queue
func (q *Queue) Enqueue(msg *Message) error {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return fmt.Errorf("queue closed")
	}
	q.mu.RUnlock()

	// Fast non-blocking send
	select {
	case q.messages <- msg:
		return nil
	default:
		return fmt.Errorf("queue full")
	}
}

// Dequeue removes and returns the next message
func (q *Queue) Dequeue() (*Message, error) {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return nil, fmt.Errorf("queue closed")
	}
	q.mu.RUnlock()

	// Non-blocking receive
	select {
	case msg := <-q.messages:
		return msg, nil
	default:
		return nil, fmt.Errorf("queue empty")
	}
}

// AddConsumer adds a consumer to the queue
func (q *Queue) AddConsumer(tag string, autoAck bool) (*Consumer, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.exclusive && len(q.consumers) > 0 {
		return nil, fmt.Errorf("queue is exclusive")
	}

	consumer := &Consumer{
		ID:       generateID(),
		Tag:      tag,
		Queue:    q,
		Messages: make(chan *Message, 100),
		Ack:      make(chan string, 100),
		Cancel:   make(chan struct{}),
	}

	q.consumers[consumer.ID] = consumer

	// Start dispatcher if this is the first consumer
	if len(q.consumers) == 1 {
		q.startConsumerDispatcher()
	}

	// Start delivery goroutine
	go consumer.deliverMessages(autoAck)

	return consumer, nil
}

// RemoveConsumer removes a consumer
func (q *Queue) RemoveConsumer(consumerID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if consumer, exists := q.consumers[consumerID]; exists {
		close(consumer.Cancel)
		delete(q.consumers, consumerID)
	}

	// Auto-delete queue if configured
	if q.autoDelete && len(q.consumers) == 0 {
		// Queue will be deleted by manager
	}
}

// startConsumerDispatcher dispatches messages from queue to consumers
func (q *Queue) startConsumerDispatcher() {
	go func() {
		for {
			q.mu.RLock()
			if q.closed {
				q.mu.RUnlock()
				return
			}
			hasConsumers := len(q.consumers) > 0
			q.mu.RUnlock()

			if !hasConsumers {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Get message from queue
			select {
			case msg := <-q.messages:
				// Send to first available consumer (blocking)
				q.mu.RLock()
				consumers := make([]*Consumer, 0, len(q.consumers))
				for _, c := range q.consumers {
					consumers = append(consumers, c)
				}
				q.mu.RUnlock()

				// Try to send to any consumer (blocking until sent)
				sent := false
				for !sent {
					for _, consumer := range consumers {
						select {
						case consumer.Messages <- msg:
							sent = true
						default:
							continue
						}
						if sent {
							break
						}
					}
					// If no consumer ready, wait a bit
					if !sent {
						time.Sleep(100 * time.Microsecond)
					}
				}
			default:
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()
}

// Consumer methods

// deliverMessages delivers messages to consumer
func (c *Consumer) deliverMessages(autoAck bool) {
	for {
		select {
		case <-c.Cancel:
			return
		case msg := <-c.Messages:
			// Check TTL
			if msg.TTL > 0 && time.Since(msg.Timestamp) > msg.TTL {
				// Message expired, send to dead letter queue if configured
				if msg.DeadLetter != "" {
					// TODO: Send to dead letter queue
				}
				continue
			}

			// Auto-acknowledge if configured
			if autoAck {
				// Message is automatically acknowledged
			}
			// Otherwise, wait for manual ack via Ack channel
		}
	}
}

// Ack acknowledges a message
func (c *Consumer) AckMessage(msgID string) {
	select {
	case c.Ack <- msgID:
	default:
	}
}

// Cancel cancels the consumer
func (c *Consumer) CancelConsumer() {
	c.Queue.RemoveConsumer(c.ID)
}

// Exchange methods

// Route routes a message based on exchange type
func (e *Exchange) Route(qm *QueueManager, routingKey string, body []byte, headers map[string]string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	switch e.Type {
	case "direct":
		return e.routeDirect(qm, routingKey, body, headers)
	case "fanout":
		return e.routeFanout(qm, body, headers)
	case "topic":
		return e.routeTopic(qm, routingKey, body, headers)
	case "headers":
		return e.routeHeaders(qm, headers, body)
	default:
		return fmt.Errorf("unknown exchange type: %s", e.Type)
	}
}

func (e *Exchange) routeDirect(qm *QueueManager, routingKey string, body []byte, headers map[string]string) error {
	bindings, exists := e.Bindings[routingKey]
	if !exists {
		return nil // No bindings, message is dropped
	}

	for _, binding := range bindings {
		if err := qm.Publish(binding.QueueName, body, headers, 0, 0); err != nil {
			return err
		}
	}

	return nil
}

func (e *Exchange) routeFanout(qm *QueueManager, body []byte, headers map[string]string) error {
	// Send to all bound queues
	for _, bindings := range e.Bindings {
		for _, binding := range bindings {
			if err := qm.Publish(binding.QueueName, body, headers, 0, 0); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Exchange) routeTopic(qm *QueueManager, routingKey string, body []byte, headers map[string]string) error {
	// Topic matching with wildcards (* and #)
	// TODO: Implement pattern matching
	return e.routeDirect(qm, routingKey, body, headers)
}

func (e *Exchange) routeHeaders(qm *QueueManager, headers map[string]string, body []byte) error {
	// Header-based routing
	// TODO: Implement header matching
	return nil
}

// Utility functions

var idCounter uint64

func generateID() string {
	idCounter++
	return fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), idCounter)
}

// Stats returns queue statistics
func (q *Queue) Stats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return map[string]interface{}{
		"name":           q.name,
		"messages":       len(q.messages), // Channel length
		"consumers":      len(q.consumers),
		"max_size":       q.maxSize,
		"durable":        q.durable,
		"auto_delete":    q.autoDelete,
		"exclusive":      q.exclusive,
	}
}

// Serialize message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// Deserialize message from JSON
func MessageFromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
