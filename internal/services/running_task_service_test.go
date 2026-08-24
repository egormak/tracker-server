package services

import (
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

func TestRunningTaskService_Adjust(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	now := time.Now()
	// Create an active running task with target 25m, started 5m ago
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:       "coding",
		Role:           "work",
		StartTime:      now.Add(-5 * time.Minute),
		Accumulated:    0,
		IsRunning:      true,
		TargetDuration: 25,
		DeadlineAt:     now.Add(20 * time.Minute),
	}

	// 1. Positive adjustment: +10 min
	adjusted, err := service.Adjust("coding", 10)
	if err != nil {
		t.Fatalf("unexpected error adjusting task: %v", err)
	}

	if adjusted.TargetDuration != 35 {
		t.Errorf("expected TargetDuration to be 35, got %d", adjusted.TargetDuration)
	}

	if adjusted.DeadlineAt.Before(now.Add(29 * time.Minute)) {
		t.Errorf("expected DeadlineAt to be extended by ~10 min, got %v", adjusted.DeadlineAt)
	}

	// 2. Negative adjustment: -5 min
	adjusted, err = service.Adjust("coding", -5)
	if err != nil {
		t.Fatalf("unexpected error adjusting task: %v", err)
	}

	if adjusted.TargetDuration != 30 {
		t.Errorf("expected TargetDuration to be 30, got %d", adjusted.TargetDuration)
	}

	// 3. Large negative adjustment should clamp to 0
	adjusted, err = service.Adjust("coding", -100)
	if err != nil {
		t.Fatalf("unexpected error adjusting task: %v", err)
	}

	if adjusted.TargetDuration != 0 {
		t.Errorf("expected TargetDuration to clamp to 0, got %d", adjusted.TargetDuration)
	}

	if !adjusted.DeadlineAt.IsZero() {
		t.Errorf("expected DeadlineAt to be zeroed when target is 0, got %v", adjusted.DeadlineAt)
	}

	// 4. Adjust non-existent task should error
	_, err = service.Adjust("non_existent", 5)
	if err == nil {
		t.Errorf("expected error adjusting non-existent task, got nil")
	}
}

// TestRunningTaskService_Adjust_OvertimeDeadlineMatchesStartAndResume ensures Adjust's
// remaining<=0 (already-overtime) handling doesn't set DeadlineAt to "now" the way it
// used to, which diverged from Start's and Resume's identical recompute (both leave
// DeadlineAt untouched in that case).
func TestRunningTaskService_Adjust_OvertimeDeadlineMatchesStartAndResume(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	now := time.Now()
	staleDeadline := now.Add(-3 * time.Minute)
	// 25m target, but 30m already accumulated/elapsed while still running: already overtime.
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:       "coding",
		Role:           "work",
		StartTime:      now.Add(-30 * time.Minute),
		Accumulated:    0,
		IsRunning:      true,
		TargetDuration: 25,
		DeadlineAt:     staleDeadline,
	}

	// Small positive adjustment that still leaves the task in overtime (target stays < elapsed).
	adjusted, err := service.Adjust("coding", 2)
	if err != nil {
		t.Fatalf("unexpected error adjusting overtime task: %v", err)
	}

	if adjusted.TargetDuration != 27 {
		t.Fatalf("expected TargetDuration 27, got %d", adjusted.TargetDuration)
	}

	if !adjusted.DeadlineAt.Equal(staleDeadline) {
		t.Errorf("expected DeadlineAt to be left untouched for an overtime adjustment (matching Start/Resume), got %v", adjusted.DeadlineAt)
	}
}
