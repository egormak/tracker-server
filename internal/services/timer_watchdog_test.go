package services

import (
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

type MockRunningTaskStorage struct {
	tasks                map[string]entity.RunningTask
	records              []entity.TaskRecord
	addTaskRecordErr     error
	deleteRunningTaskErr error
	todayDurations       map[string]int
	dateDurations        map[string]int
	activeSchedule       entity.WeeklySchedule
	taskParams           map[string]entity.TaskParams
	strictTasks          map[string]bool
}

func NewMockRunningTaskStorage() *MockRunningTaskStorage {
	return &MockRunningTaskStorage{
		tasks:          make(map[string]entity.RunningTask),
		todayDurations: make(map[string]int),
		taskParams:     make(map[string]entity.TaskParams),
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
	if m.deleteRunningTaskErr != nil {
		return m.deleteRunningTaskErr
	}
	delete(m.tasks, taskName)
	return nil
}

func (m *MockRunningTaskStorage) GetRole(taskName string) (string, error) {
	return "work", nil
}

func (m *MockRunningTaskStorage) AddTaskRecord(task entity.TaskRecord) error {
	if m.addTaskRecordErr != nil {
		return m.addTaskRecordErr
	}
	m.records = append(m.records, task)
	return nil
}

func (m *MockRunningTaskStorage) AddRoleMinutes(task entity.TaskRecord) error {
	return nil
}

func (m *MockRunningTaskStorage) TimeListDelDB(timeDuretion int) error {
	return nil
}

func (m *MockRunningTaskStorage) GetTodayTaskDuration(taskName string) (int, error) {
	if m.todayDurations != nil {
		if d, ok := m.todayDurations[taskName]; ok {
			return d, nil
		}
	}
	return 0, nil
}

func (m *MockRunningTaskStorage) GetTaskDurationForDate(taskName string, date string, sourceDay string) (int, error) {
	if m.dateDurations != nil {
		if d, ok := m.dateDurations[taskName+"|"+sourceDay]; ok {
			return d, nil
		}
	}
	return 0, nil
}

func (m *MockRunningTaskStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return m.activeSchedule, nil
}

func (m *MockRunningTaskStorage) GetTaskParams(taskName string) (entity.TaskParams, error) {
	if m.taskParams != nil {
		if p, ok := m.taskParams[taskName]; ok {
			return p, nil
		}
	}
	return entity.TaskParams{}, nil
}

func (m *MockRunningTaskStorage) IsTaskStrict(taskName string) (bool, error) {
	return m.strictTasks[taskName], nil
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

func TestTimerWatchdog_HeartbeatFreeClientIsNotPaused(t *testing.T) {
	for _, client := range []string{"bot", "mcp", "miniapp"} {
		storage := NewMockRunningTaskStorage()
		nt := &mockRunningTaskNotify{}
		watchdog := NewTimerWatchdog(NewRunningTaskService(storage, nt))

		// Started 4 minutes ago from a client that never sends heartbeats.
		storage.tasks["telegram"] = entity.RunningTask{
			TaskName:        "telegram",
			Role:            "rest",
			StartTime:       time.Now().Add(-4 * time.Minute),
			LastHeartbeatAt: time.Now().Add(-4 * time.Minute),
			IsRunning:       true,
			TargetDuration:  60,
			RunClient:       client,
		}

		watchdog.checkRunningTasks()

		if !storage.tasks["telegram"].IsRunning {
			t.Errorf("%s: expected the run to keep going without heartbeats, it was paused", client)
		}
		if len(nt.reminderCalls) != 0 {
			t.Errorf("%s: expected no reminder before %v, got %d", client, RunningReminderInterval, len(nt.reminderCalls))
		}
	}
}

func TestTimerWatchdog_HeartbeatFreeClientGetsReminders(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)
	watchdog := NewTimerWatchdog(service)

	start := time.Now().Add(-6 * time.Minute)
	storage.tasks["telegram"] = entity.RunningTask{
		TaskName:        "telegram",
		Role:            "rest",
		StartTime:       start,
		LastHeartbeatAt: start,
		IsRunning:       true,
		TargetDuration:  60,
		Accumulated:     2,
		RunClient:       "bot",
	}

	watchdog.checkRunningTasks()
	if len(nt.reminderCalls) != 1 {
		t.Fatalf("expected 1 reminder after 6 min, got %d", len(nt.reminderCalls))
	}
	if got := nt.reminderCalls[0]; got.TaskName != "telegram" || got.Elapsed != 8 || got.Target != 60 || got.PrevMsgID != 0 {
		t.Errorf("unexpected first reminder: %+v", got)
	}

	// The next watchdog tick is inside the interval: no second card.
	watchdog.checkRunningTasks()
	if len(nt.reminderCalls) != 1 {
		t.Fatalf("expected no repeat inside the interval, got %d reminders", len(nt.reminderCalls))
	}

	// Five minutes later the next card replaces the previous one.
	later := time.Now().Add(RunningReminderInterval + time.Second)
	if !service.RemindIfDue("telegram", later) {
		t.Fatal("expected a second reminder after the interval")
	}
	if got := nt.reminderCalls[1]; got.PrevMsgID != 101 {
		t.Errorf("expected the second reminder to replace card 101, got %+v", got)
	}

	// Stopping the run removes the last reminder card.
	if _, err := service.Stop("telegram"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	found := false
	for _, id := range nt.deletedMessages {
		if id == 102 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reminder card 102 to be deleted on stop, deleted: %v", nt.deletedMessages)
	}
}

func TestRunningTaskService_RemindIfDue_SilentWhileHeartbeating(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)

	// Started from the bot, but an open CLI/web timer is sending heartbeats.
	storage.tasks["telegram"] = entity.RunningTask{
		TaskName:        "telegram",
		StartTime:       time.Now().Add(-10 * time.Minute),
		LastHeartbeatAt: time.Now(),
		IsRunning:       true,
		RunClient:       "bot",
	}
	if service.RemindIfDue("telegram", time.Now()) {
		t.Error("expected no reminder while heartbeats arrive")
	}

	// A lease-held client (cli) never gets reminders: the lease pauses it instead.
	storage.tasks["telegram"] = entity.RunningTask{
		TaskName:        "telegram",
		StartTime:       time.Now().Add(-10 * time.Minute),
		LastHeartbeatAt: time.Now().Add(-10 * time.Minute),
		IsRunning:       true,
		RunClient:       "cli",
	}
	if service.RemindIfDue("telegram", time.Now()) {
		t.Error("expected no reminder for a cli run")
	}
}

func TestRunningTaskService_RunClientFollowsStartAndResume(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	task, err := service.StartWith(StartOptions{TaskName: "telegram", Role: "rest", TargetDuration: 30, Client: "bot"})
	if err != nil {
		t.Fatalf("StartWith failed: %v", err)
	}
	if task.RunClient != "bot" {
		t.Errorf("expected run_client bot after start, got %q", task.RunClient)
	}
	if _, err := service.Pause("telegram"); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	task, err = service.Resume("telegram", "cli")
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	if task.RunClient != "cli" || task.SourceDevice != "bot" {
		t.Errorf("expected run_client cli and source_device bot after a cli resume, got %q / %q", task.RunClient, task.SourceDevice)
	}
}
