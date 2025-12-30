package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	flin "github.com/skshohagmiah/flin/clients/go"
)

// UserService handles user operations
type UserService struct {
	client *flin.Client
}

// NewUserService creates a new user service
func NewUserService(client *flin.Client) *UserService {
	return &UserService{client: client}
}

// User represents a user in the system
type User struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
}

// CreateUser creates a new user
func (s *UserService) CreateUser(email, name, role string) (*User, error) {
	user := &User{
		UserID:    uuid.New().String(),
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: time.Now().Unix(),
	}

	doc := map[string]interface{}{
		"user_id":    user.UserID,
		"email":      user.Email,
		"name":       user.Name,
		"role":       user.Role,
		"created_at": float64(user.CreatedAt),
	}

	_, err := s.client.DB.Insert("users", doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(userID string) (*User, error) {
	results, err := s.client.DB.Query("users").
		Where("user_id", flin.Eq, userID).
		Take(1).
		Exec()

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	doc := results[0]
	user := &User{
		UserID:    doc["user_id"].(string),
		Email:     doc["email"].(string),
		Name:      doc["name"].(string),
		Role:      doc["role"].(string),
		CreatedAt: int64(doc["created_at"].(float64)),
	}

	return user, nil
}

// ListUsers lists all users
func (s *UserService) ListUsers() ([]*User, error) {
	results, err := s.client.DB.Query("users").Exec()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*User, 0, len(results))
	for _, doc := range results {
		user := &User{
			UserID:    doc["user_id"].(string),
			Email:     doc["email"].(string),
			Name:      doc["name"].(string),
			Role:      doc["role"].(string),
			CreatedAt: int64(doc["created_at"].(float64)),
		}
		users = append(users, user)
	}

	return users, nil
}
