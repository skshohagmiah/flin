package queue

import "time"

// Queue wraps the queue storage backend
type Queue struct {
	storage *QueueStorage
}

// MessageHandle represents a message that can be acknowledged or rejected
type MessageHandle struct {
	QueueName     string
	MessageID     string
	Value         []byte
	DeliveryCount int
	queue         *Queue
}

// Ack acknowledges successful processing and deletes the message
func (m *MessageHandle) Ack() error {
	return m.queue.storage.AckMessage(m.QueueName, m.MessageID)
}

// Nack rejects the message and requeues it (or sends to DLQ if max retries exceeded)
func (m *MessageHandle) Nack(requeue bool) error {
	return m.queue.storage.NackMessage(m.QueueName, m.MessageID, requeue)
}

// New creates a new Queue instance with BadgerDB storage
func New(path string) (*Queue, error) {
	store, err := NewStorage(path)
	if err != nil {
		return nil, err
	}

	return &Queue{
		storage: store,
	}, nil
}

// Push adds an item to the end of the queue (legacy method, kept for backward compatibility)
func (q *Queue) Push(queueName string, value []byte) error {
	_, err := q.storage.PushMessage(queueName, value)
	return err
}

// PushMessage adds a message to the queue and returns the message ID
func (q *Queue) PushMessage(queueName string, value []byte) (string, error) {
	return q.storage.PushMessage(queueName, value)
}

// Pop removes and returns the first item from the queue (DEPRECATED: use PopMessage instead)
// This is kept for backward compatibility but doesn't support acknowledgment
func (q *Queue) Pop(queueName string) ([]byte, error) {
	return q.storage.Pop(queueName)
}

// PopMessage retrieves a message with acknowledgment support
// The message must be explicitly acknowledged with Ack() or rejected with Nack()
func (q *Queue) PopMessage(queueName, consumerID string, visibilityTimeout time.Duration) (*MessageHandle, error) {
	msg, err := q.storage.PopMessage(queueName, consumerID, visibilityTimeout)
	if err != nil {
		return nil, err
	}

	return &MessageHandle{
		QueueName:     queueName,
		MessageID:     msg.ID,
		Value:         msg.Value,
		DeliveryCount: msg.DeliveryCount,
		queue:         q,
	}, nil
}

// SetDeadLetterQueue configures a dead-letter queue for a queue
func (q *Queue) SetDeadLetterQueue(queueName, dlqName string, maxRetries int) error {
	cfg, err := q.storage.GetQueueConfig(queueName)
	if err != nil {
		return err
	}

	cfg.DeadLetterQueue = dlqName
	cfg.MaxRetries = maxRetries

	return q.storage.SetQueueConfig(cfg)
}

// SetVisibilityTimeout sets the default visibility timeout for a queue
func (q *Queue) SetVisibilityTimeout(queueName string, timeout time.Duration) error {
	cfg, err := q.storage.GetQueueConfig(queueName)
	if err != nil {
		return err
	}

	cfg.DefaultVisibilityTimeout = timeout.Milliseconds()

	return q.storage.SetQueueConfig(cfg)
}

// RequeueExpiredMessages requeues messages whose visibility timeout has expired
func (q *Queue) RequeueExpiredMessages(queueName string) (int, error) {
	return q.storage.RequeueExpiredMessages(queueName)
}

// Peek returns the first item without removing it (legacy method)
func (q *Queue) Peek(queueName string) ([]byte, error) {
	return q.storage.Peek(queueName)
}

// Len returns the number of items in the queue (legacy method)
func (q *Queue) Len(queueName string) (uint64, error) {
	return q.storage.Len(queueName)
}

// Clear removes all items from the queue (legacy method)
func (q *Queue) Clear(queueName string) error {
	return q.storage.Clear(queueName)
}

// Close closes the underlying storage
func (q *Queue) Close() error {
	return q.storage.Close()
}
