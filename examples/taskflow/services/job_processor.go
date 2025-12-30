package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	flin "github.com/skshohagmiah/flin/clients/go"
)

// JobProcessor processes background jobs from the queue
type JobProcessor struct {
	client    *flin.Client
	queueName string
	running   bool
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(client *flin.Client) *JobProcessor {
	return &JobProcessor{
		client:    client,
		queueName: "background_jobs",
	}
}

// Start starts the job processor
func (p *JobProcessor) Start() {
	p.running = true
	log.Println("[JobProcessor] Started")

	go func() {
		for p.running {
			// Pop job from queue
			jobData, err := p.client.Queue.Pop(p.queueName)
			if err != nil {
				// Queue might be empty, wait a bit
				time.Sleep(1 * time.Second)
				continue
			}

			// Process job
			var job map[string]interface{}
			if err := json.Unmarshal(jobData, &job); err != nil {
				log.Printf("[JobProcessor] Failed to unmarshal job: %v", err)
				continue
			}

			p.processJob(job)
		}
	}()
}

// Stop stops the job processor
func (p *JobProcessor) Stop() {
	p.running = false
	log.Println("[JobProcessor] Stopped")
}

// processJob processes a single job
func (p *JobProcessor) processJob(job map[string]interface{}) {
	jobType, ok := job["type"].(string)
	if !ok {
		log.Printf("[JobProcessor] Invalid job type")
		return
	}

	log.Printf("[JobProcessor] Processing job: %s", jobType)

	switch jobType {
	case "send_notification":
		p.handleNotification(job)
	default:
		log.Printf("[JobProcessor] Unknown job type: %s", jobType)
	}
}

// handleNotification handles notification jobs
func (p *JobProcessor) handleNotification(job map[string]interface{}) {
	assigneeID, _ := job["assignee_id"].(string)
	message, _ := job["message"].(string)

	// Simulate sending notification (in real app, this would send email/SMS/push)
	fmt.Printf("📧 [Notification] To: %s, Message: %s\n", assigneeID, message)

	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	log.Printf("[JobProcessor] Notification sent to %s", assigneeID)
}
