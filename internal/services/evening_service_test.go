package services

import (
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

type mockStatStorage struct{}

func (m *mockStatStorage) ShowTaskList() ([]entity.TaskResult, error) {
	return nil, nil
}

func (m *mockStatStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{
		Monday: entity.DaySchedule{
			Tasks: []entity.ScheduleTask{
				{Name: "home_task", Role: "learn", Time: 60, Priority: 5},
				{Name: "movies", Role: "rest", Time: 180, Priority: 3},
			},
		},
	}, nil
}

func (m *mockStatStorage) GetRecordsForDates(dates []string) ([]entity.TaskRecord, error) {
	return nil, nil
}

func TestEveningServiceDailyReset(t *testing.T) {
	mockStorage := &mockStatStorage{}
	statSrv := NewStatisticService(mockStorage)
	eveningSrv := NewEveningService(statSrv)

	// Mock time starting on 2026-08-12
	currentTime := time.Date(2026, 8, 12, 20, 0, 0, 0, time.UTC)
	eveningSrv.nowFunc = func() time.Time {
		return currentTime
	}

	// 1. Initial recommendation on Day 1 (movies has higher deficit: 180 min vs 60 min)
	res, err := eveningSrv.GetEveningFocus("", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CurrentTask.TaskName != "movies" {
		t.Errorf("expected current task 'movies', got '%s'", res.CurrentTask.TaskName)
	}

	// 2. Skip 'movies' on Day 1
	err = eveningSrv.SkipTask("movies")
	if err != nil {
		t.Fatalf("unexpected error skipping movies: %v", err)
	}

	// Now recommendation should advance to 'home_task'
	res, err = eveningSrv.GetEveningFocus("", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CurrentTask.TaskName != "home_task" {
		t.Errorf("expected current task 'home_task' after skip, got '%s'", res.CurrentTask.TaskName)
	}

	// 3. Skip 'home_task' on Day 1
	err = eveningSrv.SkipTask("home_task")
	if err != nil {
		t.Fatalf("unexpected error skipping home_task: %v", err)
	}

	// Now no candidates should remain for Day 1
	res, err = eveningSrv.GetEveningFocus("", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CurrentTask.TaskName != "" {
		t.Errorf("expected empty current task when all snoozed, got '%s'", res.CurrentTask.TaskName)
	}

	// 4. Advance time to Day 2 (2026-08-13)
	currentTime = time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

	// 5. Get Evening Focus on Day 2 - snoozed tasks should be automatically reset!
	res, err = eveningSrv.GetEveningFocus("", 20)
	if err != nil {
		t.Fatalf("unexpected error on day 2: %v", err)
	}
	if res.CurrentTask.TaskName != "movies" {
		t.Errorf("expected 'movies' to be recommended again on Day 2 after auto-reset, got '%s'", res.CurrentTask.TaskName)
	}
	if len(res.Candidates) != 2 {
		t.Errorf("expected 2 candidates available on Day 2, got %d", len(res.Candidates))
	}
}
