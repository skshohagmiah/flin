package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

// ActivityService handles activity stream operations
type ActivityService struct {
	client *flin.Client
	topic  string
}

// NewActivityService creates a new activity service
func NewActivityService(client *flin.Client) *ActivityService {
	return &ActivityService{
		client: client,
		topic:  "activity_stream",
	}
}

// PublishActivity publishes an activity event to the stream
func (s *ActivityService) PublishActivity(eventType string, data map[string]interface{}) error {
	event := map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to stream (partition -1 for automatic partitioning)
	err = s.client.Publish(s.topic, -1, eventType, eventData)
	if err != nil {
		return fmt.Errorf("failed to publish activity: %w", err)
	}

	return nil
}

// ConsumeActivities consumes activity events from the stream
func (s *ActivityService) ConsumeActivities(group, consumer string, count int) error {
	messages, err := s.client.Stream.Consume(s.topic, group, consumer, count)
	if err != nil {
		return fmt.Errorf("failed to consume activities: %w", err)
	}

	for _, msg := range messages {
		var event map[string]interface{}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}

		log.Printf("[Activity] Type: %s, Data: %v", event["type"], event["data"])

		// Commit offset after processing
		s.client.Stream.Commit(s.topic, group, msg.Partition, msg.Offset+1)
	}

	return nil
}
