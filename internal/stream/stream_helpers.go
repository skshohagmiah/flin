package stream

import (
	"fmt"
	"time"
)

// getCachedOffset retrieves offset from cache or storage
func (s *Stream) getCachedOffset(group, topic string, partition int) (int64, error) {
	cacheKey := fmt.Sprintf("%s:%s:%d", group, topic, partition)

	// Try cache first
	s.offsetCacheMu.RLock()
	offset, exists := s.offsetCache[cacheKey]
	s.offsetCacheMu.RUnlock()

	if exists {
		return offset, nil
	}

	// Cache miss - fetch from storage
	offset, err := s.storage.GetConsumerOffset(group, topic, partition)
	if err != nil {
		return 0, err
	}

	// Update cache
	s.offsetCacheMu.Lock()
	s.offsetCache[cacheKey] = offset
	s.offsetCacheMu.Unlock()

	return offset, nil
}

// offsetFlushLoop periodically flushes cached offsets to storage
func (s *Stream) offsetFlushLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			// Final flush before shutdown
			s.flushOffsets()
			return
		case <-ticker.C:
			s.flushOffsets()
		}
	}
}

func (s *Stream) flushOffsets() {
	// Offsets are already persisted in Commit()
	// This is a placeholder for future optimizations where we might
	// batch offset commits
}
