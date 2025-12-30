package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	flin "github.com/skshohagmiah/flin/clients/go"
)

// TaskService handles task operations
type TaskService struct {
	client   *flin.Client
	jobQueue string
}

// NewTaskService creates a new task service
func NewTaskService(client *flin.Client) *TaskService {
	return &TaskService{
		client:   client,
		jobQueue: "background_jobs",
	}
}

// Task represents a task in the system
type Task struct {
	ProjectID   string `json:"project_id"`
	TaskID      string `json:"task_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeID  string `json:"assignee_id"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(projectID, title, description, priority string) (*Task, error) {
	task := &Task{
		ProjectID:   projectID,
		TaskID:      uuid.New().String(),
		Title:       title,
		Description: description,
		Status:      "todo",
		Priority:    priority,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	doc := map[string]interface{}{
		"project_id":  task.ProjectID,
		"task_id":     task.TaskID,
		"title":       task.Title,
		"description": task.Description,
		"assignee_id": "",
		"status":      task.Status,
		"priority":    task.Priority,
		"created_at":  float64(task.CreatedAt),
		"updated_at":  float64(task.UpdatedAt),
	}

	_, err := s.client.DB.Insert("tasks", doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Activity stream disabled - focusing on DB features
	// s.activityService.PublishActivity("task.created", map[string]interface{}{
	// 	"task_id":    task.TaskID,
	// 	"project_id": task.ProjectID,
	// 	"title":      task.Title,
	// })

	return task, nil
}

// AssignTask assigns a task to a user
func (s *TaskService) AssignTask(projectID, taskID, assigneeID string) error {
	// Update task
	err := s.client.DB.Update("tasks").
		Where("project_id", flin.Eq, projectID).
		Where("task_id", flin.Eq, taskID).
		Set("assignee_id", assigneeID).
		Set("updated_at", float64(time.Now().Unix())).
		Exec()

	if err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	// Queue and Activity stream disabled - focusing on DB features
	// Enqueue notification job
	// job := map[string]interface{}{
	// 	"type":        "send_notification",
	// 	"task_id":     taskID,
	// 	"assignee_id": assigneeID,
	// 	"message":     fmt.Sprintf("You have been assigned to task %s", taskID),
	// }
	// jobData, _ := json.Marshal(job)
	// if err := s.client.Queue.Push(s.jobQueue, jobData); err != nil {
	// 	fmt.Printf("Warning: failed to enqueue notification job: %v\n", err)
	// }

	// Publish activity event
	// s.activityService.PublishActivity("task.assigned", map[string]interface{}{
	// 	"task_id":     taskID,
	// 	"assignee_id": assigneeID,
	// })

	fmt.Printf("✅ Task %s assigned to %s\n", taskID, assigneeID)
	return nil
}

// ListTasksByProject lists all tasks for a project
func (s *TaskService) ListTasksByProject(projectID string) ([]*Task, error) {
	results, err := s.client.DB.Query("tasks").
		Where("project_id", flin.Eq, projectID).
		Exec()

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	tasks := make([]*Task, 0, len(results))
	for _, doc := range results {
		task := &Task{
			ProjectID:   doc["project_id"].(string),
			TaskID:      doc["task_id"].(string),
			Title:       doc["title"].(string),
			Description: doc["description"].(string),
			Status:      doc["status"].(string),
			Priority:    doc["priority"].(string),
			CreatedAt:   int64(doc["created_at"].(float64)),
			UpdatedAt:   int64(doc["updated_at"].(float64)),
		}
		if assigneeID, ok := doc["assignee_id"].(string); ok {
			task.AssigneeID = assigneeID
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
