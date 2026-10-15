package services

import (
	"reflect"
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

type mockStatStorageWithRecords struct {
	records []entity.TaskRecord
}

func (m *mockStatStorageWithRecords) ShowTaskList() ([]entity.TaskResult, error) {
	return nil, nil
}

func (m *mockStatStorageWithRecords) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return entity.WeeklySchedule{
		Monday: entity.DaySchedule{
			Tasks: []entity.ScheduleTask{
				{Name: "work", Role: "work", Time: 270, Priority: 9},
				{Name: "english", Role: "learn", Time: 20, Priority: 8},
				{Name: "movies", Role: "rest", Time: 295, Priority: 3},
				{Name: "telegram", Role: "rest", Time: 215, Priority: 5},
				{Name: "video", Role: "rest", Time: 245, Priority: 4},
			},
		},
	}, nil
}

func (m *mockStatStorageWithRecords) GetRecordsForDates(dates []string) ([]entity.TaskRecord, error) {
	return m.records, nil
}

type fakeDebtToDate map[string]int

func (f fakeDebtToDate) DebtToDate() (map[string]int, error) { return f, nil }

func eveningRankingFixture() *EveningService {
	now := time.Now()
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6
	}
	mondayDate := now.AddDate(0, 0, -daysSinceMonday).Format("2 January 2006")

	mockStorage := &mockStatStorageWithRecords{
		records: []entity.TaskRecord{
			{Name: "movies", Role: "rest", TimeDuration: 57, Date: mondayDate},
			{Name: "telegram", Role: "rest", TimeDuration: 4, Date: mondayDate},
			{Name: "video", Role: "rest", TimeDuration: 127, Date: mondayDate},
			{Name: "work", Role: "work", TimeDuration: 270, Date: mondayDate},
		},
	}
	return NewEveningService(NewStatisticService(mockStorage))
}

func candidateNames(cs []entity.EveningFocusCandidate) []string {
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = c.TaskName
	}
	return names
}

// The evening picks the task most behind to date: previous days' debt + today's remainder.
func TestEveningServiceRanksByDebtToDate(t *testing.T) {
	eveningSrv := eveningRankingFixture()
	eveningSrv.SetDebtProvider(fakeDebtToDate{"video": 90, "telegram": 30, "movies": 30})

	res, err := eveningSrv.GetEveningFocus("", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CurrentTask.TaskName != "video" || res.CurrentTask.DebtToDate != 90 {
		t.Errorf("expected current task video with debt 90, got %+v", res.CurrentTask)
	}
	if res.CurrentTask.WeeklyDone != 127 || res.CurrentTask.WeeklyTarget != 245 || res.CurrentTask.WeeklyGap != 118 {
		t.Errorf("unexpected weekly figures for video: %+v", res.CurrentTask)
	}
	// telegram and movies tie on debt; the larger weekly gap goes first (movies 238 > telegram 211).
	if got, want := candidateNames(res.Candidates), []string{"video", "movies", "telegram"}; !reflect.DeepEqual(got, want) {
		t.Errorf("expected order %v (work and english excluded), got %v", want, got)
	}
}

// Without debt the weekly gap decides.
func TestEveningServiceRanksByWeeklyGapWithoutDebt(t *testing.T) {
	res, err := eveningRankingFixture().GetEveningFocus("", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := candidateNames(res.Candidates), []string{"movies", "telegram", "video"}; !reflect.DeepEqual(got, want) {
		t.Errorf("expected order %v, got %v", want, got)
	}
}

func TestEveningServiceBreakOptions(t *testing.T) {
	now := time.Now()
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6
	}
	mondayDate := now.AddDate(0, 0, -daysSinceMonday).Format("2 January 2006")

	mockStorage := &mockStatStorageWithRecords{
		records: []entity.TaskRecord{
			{Name: "movies", Role: "rest", TimeDuration: 57, Date: mondayDate},
			{Name: "telegram", Role: "rest", TimeDuration: 4, Date: mondayDate},
			{Name: "video", Role: "rest", TimeDuration: 127, Date: mondayDate},
			{Name: "work", Role: "work", TimeDuration: 270, Date: mondayDate},
		},
	}
	eveningSrv := NewEveningService(NewStatisticService(mockStorage))

	// A task snoozed in evening mode is still a valid break choice.
	if err := eveningSrv.SkipTask("telegram"); err != nil {
		t.Fatalf("unexpected error skipping telegram: %v", err)
	}

	res, err := eveningSrv.GetBreakOptions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var names []string
	for i, opt := range res.Options {
		names = append(names, opt.TaskName)
		if opt.IsVariety != (i == 0) {
			t.Errorf("only the first option must be the variety one, got %+v at %d", opt, i)
		}
	}
	want := []string{"telegram", "movies", "video"}
	if len(names) != len(want) {
		t.Fatalf("expected options %v (work and english excluded), got %v", want, names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("expected options %v ordered by least done, got %v", want, names)
			break
		}
	}
	if res.Options[0].WeeklyDone != 4 {
		t.Errorf("expected weekly_done 4 for telegram, got %d", res.Options[0].WeeklyDone)
	}
	if res.DefaultDuration != 15 || len(res.Durations) != 3 {
		t.Errorf("expected durations [15 20 30] with default 15, got %v / %d", res.Durations, res.DefaultDuration)
	}
}
