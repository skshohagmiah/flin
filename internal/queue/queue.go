package queue

// Queue wraps the queue storage backend
type Queue struct {
	storage *QueueStorage
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

// Push adds an item to the end of the queue
func (q *Queue) Push(queueName string, value []byte) error {
	return q.storage.Push(queueName, value)
}

// Pop removes and returns the first item from the queue
func (q *Queue) Pop(queueName string) ([]byte, error) {
	return q.storage.Pop(queueName)
}

// Peek returns the first item without removing it
func (q *Queue) Peek(queueName string) ([]byte, error) {
	return q.storage.Peek(queueName)
}

// Len returns the number of items in the queue
func (q *Queue) Len(queueName string) (uint64, error) {
	return q.storage.Len(queueName)
}

// Clear removes all items from the queue
func (q *Queue) Clear(queueName string) error {
	return q.storage.Clear(queueName)
}

// Close closes the underlying storage
func (q *Queue) Close() error {
	return q.storage.Close()
}
