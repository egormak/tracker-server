package services

import (
	"errors"
	"testing"
	"tracker-server/internal/domain/entity"
)

type mockStatisticStorage struct {
	tasks      []entity.TaskResult
	schedules  map[string]entity.WeeklySchedule
	records    []entity.TaskRecord
	queryCount int
}

func (m *mockStatisticStorage) ShowTaskList() ([]entity.TaskResult, error) {
	m.queryCount++
	return m.tasks, nil
}

func (m *mockStatisticStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{}, nil
}

func (m *mockStatisticStorage) GetRecordsForDates(dates []string) ([]entity.TaskRecord, error) {
	return m.records, nil
}

type mockTodayScheduleEnsurer struct {
	called    bool
	callCount int
	onEnsure  func() error
}

func (m *mockTodayScheduleEnsurer) EnsureTodaySchedule() error {
	m.called = true
	m.callCount++
	if m.onEnsure != nil {
		return m.onEnsure()
	}
	return nil
}

func TestStatisticService_ShowTaskList_Empty_TriggersEnsureSchedule(t *testing.T) {
	storage := &mockStatisticStorage{
		tasks: []entity.TaskResult{}, // empty initially (e.g. morning before manual apply)
	}

	ensurer := &mockTodayScheduleEnsurer{
		onEnsure: func() error {
			// Populate tasks upon ensure
			storage.tasks = []entity.TaskResult{
				{Name: "Backend Architecture", Role: "work", TimeDuration: 120, TimeDone: 0, Priority: 1},
			}
			return nil
		},
	}

	statService := NewStatisticService(storage, ensurer)

	list, err := statService.ShowTaskList()
	if err != nil {
		t.Fatalf("unexpected error from ShowTaskList: %v", err)
	}

	if !ensurer.called {
		t.Errorf("expected EnsureTodaySchedule to be called when task list is empty")
	}

	if len(list) != 1 || list[0].Name != "Backend Architecture" {
		t.Errorf("expected returned list to contain newly ensured tasks, got: %+v", list)
	}

	// Should have queried twice: first empty, second after ensure
	if storage.queryCount != 2 {
		t.Errorf("expected 2 queries to storage.ShowTaskList, got %d", storage.queryCount)
	}
}

func TestStatisticService_ShowTaskList_NonEmpty_DoesNotTriggerEnsure(t *testing.T) {
	storage := &mockStatisticStorage{
		tasks: []entity.TaskResult{
			{Name: "Existing Task", Role: "work", TimeDuration: 60, TimeDone: 30, Priority: 1},
		},
	}

	ensurer := &mockTodayScheduleEnsurer{}
	statService := NewStatisticService(storage, ensurer)

	list, err := statService.ShowTaskList()
	if err != nil {
		t.Fatalf("unexpected error from ShowTaskList: %v", err)
	}

	if ensurer.called {
		t.Errorf("expected EnsureTodaySchedule NOT to be called when tasks already exist")
	}

	if len(list) != 1 || list[0].Name != "Existing Task" {
		t.Errorf("unexpected task list: %+v", list)
	}

	// Should only query once
	if storage.queryCount != 1 {
		t.Errorf("expected 1 query to storage.ShowTaskList, got %d", storage.queryCount)
	}
}

func TestStatisticService_ShowTaskList_NilEnsurer_Graceful(t *testing.T) {
	storage := &mockStatisticStorage{
		tasks: []entity.TaskResult{},
	}

	statService := NewStatisticService(storage) // no ensurer provided

	list, err := statService.ShowTaskList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected empty list, got %v", list)
	}
}

func TestStatisticService_ShowTaskList_EnsurerError_Graceful(t *testing.T) {
	storage := &mockStatisticStorage{
		tasks: []entity.TaskResult{},
	}

	ensurer := &mockTodayScheduleEnsurer{
		onEnsure: func() error {
			return errors.New("no active schedule configured")
		},
	}

	statService := NewStatisticService(storage, ensurer)

	// Should warn and return the empty list gracefully without blowing up
	list, err := statService.ShowTaskList()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected empty list, got %v", list)
	}
}

func TestStatisticService_GetTaskRecordToday_DelegatesToShowTaskList(t *testing.T) {
	storage := &mockStatisticStorage{
		tasks: []entity.TaskResult{},
	}

	ensurer := &mockTodayScheduleEnsurer{
		onEnsure: func() error {
			storage.tasks = []entity.TaskResult{
				{Name: "Lazy Task", Role: "work", TimeDuration: 90, Priority: 1},
			}
			return nil
		},
	}

	statService := NewStatisticService(storage, ensurer)

	list, err := statService.GetTaskRecordToday()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ensurer.called {
		t.Errorf("expected GetTaskRecordToday to trigger EnsureTodaySchedule via ShowTaskList")
	}

	if len(list) != 1 || list[0].Name != "Lazy Task" {
		t.Errorf("unexpected list: %+v", list)
	}
}
