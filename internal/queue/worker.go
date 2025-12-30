package queue

import (
	"log"
	"sync"
	"time"
)

// VisibilityTimeoutWorker requeues messages whose visibility timeout has expired
type VisibilityTimeoutWorker struct {
	storage       *QueueStorage
	queueNames    []string
	checkInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

// NewVisibilityTimeoutWorker creates a new worker
func NewVisibilityTimeoutWorker(storage *QueueStorage, checkInterval time.Duration) *VisibilityTimeoutWorker {
	if checkInterval == 0 {
		checkInterval = 10 * time.Second // Default check every 10 seconds
	}

	return &VisibilityTimeoutWorker{
		storage:       storage,
		queueNames:    make([]string, 0),
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// AddQueue adds a queue to monitor for expired messages
func (w *VisibilityTimeoutWorker) AddQueue(queueName string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if already exists
	for _, name := range w.queueNames {
		if name == queueName {
			return
		}
	}

	w.queueNames = append(w.queueNames, queueName)
}

// RemoveQueue removes a queue from monitoring
func (w *VisibilityTimeoutWorker) RemoveQueue(queueName string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, name := range w.queueNames {
		if name == queueName {
			w.queueNames = append(w.queueNames[:i], w.queueNames[i+1:]...)
			return
		}
	}
}

// Start begins the background worker
func (w *VisibilityTimeoutWorker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop stops the background worker
func (w *VisibilityTimeoutWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// run is the main worker loop
func (w *VisibilityTimeoutWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.checkExpiredMessages()
		}
	}
}

// checkExpiredMessages checks all monitored queues for expired messages
func (w *VisibilityTimeoutWorker) checkExpiredMessages() {
	w.mu.RLock()
	queues := make([]string, len(w.queueNames))
	copy(queues, w.queueNames)
	w.mu.RUnlock()

	for _, queueName := range queues {
		count, err := w.storage.RequeueExpiredMessages(queueName)
		if err != nil {
			log.Printf("[VisibilityWorker] Error requeuing expired messages for queue %s: %v", queueName, err)
			continue
		}

		if count > 0 {
			log.Printf("[VisibilityWorker] Requeued %d expired messages in queue %s", count, queueName)
		}
	}
}
