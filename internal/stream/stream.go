package stream

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

// Stream manages the stream processing system
type Stream struct {
	storage *StreamStorage

	// In-memory cache of topic metadata
	topics   map[string]*TopicMetadata
	topicsMu sync.RWMutex

	// Consumer group state
	groups   map[string]*ConsumerGroup
	groupsMu sync.RWMutex

	// Offset cache for faster reads (key: "group:topic:partition")
	offsetCache   map[string]int64
	offsetCacheMu sync.RWMutex

	// Background tasks
	stopChan  chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// ConsumerGroup manages consumers and partition assignments
type ConsumerGroup struct {
	Name      string
	Topic     string
	Consumers map[string]*Consumer
	mu        sync.RWMutex
}

// Consumer represents a member of a consumer group
type Consumer struct {
	ID         string
	LastSeen   time.Time
	Partitions []int
}

// New creates a new Stream instance
// path should be a separate directory from KV/Queue storage
func New(path string) (*Stream, error) {
	store, err := NewStorage(path)
	if err != nil {
		return nil, err
	}

	s := &Stream{
		storage:     store,
		topics:      make(map[string]*TopicMetadata),
		groups:      make(map[string]*ConsumerGroup),
		offsetCache: make(map[string]int64),
		stopChan:    make(chan struct{}),
	}

	// Load existing topics (TODO: Implement listing/loading from storage if needed)
	// For now, we rely on lazy loading or explicit creation

	// Start background tasks
	s.wg.Add(2)
	go s.retentionLoop()
	go s.offsetFlushLoop()

	return s, nil
}

// CreateTopic creates a new topic
func (s *Stream) CreateTopic(name string, partitions int, retentionMs int64) error {
	if partitions <= 0 {
		partitions = 4 // Default
	}
	if retentionMs <= 0 {
		retentionMs = 7 * 24 * 60 * 60 * 1000 // 7 days default
	}

	meta := &TopicMetadata{
		Name:        name,
		Partitions:  partitions,
		RetentionMs: retentionMs,
		CreatedAt:   time.Now().UnixMilli(),
	}

	if err := s.storage.CreateTopic(meta); err != nil {
		return err
	}

	s.topicsMu.Lock()
	s.topics[name] = meta
	s.topicsMu.Unlock()

	return nil
}

// GetTopicMetadata returns metadata for a topic
func (s *Stream) GetTopicMetadata(name string) (*TopicMetadata, error) {
	s.topicsMu.RLock()
	meta, ok := s.topics[name]
	s.topicsMu.RUnlock()

	if ok {
		return meta, nil
	}

	// Try loading from storage
	meta, err := s.storage.GetTopicMetadata(name)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		// Auto-create topic if it doesn't exist (optional feature, useful for dev)
		// For strictness, we might return error. Let's auto-create with defaults for now.
		return nil, fmt.Errorf("topic not found: %s", name)
	}

	s.topicsMu.Lock()
	s.topics[name] = meta
	s.topicsMu.Unlock()

	return meta, nil
}

// Publish appends a message to a topic
// If partition is -1, it is selected based on key hash or round-robin
func (s *Stream) Publish(topic string, partition int, key string, value []byte) (int64, error) {
	// Ensure topic exists or get metadata
	meta, err := s.GetTopicMetadata(topic)
	if err != nil {
		// Auto-create topic on publish?
		// Let's create with defaults
		err = s.CreateTopic(topic, 4, 0)
		if err != nil {
			return 0, err
		}
		meta, _ = s.GetTopicMetadata(topic)
	}

	// Select partition
	if partition < 0 || partition >= meta.Partitions {
		if key != "" {
			// Hash partition
			h := fnv.New32a()
			h.Write([]byte(key))
			partition = int(h.Sum32()) % meta.Partitions
		} else {
			// Round-robin (simplified: random or time-based for now)
			partition = int(time.Now().UnixNano()) % meta.Partitions
		}
	}

	return s.storage.AppendMessage(topic, partition, key, value)
}

// PublishMessage represents a message to be published
type PublishMessage struct {
	Partition int // -1 for auto-select
	Key       string
	Value     []byte
}

// PublishBatch publishes multiple messages atomically to a topic
// This is significantly faster than calling Publish multiple times
func (s *Stream) PublishBatch(topic string, messages []*PublishMessage) ([]int64, error) {
	// Ensure topic exists or get metadata
	meta, err := s.GetTopicMetadata(topic)
	if err != nil {
		// Auto-create topic on publish
		err = s.CreateTopic(topic, 4, 0)
		if err != nil {
			return nil, err
		}
		meta, _ = s.GetTopicMetadata(topic)
	}

	// Assign partitions for messages that need auto-selection
	for _, msg := range messages {
		if msg.Partition < 0 || msg.Partition >= meta.Partitions {
			if msg.Key != "" {
				// Hash partition
				h := fnv.New32a()
				h.Write([]byte(msg.Key))
				msg.Partition = int(h.Sum32()) % meta.Partitions
			} else {
				// Round-robin (simplified: time-based)
				msg.Partition = int(time.Now().UnixNano()) % meta.Partitions
			}
		}
	}

	// Batch append to storage
	return s.storage.AppendMessageBatch(topic, messages)
}

// Subscribe registers a consumer in a group
func (s *Stream) Subscribe(topic, group, consumerID string) error {
	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()

	g, exists := s.groups[group]
	if !exists {
		g = &ConsumerGroup{
			Name:      group,
			Topic:     topic,
			Consumers: make(map[string]*Consumer),
		}
		s.groups[group] = g
	}

	if g.Topic != topic {
		return fmt.Errorf("group %s already subscribed to %s", group, g.Topic)
	}

	// Register/Update consumer
	g.mu.Lock()
	c, exists := g.Consumers[consumerID]
	if !exists {
		c = &Consumer{
			ID: consumerID,
		}
		g.Consumers[consumerID] = c
	}
	c.LastSeen = time.Now()
	g.mu.Unlock()

	// Rebalance partitions
	return s.rebalanceGroup(g)
}

// Unsubscribe removes a consumer from a group
func (s *Stream) Unsubscribe(topic, group, consumerID string) error {
	s.groupsMu.Lock()
	defer s.groupsMu.Unlock()

	g, exists := s.groups[group]
	if !exists {
		return nil
	}

	g.mu.Lock()
	delete(g.Consumers, consumerID)
	empty := len(g.Consumers) == 0
	g.mu.Unlock()

	if empty {
		delete(s.groups, group)
		return nil
	}

	return s.rebalanceGroup(g)
}

// rebalanceGroup assigns partitions to consumers
// Simple strategy: Range assignment
func (s *Stream) rebalanceGroup(g *ConsumerGroup) error {
	meta, err := s.GetTopicMetadata(g.Topic)
	if err != nil {
		return err
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	consumers := make([]string, 0, len(g.Consumers))
	for id := range g.Consumers {
		consumers = append(consumers, id)
	}

	if len(consumers) == 0 {
		return nil
	}

	// Sort consumers for deterministic assignment?
	// Map iteration is random, so assignments might jump around.
	// For MVP, we just iterate.

	partitionsPerConsumer := meta.Partitions / len(consumers)
	extraPartitions := meta.Partitions % len(consumers)

	currentPart := 0
	for _, id := range consumers {
		c := g.Consumers[id]
		numParts := partitionsPerConsumer
		if extraPartitions > 0 {
			numParts++
			extraPartitions--
		}

		c.Partitions = make([]int, 0, numParts)
		for i := 0; i < numParts; i++ {
			c.Partitions = append(c.Partitions, currentPart)
			currentPart++
		}
	}

	return nil
}

// Consume fetches messages for a consumer in a group using parallel partition fetching
func (s *Stream) Consume(topic, group, consumerID string, count int) ([]*Message, error) {
	// Ensure subscription and get assigned partitions
	s.groupsMu.RLock()
	g, exists := s.groups[group]
	s.groupsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("consumer group not found")
	}

	g.mu.RLock()
	c, exists := g.Consumers[consumerID]
	var partitions []int
	if exists {
		partitions = make([]int, len(c.Partitions))
		copy(partitions, c.Partitions)
		c.LastSeen = time.Now() // Heartbeat
	}
	g.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("consumer not registered")
	}

	if len(partitions) == 0 {
		return []*Message{}, nil
	}

	// Parallel fetch from all partitions
	type partitionResult struct {
		messages []*Message
		err      error
	}

	resultChan := make(chan partitionResult, len(partitions))
	perPart := count / len(partitions)
	if perPart == 0 {
		perPart = 1
	}

	// Launch goroutine for each partition
	for _, p := range partitions {
		go func(partition int) {
			// Get current offset from cache or storage
			offset, err := s.getCachedOffset(group, topic, partition)
			if err != nil {
				resultChan <- partitionResult{err: err}
				return
			}

			// Fetch messages
			msgs, err := s.storage.FetchMessages(topic, partition, offset, perPart)
			resultChan <- partitionResult{messages: msgs, err: err}
		}(p)
	}

	// Collect results
	result := make([]*Message, 0, count)
	for i := 0; i < len(partitions); i++ {
		res := <-resultChan
		if res.err != nil {
			return nil, res.err
		}
		result = append(result, res.messages...)
		if len(result) >= count {
			break
		}
	}

	return result, nil
}

// Commit commits an offset for a consumer group and updates cache
func (s *Stream) Commit(topic, group string, partition int, offset int64) error {
	// Update cache
	cacheKey := fmt.Sprintf("%s:%s:%d", group, topic, partition)
	s.offsetCacheMu.Lock()
	s.offsetCache[cacheKey] = offset
	s.offsetCacheMu.Unlock()

	// Persist to storage
	return s.storage.CommitOffset(group, topic, partition, offset)
}

// Close closes the stream system
func (s *Stream) Close() error {
	s.closeOnce.Do(func() {
		close(s.stopChan)
	})
	s.wg.Wait()
	return s.storage.Close()
}

// retentionLoop periodically cleans up old messages
func (s *Stream) retentionLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.enforceRetention()
		}
	}
}

func (s *Stream) enforceRetention() {
	s.topicsMu.RLock()
	topics := make([]*TopicMetadata, 0, len(s.topics))
	for _, t := range s.topics {
		topics = append(topics, t)
	}
	s.topicsMu.RUnlock()

	for _, t := range topics {
		if t.RetentionMs > 0 {
			for p := 0; p < t.Partitions; p++ {
				s.storage.DeleteOldMessages(t.Name, p, t.RetentionMs)
			}
		}
	}
}
