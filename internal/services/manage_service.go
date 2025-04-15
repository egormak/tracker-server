package services

import (
	"fmt"
)

// ManageStorage defines the interface for manage-related storage operations
type ManageStorage interface {
	CreateTask(taskName string, role string) error
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
func (m *ManageService) CreateTaskWithRole(taskName string, role string) error {
	if taskName == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	// Call the storage layer to create the task
	if err := m.st.CreateTask(taskName, role); err != nil {
		return fmt.Errorf("failed to create task with role: %w", err)
	}

	return nil
}
