package services

import (
	"fmt"
	"log/slog"
)

// RestStorage defines the interface for rest-related storage operations
type RestStorage interface {
	AddRest(restTime int) error
	RestSpend(restTime int) error
	GetRest() (int, error)
}

// RestService handles business logic for rest operations
type RestService struct {
	st RestStorage
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

	if err := r.st.RestSpend(restTime); err != nil {
		slog.Error("failed to spend rest time",
			"operation", "rest_spend",
			"rest_time", restTime,
			"error", err)
		return fmt.Errorf("failed to spend rest time: %w", err)
	}

	return nil
}

// AddRest adds the specified rest time
func (r *RestService) AddRest(restTime int) error {
	if restTime <= 0 {
		return fmt.Errorf("invalid rest time: must be positive")
	}

	if err := r.st.AddRest(restTime); err != nil {
		slog.Error("failed to add rest time",
			"operation", "add_rest",
			"rest_time", restTime,
			"error", err)
		return fmt.Errorf("failed to add rest time: %w", err)
	}

	return nil
}

// RestGet retrieves the current rest time
func (r *RestService) RestGet() (int, error) {
	restTime, err := r.st.GetRest()
	if err != nil {
		slog.Error("failed to get rest time",
			"operation", "get_rest",
			"error", err)
		return 0, fmt.Errorf("failed to get rest time: %w", err)
	}

	return restTime, nil
}
