package services

import (
	"strings"
	"testing"
	"tracker-server/internal/domain/entity"
)

type MockScheduleStorage struct {
	schedules      map[string]entity.WeeklySchedule
	activeID       string
	records        []entity.TaskRecord
	taskParams     map[string]entity.TaskParams
	todayDurations map[string]int
}

func NewMockScheduleStorage() *MockScheduleStorage {
	return &MockScheduleStorage{
		schedules:      make(map[string]entity.WeeklySchedule),
		taskParams:     make(map[string]entity.TaskParams),
		todayDurations: make(map[string]int),
	}
}

func (m *MockScheduleStorage) CreateSchedule(schedule entity.WeeklySchedule) (string, error) {
	id := "sched_123"
	schedule.ID = id
	m.schedules[id] = schedule
	if schedule.IsActive {
		m.activeID = id
	}
	return id, nil
}

func (m *MockScheduleStorage) GetSchedule(id string) (entity.WeeklySchedule, error) {
	return m.schedules[id], nil
}

func (m *MockScheduleStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return m.schedules[m.activeID], nil
}

func (m *MockScheduleStorage) GetAllSchedules() ([]entity.WeeklySchedule, error) {
	var list []entity.WeeklySchedule
	for _, s := range m.schedules {
		list = append(list, s)
	}
	return list, nil
}

// UpdateSchedule mirrors the real Mongo storage's optimistic-concurrency check: the
// write only applies if schedule.Version still matches the stored version.
func (m *MockScheduleStorage) UpdateSchedule(id string, schedule entity.WeeklySchedule) error {
	existing, ok := m.schedules[id]
	if ok && existing.Version != schedule.Version {
		return entity.ErrScheduleVersionConflict
	}
	schedule.Version++
	m.schedules[id] = schedule
	return nil
}

func (m *MockScheduleStorage) DeleteSchedule(id string) error {
	delete(m.schedules, id)
	return nil
}

func (m *MockScheduleStorage) SetActiveSchedule(id string) error {
	m.activeID = id
	return nil
}

func (m *MockScheduleStorage) GetDaySchedule(day string) (entity.DaySchedule, error) {
	sched := m.schedules[m.activeID]
	switch day {
	case "monday":
		return sched.Monday, nil
	case "tuesday":
		return sched.Tuesday, nil
	case "wednesday":
		return sched.Wednesday, nil
	case "thursday":
		return sched.Thursday, nil
	case "friday":
		return sched.Friday, nil
	case "saturday":
		return sched.Saturday, nil
	case "sunday":
		return sched.Sunday, nil
	default:
		return entity.DaySchedule{}, nil
	}
}

func (m *MockScheduleStorage) GetTodayTaskDuration(taskName string) (int, error) {
	return m.todayDurations[taskName], nil
}

func (m *MockScheduleStorage) GetTaskDurationForDate(taskName string, date string, sourceDay string) (int, error) {
	total := 0
	for _, r := range m.records {
		if r.Name == taskName && (r.Date == date || r.SourceDay == sourceDay) {
			total += r.TimeDuration
		}
	}
	return total, nil
}

// GetRecordsForDatesOrSourceDays mirrors the real Mongo query in
// storage/mongo/statistic.go: a record matches if its source_day is in
// sourceDays, OR its date is in dates AND it has no source_day. A record's
// date field alone is never enough to match once source_day is set.
func (m *MockScheduleStorage) GetRecordsForDatesOrSourceDays(dates []string, sourceDays []string) ([]entity.TaskRecord, error) {
	dateSet := make(map[string]bool)
	for _, d := range dates {
		dateSet[d] = true
	}
	daySet := make(map[string]bool)
	for _, d := range sourceDays {
		daySet[d] = true
	}
	var res []entity.TaskRecord
	for _, r := range m.records {
		if r.SourceDay != "" {
			if daySet[strings.ToLower(r.SourceDay)] {
				res = append(res, r)
			}
		} else if dateSet[r.Date] {
			res = append(res, r)
		}
	}
	return res, nil
}

func (m *MockScheduleStorage) GetTaskParams(taskName string) (entity.TaskParams, error) {
	return m.taskParams[taskName], nil
}

func (m *MockScheduleStorage) CreateTask(taskDefinition entity.TaskDefinition) error {
	return nil
}

func (m *MockScheduleStorage) GetTaskNamesForDate(date string) ([]string, error) {
	return []string{}, nil
}

func (m *MockScheduleStorage) MoveTaskToPreviousDate(taskName string, currentDate string) error {
	return nil
}

func (m *MockScheduleStorage) TimerGlobalSet(timeScheduler int) error {
	return nil
}

// concurrentWriteScheduleStorage wraps MockScheduleStorage to simulate another writer
// completing its own update right after a GetActiveSchedule read, exercising the
// optimistic-concurrency retry loop in ScheduleService.UpdateTaskTime.
type concurrentWriteScheduleStorage struct {
	*MockScheduleStorage
	remainingRaces int
}

func (m *concurrentWriteScheduleStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	sched, err := m.MockScheduleStorage.GetActiveSchedule()
	if err != nil {
		return sched, err
	}
	if m.remainingRaces > 0 {
		m.remainingRaces--
		racing := sched
		racing.Version++
		m.schedules[sched.ID] = racing
	}
	return sched, nil
}

// TestScheduleService_UpdateTaskTime_RetriesOnVersionConflict reproduces two
// near-simultaneous PATCH /api/v1/schedule/active/task-time requests: a concurrent
// write lands between this call's read and write. Without optimistic concurrency,
// the second UpdateSchedule call would silently discard the racing write; the fix
// must detect the conflict and retry against a fresh read instead of losing data.
func TestScheduleService_UpdateTaskTime_RetriesOnVersionConflict(t *testing.T) {
	base := NewMockScheduleStorage()
	sched := entity.WeeklySchedule{
		ID:       "sched_1",
		IsActive: true,
		Version:  1,
		Monday: entity.DaySchedule{
			Day:       "monday",
			TotalTime: 270,
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 1},
			},
		},
	}
	base.schedules["sched_1"] = sched
	base.activeID = "sched_1"

	storage := &concurrentWriteScheduleStorage{MockScheduleStorage: base, remainingRaces: 1}
	service := NewScheduleService(storage)

	min300 := 300
	updated, err := service.UpdateTaskTime(entity.UpdateScheduleTaskTimeRequest{
		TaskName: "work",
		Minutes:  &min300,
		Day:      "monday",
	})
	if err != nil {
		t.Fatalf("expected UpdateTaskTime to retry past the version conflict, got error: %v", err)
	}
	if updated.Monday.Tasks[0].Time != 300 {
		t.Errorf("expected Monday work time 300 after retry, got %d", updated.Monday.Tasks[0].Time)
	}
	if storage.remainingRaces != 0 {
		t.Errorf("expected the simulated concurrent write to have been consumed by a retry")
	}
}

func TestScheduleService_UpdateTaskTime(t *testing.T) {
	storage := NewMockScheduleStorage()
	service := NewScheduleService(storage)

	sched := entity.WeeklySchedule{
		ID:       "sched_1",
		IsActive: true,
		Monday: entity.DaySchedule{
			Day:       "monday",
			TotalTime: 290,
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 1},
				{Name: "english", Role: "learn", Time: 20, Priority: 2},
			},
		},
		Tuesday: entity.DaySchedule{
			Day:       "tuesday",
			TotalTime: 290,
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 1},
				{Name: "english", Role: "learn", Time: 20, Priority: 2},
			},
		},
	}
	storage.schedules["sched_1"] = sched
	storage.activeID = "sched_1"

	// 1. Update work time on Monday only to 300
	min300 := 300
	updated, err := service.UpdateTaskTime(entity.UpdateScheduleTaskTimeRequest{
		TaskName: "work",
		Minutes:  &min300,
		Day:      "monday",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Monday.Tasks[0].Time != 300 {
		t.Errorf("expected Monday work time 300, got %d", updated.Monday.Tasks[0].Time)
	}
	if updated.Monday.TotalTime != 320 {
		t.Errorf("expected Monday TotalTime 320, got %d", updated.Monday.TotalTime)
	}
	if updated.Tuesday.Tasks[0].Time != 270 {
		t.Errorf("expected Tuesday work time 270, got %d", updated.Tuesday.Tasks[0].Time)
	}

	// 2. Delta adjustment on all days: +10 min to english
	delta10 := 10
	updated, err = service.UpdateTaskTime(entity.UpdateScheduleTaskTimeRequest{
		TaskName:     "english",
		DeltaMinutes: &delta10,
		Day:          "all",
	})
	if err != nil {
		t.Fatalf("unexpected error on all days update: %v", err)
	}

	if updated.Monday.Tasks[1].Time != 30 {
		t.Errorf("expected Monday english time 30, got %d", updated.Monday.Tasks[1].Time)
	}
	if updated.Tuesday.Tasks[1].Time != 30 {
		t.Errorf("expected Tuesday english time 30, got %d", updated.Tuesday.Tasks[1].Time)
	}

	// 3. Error when task not found
	min50 := 50
	_, err = service.UpdateTaskTime(entity.UpdateScheduleTaskTimeRequest{
		TaskName: "non_existent_task",
		Minutes:  &min50,
		Day:      "monday",
	})
	if err == nil {
		t.Errorf("expected error updating non-existent task, got nil")
	}
}

func TestScheduleService_GetRolloverTasks_BatchOptimization(t *testing.T) {
	storage := NewMockScheduleStorage()
	service := NewScheduleService(storage)

	sched := entity.WeeklySchedule{
		ID:       "sched_1",
		IsActive: true,
		Monday: entity.DaySchedule{
			Day:       "monday",
			TotalTime: 290,
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 1},
				{Name: "english", Role: "learn", Time: 20, Priority: 2},
			},
		},
		Tuesday: entity.DaySchedule{
			Day:       "tuesday",
			TotalTime: 290,
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 1},
			},
		},
	}
	storage.schedules["sched_1"] = sched
	storage.activeID = "sched_1"

	// Add record: 200 min work done on Monday (deficit = 70 min)
	storage.records = append(storage.records, entity.TaskRecord{
		Name:         "work",
		Role:         "work",
		TimeDuration: 200,
		SourceDay:    "monday",
	})

	// Get rollovers for tuesday
	rollovers, err := service.GetRolloverTasks("tuesday")
	if err != nil {
		t.Fatalf("unexpected error getting rollovers: %v", err)
	}

	if len(rollovers) == 0 {
		t.Fatalf("expected rollover tasks, got 0")
	}

	foundWorkDeficit := false
	foundEnglishDeficit := false
	for _, r := range rollovers {
		if r.TaskName == "work" && r.SourceDay == "monday" {
			if r.RemainingTime != 70 {
				t.Errorf("expected Monday work deficit 70, got %d", r.RemainingTime)
			}
			foundWorkDeficit = true
		}
		if r.TaskName == "english" && r.SourceDay == "monday" {
			if r.RemainingTime != 20 {
				t.Errorf("expected Monday english deficit 20, got %d", r.RemainingTime)
			}
			foundEnglishDeficit = true
		}
	}

	if !foundWorkDeficit {
		t.Errorf("expected Monday work deficit in rollovers")
	}
	if !foundEnglishDeficit {
		t.Errorf("expected Monday english deficit in rollovers")
	}
}

// TestScheduleService_GetRolloverTasks_SourceDayCompletedOnLaterDate reproduces a task
// that rolled over from Monday and was completed on Thursday: the TaskRecord carries
// source_day="monday" but date=Thursday's date, which falls outside Monday's queried
// date window. The batched fetch must still find it via source_day, or the task is
// wrongly reported as a full deficit despite being completed.
func TestScheduleService_GetRolloverTasks_SourceDayCompletedOnLaterDate(t *testing.T) {
	storage := NewMockScheduleStorage()
	service := NewScheduleService(storage)

	sched := entity.WeeklySchedule{
		ID:       "sched_1",
		IsActive: true,
		Monday: entity.DaySchedule{
			Day:       "monday",
			TotalTime: 270,
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 1},
			},
		},
	}
	storage.schedules["sched_1"] = sched
	storage.activeID = "sched_1"

	// Completed on Thursday (a date well outside Monday's queried window), but
	// tagged with source_day="monday" as a rollover backfill record.
	storage.records = append(storage.records, entity.TaskRecord{
		Name:         "work",
		Role:         "work",
		TimeDuration: 270,
		Date:         "9999-thursday-not-in-window",
		SourceDay:    "monday",
	})

	rollovers, err := service.GetRolloverTasks("thursday")
	if err != nil {
		t.Fatalf("unexpected error getting rollovers: %v", err)
	}

	for _, r := range rollovers {
		if r.TaskName == "work" && r.SourceDay == "monday" {
			t.Errorf("expected no Monday work deficit (fully completed via source_day rollover record), got RemainingTime=%d", r.RemainingTime)
		}
	}
}
