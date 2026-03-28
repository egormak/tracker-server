package services

import (
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/service"
)

type MockTaskRecordStorage struct {
	Records []entity.TaskRecord
}

func (m *MockTaskRecordStorage) GetRole(taskName string) (string, error) {
	return "work", nil
}
func (m *MockTaskRecordStorage) AddTaskRecord(task entity.TaskRecord) error {
	m.Records = append(m.Records, task)
	return nil
}
func (m *MockTaskRecordStorage) AddRoleMinutes(task entity.TaskRecord) error {
	return nil
}
func (m *MockTaskRecordStorage) AddRest(restTime int) error {
	return nil
}
func (m *MockTaskRecordStorage) GetGroupPlanPercent() (int, error) {
	return 0, nil
}
func (m *MockTaskRecordStorage) GetGroupPercent(groupPlanOrdinal int) (int, error) {
	return 0, nil
}
func (m *MockTaskRecordStorage) CheckIfPlanPercentEmpty() error {
	return nil
}
func (m *MockTaskRecordStorage) ChangeGroupPlanPercent(groupPlanOrdinal int) error {
	return nil
}
func (m *MockTaskRecordStorage) GetGroupName(groupPlanOrdinal int) (string, error) {
	return "", nil
}
func (m *MockTaskRecordStorage) GetTaskNamePlanPercent(groupName string, groupPercent int) (string, error) {
	return "", nil
}
func (m *MockTaskRecordStorage) DelGroupPercent(groupName string) error {
	return nil
}
func (m *MockTaskRecordStorage) GetTodayTaskDuration(taskName string) (int, error) {
	return 0, nil
}
func (m *MockTaskRecordStorage) GetTaskParams(taskName string) (entity.TaskParams, error) {
	return entity.TaskParams{}, nil
}

// Key Methods for Backfill
func (m *MockTaskRecordStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{
		Monday: entity.DaySchedule{
			Tasks: []entity.ScheduleTask{
				{Name: "test_task", Time: 60},
			},
		},
	}, nil
}
func (m *MockTaskRecordStorage) GetTaskDurationForDate(taskName string, date string) (int, error) {
	return 0, nil
}

func TestBackfillLogic(t *testing.T) {
	// Only run if today is NOT Monday, otherwise backfill from Monday won't work
	if time.Now().Weekday() == time.Monday {
		t.Skip("Skipping backfill test on Monday (cannot backfill Monday)")
	}

	mockStorage := &MockTaskRecordStorage{}
	svc := NewTaskRecordService(mockStorage)

	req := entity.TaskRecordRequest{
		TaskName:        "test_task",
		TimeDone:        100,
		ManageByService: true,
	}

	if err := svc.AddRecord(req); err != nil {
		t.Fatalf("AddRecord failed: %v", err)
	}

	// Expect 2 records:
	// 1. Monday (60 mins)
	// 2. Today (40 mins)
	// Note: If today is e.g. Tuesday, it checks Monday.
	// If today is actually Monday (skipped above), loop doesn't run.

	// Also note: logic iterates Monday -> Today.
	// Iterate through mockStorage.Records and check

	totalMinutes := 0
	foundMonday := false
	foundToday := false

	mondayDate := service.CalculateDateForDay("monday")
	todayDate := time.Now().Format("2 January 2006")

	for _, r := range mockStorage.Records {
		totalMinutes += r.TimeDuration
		if r.Date == mondayDate && r.TimeDuration == 60 {
			foundMonday = true
		}
		if r.Date == todayDate {
			foundToday = true
			if r.TimeDuration != 40 {
				t.Errorf("Expected today record to be 40, got %d", r.TimeDuration)
			}
		}
	}

	if totalMinutes != 100 {
		t.Errorf("Expected total 100 mins, got %d", totalMinutes)
	}
	if !foundMonday {
		t.Errorf("Did not find backfill record for Monday")
		// Debug info
		for _, r := range mockStorage.Records {
			t.Logf("Record: Date=%s, Time=%d", r.Date, r.TimeDuration)
		}
		t.Logf("Expected Monday Date: %s", mondayDate)
	}
	if !foundToday {
		t.Errorf("Did not find record for Today")
	}
}
