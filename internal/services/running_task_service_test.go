package services

import (
	"errors"
	"reflect"
	"strings"
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

	// 4. Adjust with non-existent task name should error and NOT touch active task
	_, err = service.Adjust("non_existent", 5)
	if err == nil {
		t.Errorf("expected error adjusting non-existent task, got nil")
	}
	if storage.tasks["coding"].TargetDuration != 0 {
		t.Errorf("expected active task 'coding' TargetDuration to remain 0, got %d", storage.tasks["coding"].TargetDuration)
	}

	// 5. Adjust with empty taskName adjusts the currently active task
	adjusted, err = service.Adjust("", 15)
	if err != nil {
		t.Fatalf("unexpected error adjusting active task with empty name: %v", err)
	}
	if adjusted.TaskName != "coding" || adjusted.TargetDuration != 15 {
		t.Errorf("expected active task 'coding' adjusted to 15, got %s with duration %d", adjusted.TaskName, adjusted.TargetDuration)
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

type mockRunningTaskNotify struct {
	startCalls      []string
	stopCalls       []string
	completionCalls []completionCall
}

type completionCall struct {
	TaskName       string
	TimeDone       int
	TodayDone      int
	TargetDuration int
	RemainingTasks []string
	NextTask       string
	MsgID          int
}

func (m *mockRunningTaskNotify) SendMessageStart(taskName string) (int, error) {
	m.startCalls = append(m.startCalls, taskName)
	return 42, nil
}

func (m *mockRunningTaskNotify) SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error {
	m.stopCalls = append(m.stopCalls, taskName)
	return nil
}

func (m *mockRunningTaskNotify) SendMessageCompletion(taskName string, timeDone int, todayDone int, targetDuration int, remainingTasks []string, nextTask string, msgID int) error {
	m.completionCalls = append(m.completionCalls, completionCall{
		TaskName:       taskName,
		TimeDone:       timeDone,
		TodayDone:      todayDone,
		TargetDuration: targetDuration,
		RemainingTasks: remainingTasks,
		NextTask:       nextTask,
		MsgID:          msgID,
	})
	return nil
}

func (m *mockRunningTaskNotify) SendCustomMessage(message string) error {
	return nil
}

func TestRunningTaskService_Start_AttachesTelegramMessageID(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)

	task, err := service.Start("coding", "work", 30, "")
	if err != nil {
		t.Fatalf("unexpected error starting task: %v", err)
	}

	if task.TelegramMessageID != 42 {
		t.Errorf("expected TelegramMessageID to be 42, got %d", task.TelegramMessageID)
	}
	if len(nt.startCalls) != 1 || nt.startCalls[0] != "coding" {
		t.Errorf("expected SendMessageStart call for 'coding', got %v", nt.startCalls)
	}
}

func TestRunningTaskService_Stop_TelegramPushCompletion(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)

	todayDay := strings.ToLower(time.Now().Weekday().String())
	var sched entity.WeeklySchedule
	setDaySchedule(&sched, todayDay, entity.DaySchedule{
		Day: todayDay,
		Tasks: []entity.ScheduleTask{
			{Name: "coding", Time: 25},
			{Name: "english", Time: 20},
			{Name: "sport", Time: 30},
			{Name: "reading", Time: 15},
		},
	})
	storage.activeSchedule = sched

	// english has done=5 < 20 (remaining)
	// sport has done=30 >= 30 (completed, should NOT be in remaining)
	// reading has done=0 < 15 (remaining)
	storage.todayDurations["english"] = 5
	storage.todayDurations["sport"] = 30
	storage.todayDurations["reading"] = 0
	storage.todayDurations["coding"] = 25

	now := time.Now()
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:          "coding",
		Role:              "work",
		StartTime:         now.Add(-25 * time.Minute),
		IsRunning:         true,
		TargetDuration:    25,
		TelegramMessageID: 101,
	}

	record, err := service.Stop("coding", "manual")
	if err != nil {
		t.Fatalf("unexpected error stopping task: %v", err)
	}

	if len(nt.completionCalls) != 1 {
		t.Fatalf("expected 1 completion call, got %d", len(nt.completionCalls))
	}

	call := nt.completionCalls[0]
	if call.TaskName != "coding" {
		t.Errorf("expected TaskName 'coding', got %q", call.TaskName)
	}
	if call.TimeDone != record.TimeDuration {
		t.Errorf("expected TimeDone %d, got %d", record.TimeDuration, call.TimeDone)
	}
	if call.TodayDone != 25 {
		t.Errorf("expected TodayDone 25, got %d", call.TodayDone)
	}
	if call.TargetDuration != 25 {
		t.Errorf("expected TargetDuration 25, got %d", call.TargetDuration)
	}
	expectedRemaining := []string{"english", "reading"}
	if !reflect.DeepEqual(call.RemainingTasks, expectedRemaining) {
		t.Errorf("expected RemainingTasks %v, got %v", expectedRemaining, call.RemainingTasks)
	}
	if call.NextTask != "english" {
		t.Errorf("expected NextTask 'english', got %q", call.NextTask)
	}
	if call.MsgID != 101 {
		t.Errorf("expected MsgID 101, got %d", call.MsgID)
	}
}

func TestRunningTaskService_Stop_TelegramPushCompletion_AllDone(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)

	todayDay := strings.ToLower(time.Now().Weekday().String())
	var sched entity.WeeklySchedule
	setDaySchedule(&sched, todayDay, entity.DaySchedule{
		Day: todayDay,
		Tasks: []entity.ScheduleTask{
			{Name: "coding", Time: 25},
			{Name: "english", Time: 20},
		},
	})
	storage.activeSchedule = sched

	// All other tasks done
	storage.todayDurations["english"] = 20

	now := time.Now()
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:          "coding",
		Role:              "work",
		StartTime:         now.Add(-25 * time.Minute),
		IsRunning:         true,
		TargetDuration:    25,
		TelegramMessageID: 202,
	}

	_, err := service.Stop("coding", "manual")
	if err != nil {
		t.Fatalf("unexpected error stopping task: %v", err)
	}

	if len(nt.completionCalls) != 1 {
		t.Fatalf("expected 1 completion call, got %d", len(nt.completionCalls))
	}

	call := nt.completionCalls[0]
	if len(call.RemainingTasks) != 0 {
		t.Errorf("expected 0 remaining tasks, got %v", call.RemainingTasks)
	}
	if call.NextTask != "" {
		t.Errorf("expected empty NextTask, got %q", call.NextTask)
	}
	if call.MsgID != 202 {
		t.Errorf("expected MsgID 202, got %d", call.MsgID)
	}
}

func TestRunningTaskService_Stop_TelegramZeroMessageID(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)

	now := time.Now()
	// TelegramMessageID is 0
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:          "coding",
		Role:              "work",
		StartTime:         now.Add(-10 * time.Minute),
		IsRunning:         true,
		TargetDuration:    25,
		TelegramMessageID: 0,
	}

	_, err := service.Stop("coding", "manual")
	if err != nil {
		t.Fatalf("unexpected error stopping task: %v", err)
	}

	if len(nt.completionCalls) != 0 {
		t.Errorf("expected 0 completion calls when TelegramMessageID is 0, got %d", len(nt.completionCalls))
	}
}

func TestRunningTaskService_Stop_TelegramPushCompletion_BackfillUsesSourceDay(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	service := NewRunningTaskService(storage, nt)

	todayDay := strings.ToLower(time.Now().Weekday().String())
	sourceDay := strings.ToLower(time.Now().AddDate(0, 0, -1).Weekday().String())
	var sched entity.WeeklySchedule
	setDaySchedule(&sched, todayDay, entity.DaySchedule{Day: todayDay, Tasks: []entity.ScheduleTask{{Name: "coding", Time: 90}}})
	setDaySchedule(&sched, sourceDay, entity.DaySchedule{Day: sourceDay, Tasks: []entity.ScheduleTask{{Name: "coding", Time: 40}}})
	storage.activeSchedule = sched

	storage.todayDurations["coding"] = 70
	storage.dateDurations = map[string]int{"coding|" + sourceDay: 35}

	storage.tasks["coding"] = entity.RunningTask{
		TaskName:          "coding",
		Role:              "work",
		StartTime:         time.Now().Add(-15 * time.Minute),
		IsRunning:         true,
		SourceDay:         sourceDay,
		TelegramMessageID: 303,
	}

	if _, err := service.Stop("coding", "manual"); err != nil {
		t.Fatalf("unexpected error stopping task: %v", err)
	}
	if len(nt.completionCalls) != 1 {
		t.Fatalf("expected 1 completion call, got %d", len(nt.completionCalls))
	}
	call := nt.completionCalls[0]
	if call.TodayDone != 35 {
		t.Errorf("expected TodayDone from source day (35), got %d", call.TodayDone)
	}
	if call.TargetDuration != 40 {
		t.Errorf("expected TargetDuration from source day schedule (40), got %d", call.TargetDuration)
	}
}

func TestRunningTaskService_Pause_ExplicitNamedTaskCannotTouchActiveTask(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)

	now := time.Now()
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:       "coding",
		Role:           "work",
		StartTime:      now.Add(-10 * time.Minute),
		Accumulated:    0,
		IsRunning:      true,
		TargetDuration: 25,
	}

	// 1. Calling Pause("stale_task") when a different task is active must return error and NOT touch active task
	_, err := service.Pause("stale_task")
	if err == nil {
		t.Fatalf("expected error pausing stale/non-existent task, got nil")
	}

	activeTask, ok := storage.tasks["coding"]
	if !ok || !activeTask.IsRunning {
		t.Fatalf("active task 'coding' was mutated or paused by stale task pause")
	}

	// 2. Calling Adjust("stale_task", 5) when a different task is active must return error and NOT touch active task
	_, err = service.Adjust("stale_task", 5)
	if err == nil {
		t.Fatalf("expected error adjusting stale/non-existent task, got nil")
	}
	if storage.tasks["coding"].TargetDuration != 25 {
		t.Fatalf("active task 'coding' TargetDuration was mutated by stale task adjust")
	}

	// 3. Calling Pause("") pauses the active task
	paused, err := service.Pause("")
	if err != nil {
		t.Fatalf("unexpected error pausing active task with empty name: %v", err)
	}
	if paused.TaskName != "coding" {
		t.Errorf("expected paused task to be 'coding', got '%s'", paused.TaskName)
	}
	if paused.IsRunning {
		t.Errorf("expected paused task to have IsRunning=false")
	}
	if paused.Accumulated < 10 {
		t.Errorf("expected accumulated duration >= 10, got %d", paused.Accumulated)
	}

	// 4. Calling Pause("") when no active task exists should error
	_, err = service.Pause("")
	if err == nil {
		t.Errorf("expected error pausing with no active task, got nil")
	}
}

func TestRunningTaskService_Stop_CallsAutoRecalculateOnRecord(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)
	rampMock := &mockRampRecalculator{}
	service.SetRampService(rampMock)

	now := time.Now()
	storage.tasks["coding"] = entity.RunningTask{
		TaskName:       "coding",
		Role:           "work",
		StartTime:      now.Add(-20 * time.Minute),
		Accumulated:    0,
		IsRunning:      true,
		TargetDuration: 25,
	}

	record, err := service.Stop("coding", "manual")
	if err != nil {
		t.Fatalf("unexpected error stopping task: %v", err)
	}

	if !rampMock.called {
		t.Fatalf("expected AutoRecalculateOnRecord to be called")
	}
	if rampMock.taskName != "coding" {
		t.Errorf("expected taskName 'coding', got '%s'", rampMock.taskName)
	}
	if rampMock.duration != record.TimeDuration {
		t.Errorf("expected duration %d, got %d", record.TimeDuration, rampMock.duration)
	}
}

