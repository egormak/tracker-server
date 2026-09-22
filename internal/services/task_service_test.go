package services

import (
	"errors"
	"strings"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"
)

type mockTaskStorage struct {
	taskParams      map[string]entity.TaskParams
	getParamsErr    error
	schedule        entity.WeeklySchedule
	scheduleErr     error
	dayRecords      map[string]int
	setParamsCalled bool
	lastSetParams   entity.TaskParams
	setParamsErr    error
}

func newMockTaskStorage() *mockTaskStorage {
	return &mockTaskStorage{
		taskParams: make(map[string]entity.TaskParams),
		dayRecords: make(map[string]int),
	}
}

func (m *mockTaskStorage) SetTaskParams(params entity.TaskParams) error {
	m.setParamsCalled = true
	m.lastSetParams = params
	if m.setParamsErr != nil {
		return m.setParamsErr
	}
	m.taskParams[params.Name] = params
	return nil
}

func (m *mockTaskStorage) GetTaskParams(taskName string) (entity.TaskParams, error) {
	if m.getParamsErr != nil {
		return entity.TaskParams{}, m.getParamsErr
	}
	params, ok := m.taskParams[taskName]
	if !ok {
		return entity.TaskParams{}, storage.ErrTaskNotFound
	}
	return params, nil
}

func (m *mockTaskStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	if m.scheduleErr != nil {
		return entity.WeeklySchedule{}, m.scheduleErr
	}
	return m.schedule, nil
}

func (m *mockTaskStorage) GetTaskDurationForDate(taskName string, date string, sourceDay string) (int, error) {
	return 0, nil
}

func (m *mockTaskStorage) GetDayTaskRecord(taskName string) (int, error) {
	return m.dayRecords[taskName], nil
}

type mockTaskNotify struct{}

func (n *mockTaskNotify) SendMessageStart(taskName string) (int, error) {
	return 1, nil
}

func (n *mockTaskNotify) SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error {
	return nil
}

func setDaySchedule(sched *entity.WeeklySchedule, day string, ds entity.DaySchedule) {
	switch strings.ToLower(day) {
	case "monday":
		sched.Monday = ds
	case "tuesday":
		sched.Tuesday = ds
	case "wednesday":
		sched.Wednesday = ds
	case "thursday":
		sched.Thursday = ds
	case "friday":
		sched.Friday = ds
	case "saturday":
		sched.Saturday = ds
	case "sunday":
		sched.Sunday = ds
	}
}

func TestGetTaskParams_FoundInTodayTaskList(t *testing.T) {
	st := newMockTaskStorage()
	st.taskParams["coding"] = entity.TaskParams{Name: "coding", Time: 60, Priority: 1}
	svc := NewTaskService(st, &mockTaskNotify{})

	params, err := svc.GetTaskParams("coding")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Name != "coding" || params.Time != 60 || params.Priority != 1 {
		t.Errorf("unexpected params: %+v", params)
	}
}

func TestGetTaskParams_FoundInTodaySchedule_WhenTaskListMisses(t *testing.T) {
	st := newMockTaskStorage()
	st.getParamsErr = storage.ErrTaskNotFound

	today := strings.ToLower(time.Now().Weekday().String())
	var sched entity.WeeklySchedule
	sched.IsActive = true
	setDaySchedule(&sched, today, entity.DaySchedule{
		Day: today,
		Tasks: []entity.ScheduleTask{
			{Name: "today_task", Time: 90, Priority: 3},
		},
	})
	st.schedule = sched

	svc := NewTaskService(st, &mockTaskNotify{})

	params, err := svc.GetTaskParams("today_task")
	if err != nil {
		t.Fatalf("expected to find task in today's schedule, got error: %v", err)
	}
	if params.Name != "today_task" || params.Time != 90 || params.Priority != 3 {
		t.Errorf("unexpected params: %+v", params)
	}
}

func TestGetTaskParams_FoundInPreviousDaySchedule_Rollover(t *testing.T) {
	st := newMockTaskStorage()
	st.getParamsErr = storage.ErrTaskNotFound

	today := strings.ToLower(time.Now().Weekday().String())
	previousDays := getPreviousDaysForTask(today)
	if len(previousDays) == 0 {
		t.Skip("today is Monday, no previous days in week to test rollover")
	}

	targetPrevDay := previousDays[0]
	var sched entity.WeeklySchedule
	sched.IsActive = true
	setDaySchedule(&sched, targetPrevDay, entity.DaySchedule{
		Day: targetPrevDay,
		Tasks: []entity.ScheduleTask{
			{Name: "rollover_task", Time: 45, Priority: 2},
		},
	})
	st.schedule = sched

	svc := NewTaskService(st, &mockTaskNotify{})

	params, err := svc.GetTaskParams("rollover_task")
	if err != nil {
		t.Fatalf("expected to find rollover task in %s schedule, got error: %v", targetPrevDay, err)
	}
	if params.Name != "rollover_task" || params.Time != 45 || params.Priority != 2 {
		t.Errorf("unexpected params: %+v", params)
	}
}

func TestGetTaskParams_NotFoundAnywhere(t *testing.T) {
	st := newMockTaskStorage()
	st.getParamsErr = storage.ErrTaskNotFound
	st.schedule = entity.WeeklySchedule{IsActive: true}

	svc := NewTaskService(st, &mockTaskNotify{})

	_, err := svc.GetTaskParams("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task, got nil")
	}
}

func TestGetTaskParams_ScheduleError(t *testing.T) {
	st := newMockTaskStorage()
	st.getParamsErr = storage.ErrTaskNotFound
	st.scheduleErr = errors.New("db connection lost")

	svc := NewTaskService(st, &mockTaskNotify{})

	_, err := svc.GetTaskParams("task1")
	if err == nil {
		t.Fatal("expected error when schedule fails, got nil")
	}
	if !errors.Is(err, storage.ErrTaskNotFound) {
		t.Errorf("expected original error ErrTaskNotFound, got: %v", err)
	}
}

func TestSetTaskParams(t *testing.T) {
	st := newMockTaskStorage()
	svc := NewTaskService(st, &mockTaskNotify{})

	input := entity.TaskParams{Name: "writing", Time: 30, Priority: 2}
	err := svc.SetTaskParams(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.setParamsCalled {
		t.Error("expected SetTaskParams to be called on storage")
	}
	if st.lastSetParams != input {
		t.Errorf("expected %+v, got %+v", input, st.lastSetParams)
	}
}

func TestGetDayTaskRecord(t *testing.T) {
	st := newMockTaskStorage()
	st.dayRecords["reading"] = 120
	svc := NewTaskService(st, &mockTaskNotify{})

	val, err := svc.GetDayTaskRecord("reading")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 120 {
		t.Errorf("expected 120, got %d", val)
	}
}
