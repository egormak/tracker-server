package services

import (
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

type MockRunningTaskStorage struct {
	tasks   map[string]entity.RunningTask
	records []entity.TaskRecord
}

func NewMockRunningTaskStorage() *MockRunningTaskStorage {
	return &MockRunningTaskStorage{
		tasks: make(map[string]entity.RunningTask),
	}
}

func (m *MockRunningTaskStorage) GetRunningTask(taskName string) (entity.RunningTask, error) {
	if t, ok := m.tasks[taskName]; ok {
		return t, nil
	}
	return entity.RunningTask{}, nil
}

func (m *MockRunningTaskStorage) GetActiveRunningTask() (entity.RunningTask, error) {
	for _, t := range m.tasks {
		if t.IsRunning {
			return t, nil
		}
	}
	return entity.RunningTask{}, nil
}

func (m *MockRunningTaskStorage) GetAllRunningTasks() ([]entity.RunningTask, error) {
	var list []entity.RunningTask
	for _, t := range m.tasks {
		list = append(list, t)
	}
	return list, nil
}

func (m *MockRunningTaskStorage) UpsertRunningTask(task entity.RunningTask) error {
	m.tasks[task.TaskName] = task
	return nil
}

func (m *MockRunningTaskStorage) DeleteRunningTask(taskName string) error {
	delete(m.tasks, taskName)
	return nil
}

func (m *MockRunningTaskStorage) GetRole(taskName string) (string, error) {
	return "work", nil
}

func (m *MockRunningTaskStorage) AddTaskRecord(task entity.TaskRecord) error {
	m.records = append(m.records, task)
	return nil
}

func (m *MockRunningTaskStorage) AddRoleMinutes(task entity.TaskRecord) error {
	return nil
}

func (m *MockRunningTaskStorage) AddRest(restTime int) error {
	return nil
}

func (m *MockRunningTaskStorage) TimeListDelDB(timeDuretion int) error {
	return nil
}

func (m *MockRunningTaskStorage) GetTodayTaskDuration(taskName string) (int, error) {
	return 0, nil
}

func (m *MockRunningTaskStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{}, nil
}

func (m *MockRunningTaskStorage) GetTaskParams(taskName string) (entity.TaskParams, error) {
	return entity.TaskParams{}, nil
}

func (m *MockRunningTaskStorage) IsTaskStrict(taskName string) (bool, error) {
	return false, nil
}

func TestTimerWatchdog_TargetDurationAutoStop(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	runningService := NewRunningTaskService(storage, nil)
	watchdog := NewTimerWatchdog(runningService)

	// Simulate a task with 25m target duration started 26 minutes ago
	storage.tasks["pomodoro_task"] = entity.RunningTask{
		TaskName:        "pomodoro_task",
		Role:            "work",
		StartTime:       time.Now().Add(-26 * time.Minute),
		LastHeartbeatAt: time.Now(),
		IsRunning:       true,
		TargetDuration:  25,
	}

	watchdog.checkRunningTasks()

	// Verify the task was auto-stopped and removed from running tasks
	if _, ok := storage.tasks["pomodoro_task"]; ok {
		t.Errorf("Expected pomodoro_task to be stopped and removed, but it is still in running tasks")
	}

	// Verify a task record was created
	if len(storage.records) != 1 {
		t.Fatalf("Expected 1 task record created, got %d", len(storage.records))
	}
	if storage.records[0].Name != "pomodoro_task" {
		t.Errorf("Expected record for pomodoro_task, got %s", storage.records[0].Name)
	}
}

func TestTimerWatchdog_SafetyCapAutoStop(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	runningService := NewRunningTaskService(storage, nil)
	watchdog := NewTimerWatchdog(runningService)

	// Simulate a free stopwatch task (target_duration = 0) that exceeded 20 minutes
	storage.tasks["free_task"] = entity.RunningTask{
		TaskName:        "free_task",
		Role:            "work",
		StartTime:       time.Now().Add(-21 * time.Minute),
		LastHeartbeatAt: time.Now(),
		IsRunning:       true,
		TargetDuration:  0,
	}

	watchdog.checkRunningTasks()

	// Verify the task was auto-stopped due to safety cap (20m)
	if _, ok := storage.tasks["free_task"]; ok {
		t.Errorf("Expected free_task to be stopped by safety cap, but it is still running")
	}

	if len(storage.records) != 1 {
		t.Fatalf("Expected 1 task record created, got %d", len(storage.records))
	}
}

func TestTimerWatchdog_HeartbeatLeaseExpiry(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	runningService := NewRunningTaskService(storage, nil)
	watchdog := NewTimerWatchdog(runningService)

	// Simulate a task whose last heartbeat was 4 minutes ago (lease is 3 min)
	storage.tasks["abandoned_task"] = entity.RunningTask{
		TaskName:        "abandoned_task",
		Role:            "work",
		StartTime:       time.Now().Add(-10 * time.Minute),
		LastHeartbeatAt: time.Now().Add(-4 * time.Minute),
		IsRunning:       true,
		TargetDuration:  30,
	}

	watchdog.checkRunningTasks()

	// Verify the task was paused on heartbeat timeout
	task := storage.tasks["abandoned_task"]
	if task.IsRunning {
		t.Errorf("Expected abandoned_task to be paused on heartbeat timeout, but IsRunning is true")
	}
}

func TestRunningTaskService_Heartbeat(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	runningService := NewRunningTaskService(storage, nil)

	// Start task
	task, err := runningService.Start("hb_task", "work", 25, "")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	firstHb := task.LastHeartbeatAt
	time.Sleep(10 * time.Millisecond)

	updated, err := runningService.Heartbeat("hb_task")
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if !updated.LastHeartbeatAt.After(firstHb) {
		t.Errorf("Expected LastHeartbeatAt to be updated, got %v <= %v", updated.LastHeartbeatAt, firstHb)
	}
}
