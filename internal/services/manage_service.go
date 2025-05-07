package services

import (
	"fmt"
	"strings"
	"tracker-server/internal/domain/entity"
)

// Common errors
var (
	ErrEmptyTaskName = fmt.Errorf("task name cannot be empty")
	ErrEmptyRole     = fmt.Errorf("role cannot be empty")
	ErrInvalidName   = fmt.Errorf("task name contains invalid characters")
)

// ManageStorage defines the interface for manage-related storage operations
type ManageStorage interface {
	// CreateTask creates a new task with the specified task definition
	// Returns an error if the task could not be created
	CreateTask(taskDefinition entity.TaskDefinition) error
}

// ManageService handles business logic for managing tasks
type ManageService struct {
	st ManageStorage
}

// NewManageService creates a new instance of ManageService
func NewManageService(st ManageStorage) *ManageService {
	return &ManageService{st: st}
}

// CreateTaskWithRole creates a new task with the specified name and role
// Valid roles might include: "admin", "user", "viewer" (documentation should specify allowed roles)
// Returns error if validation fails or if the storage operation fails
func (m *ManageService) CreateTaskWithRole(taskName string, role string) error {
	// Validate task name
	if taskName == "" {
		return ErrEmptyTaskName
	}

	if len(taskName) > 100 {
		return fmt.Errorf("task name too long (max 100 characters)")
	}

	if strings.ContainsAny(taskName, "<>\"'&") {
		return ErrInvalidName
	}

	// Validate role
	if role == "" {
		return ErrEmptyRole
	}

	// Call the storage layer to create the task
	if err := m.st.CreateTask(entity.TaskDefinition{Name: taskName, Role: role}); err != nil {
		return fmt.Errorf("failed to create task with role: %w", err)
	}

	return nil
}
