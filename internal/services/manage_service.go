package services

import (
	"errors"
	"fmt"
	"strings"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"
)

// Common errors
var (
	ErrEmptyTaskName            = fmt.Errorf("task name cannot be empty")
	ErrEmptyRole                = fmt.Errorf("role cannot be empty")
	ErrInvalidName              = fmt.Errorf("task name contains invalid characters")
	ErrInvalidPlanPercentGroup  = fmt.Errorf("invalid plan percent group")
	ErrInvalidPlanPercentValue  = fmt.Errorf("invalid plan percent value")
	ErrPlanPercentValueNotFound = fmt.Errorf("plan percent value not found")
)

// ManageStorage defines the interface for manage-related storage operations
type ManageStorage interface {
	// CreateTask creates a new task with the specified task definition
	// Returns an error if the task could not be created
	CreateTask(taskDefinition entity.TaskDefinition) error

	// GetPlanProcents retrieves the plan percents configuration
	GetPlanProcents() (storage.Procents, error)

	// RemovePlanPercent removes a specific percent value from the given group
	RemovePlanPercent(group string, value int) error
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

// GetPlanPercents retrieves the plan percents for plan, work, learn and rest
// Returns the plan percents data or an error if retrieval fails
func (m *ManageService) GetPlanPercents() (*entity.PlanPercents, error) {
	procents, err := m.st.GetPlanProcents()
	if err != nil {
		return nil, fmt.Errorf("failed to get plan percents: %w", err)
	}

	result := &entity.PlanPercents{
		Title:         procents.Title,
		Date:          procents.Date,
		CurrentChoice: procents.CurrentChoice,
		Plans:         procents.Plans,
		Plan:          procents.Plan,
		Work:          procents.Work,
		Learn:         procents.Learn,
		Rest:          procents.Rest,
	}

	return result, nil
}

var planPercentGroups = map[string]struct{}{
	"plan":  {},
	"work":  {},
	"learn": {},
	"rest":  {},
}

// RemovePlanPercent removes a specific value from a plan percent group
func (m *ManageService) RemovePlanPercent(group string, value int) error {
	group = strings.ToLower(strings.TrimSpace(group))
	if _, ok := planPercentGroups[group]; !ok {
		return ErrInvalidPlanPercentGroup
	}

	if value <= 0 || value > 100 {
		return ErrInvalidPlanPercentValue
	}

	if err := m.st.RemovePlanPercent(group, value); err != nil {
		if errors.Is(err, storage.ErrPlanPercentValueNotFound) {
			return ErrPlanPercentValueNotFound
		}
		return fmt.Errorf("failed to remove plan percent: %w", err)
	}

	return nil
}
