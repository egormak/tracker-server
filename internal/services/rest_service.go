package services

import (
	"fmt"
	"log/slog"
	"sync"
)

const (
	// MaxDailyRestUnits defines the 60 minutes cap (60 * 100 units).
	MaxDailyRestUnits = 6000
)

// RestStorage defines the interface for rest-related storage operations
type RestStorage interface {
	AddRest(restTime int) error
	AddRestMinutes(minutes int) error
	RestSpend(restTime int) error
	GetRest() (int, error)
	ResetRest() error
}

// RestService handles business logic for rest operations
type RestService struct {
	st RestStorage
	mu sync.Mutex
}

// NewRestService creates a new instance of RestService
func NewRestService(st RestStorage) *RestService {
	return &RestService{st: st}
}

// RestSpend deducts the specified rest time
func (r *RestService) RestSpend(restTime int) error {
	if restTime <= 0 {
		return fmt.Errorf("invalid rest time: must be positive")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.st.RestSpend(restTime); err != nil {
		slog.Error("failed to spend rest time",
			"operation", "rest_spend",
			"rest_time", restTime,
			"error", err)
		return fmt.Errorf("failed to spend rest time: %w", err)
	}

	return nil
}

// AddRest adds the specified rest minutes (minutes * 100 units)
func (r *RestService) AddRest(restMinutes int) error {
	if restMinutes <= 0 {
		return fmt.Errorf("invalid rest time: must be positive")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.st.AddRestMinutes(restMinutes); err != nil {
		slog.Error("failed to add rest time",
			"operation", "add_rest",
			"rest_time", restMinutes,
			"error", err)
		return fmt.Errorf("failed to add rest time: %w", err)
	}

	return nil
}

// RestGet retrieves the current rest time
func (r *RestService) RestGet() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	restTime, err := r.st.GetRest()
	if err != nil {
		slog.Error("failed to get rest time",
			"operation", "get_rest",
			"error", err)
		return 0, fmt.Errorf("failed to get rest time: %w", err)
	}

	return restTime, nil
}

// ResetRest resets the rest time
func (r *RestService) ResetRest() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.st.ResetRest(); err != nil {
		slog.Error("failed to reset rest time",
			"operation", "reset_rest",
			"error", err)
		return fmt.Errorf("failed to reset rest time: %w", err)
	}

	return nil
}

// GetRest retrieves the current rest time
func (r *RestService) GetRest() (int, error) {
	return r.RestGet()
}
