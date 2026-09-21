package services

import (
	"errors"
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

func TestRunningTaskService_SeamlessSwitching(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	// 1. Start task A ("work")
	taskA, err := service.Start("work", "work", 30, "")
	if err != nil {
		t.Fatalf("failed to start task A: %v", err)
	}
	if !taskA.IsRunning || taskA.TaskName != "work" {
		t.Fatalf("expected task A to be running, got %+v", taskA)
	}

	// 2. Seamlessly start task B ("english") while task A is running
	taskB, err := service.Start("english", "learn", 20, "")
	if err != nil {
		t.Fatalf("failed to seamlessly switch to task B: %v", err)
	}
	if !taskB.IsRunning || taskB.TaskName != "english" {
		t.Fatalf("expected task B to be running, got %+v", taskB)
	}

	// 3. Verify task A was auto-stopped and saved to storage.records
	if len(storage.records) != 1 {
		t.Fatalf("expected 1 task record for stopped task A, got %d", len(storage.records))
	}
	recordA := storage.records[0]
	if recordA.Name != "work" {
		t.Errorf("expected record name 'work', got '%s'", recordA.Name)
	}
	if recordA.CreatedAt.IsZero() {
		t.Errorf("expected non-zero CreatedAt on stopped task record")
	}

	// 4. Verify task A is no longer in running tasks and task B is active
	active, err := storage.GetActiveRunningTask()
	if err != nil {
		t.Fatalf("failed to get active task: %v", err)
	}
	if active.TaskName != "english" {
		t.Errorf("expected active task 'english', got '%s'", active.TaskName)
	}
	if _, ok := storage.tasks["work"]; ok {
		t.Errorf("expected task A 'work' to be deleted from running tasks")
	}
}

func TestRunningTaskService_Stop_CreatedAt(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	_, err := service.Start("reading", "learn", 15, "")
	if err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	record, err := service.Stop("reading", "manual")
	if err != nil {
		t.Fatalf("failed to stop task: %v", err)
	}

	if record.Name != "reading" {
		t.Errorf("expected record name 'reading', got '%s'", record.Name)
	}
	if record.CreatedAt.IsZero() {
		t.Errorf("expected non-zero CreatedAt on task record")
	}
	if record.CreatedAt.Location() != time.UTC {
		t.Errorf("expected CreatedAt in UTC, got %v", record.CreatedAt.Location())
	}
}

func TestRunningTaskService_SeamlessSwitching_AutoStopFailure(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	// 1. Start task A ("work")
	_, err := service.Start("work", "work", 30, "")
	if err != nil {
		t.Fatalf("failed to start task A: %v", err)
	}

	// 2. Inject failure when auto-stopping task A
	storage.addTaskRecordErr = errors.New("database write failure during stop")

	// 3. Attempt seamless switch to task B ("english")
	_, err = service.Start("english", "learn", 20, "")
	if err == nil {
		t.Fatalf("expected Start to fail when auto-stop fails, got nil")
	}

	// 4. Verify invariant: task B was NOT started, and task A remains in storage (no concurrent zombie task created)
	if _, ok := storage.tasks["english"]; ok {
		t.Errorf("task B 'english' should NOT have been started after auto-stop failure")
	}
	if taskA, ok := storage.tasks["work"]; !ok || !taskA.IsRunning {
		t.Errorf("task A 'work' should remain intact after failed auto-stop switch, got %+v", taskA)
	}
}


