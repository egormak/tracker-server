package services

import (
	"errors"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

type mockDigestStorage struct {
	schedule entity.WeeklySchedule
	records  []entity.TaskRecord
}

func (m *mockDigestStorage) ShowTaskList() ([]entity.TaskResult, error) { return nil, nil }

func (m *mockDigestStorage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	return m.schedule, nil
}

// Filters by date like the Mongo query, so a wrong week window shows up as missing records.
func (m *mockDigestStorage) GetRecordsForDates(dates []string) ([]entity.TaskRecord, error) {
	wanted := make(map[string]bool, len(dates))
	for _, d := range dates {
		wanted[d] = true
	}
	var out []entity.TaskRecord
	for _, r := range m.records {
		if wanted[r.Date] {
			out = append(out, r)
		}
	}
	return out, nil
}

const digestDateLayout = "2 January 2006"

// Week of Monday 5 October 2026 .. Sunday 11 October 2026.
func newDigestService(now time.Time) *StatisticService {
	day := func(d int) string { return time.Date(2026, 10, d, 12, 0, 0, 0, time.UTC).Format(digestDateLayout) }
	tasks := []entity.ScheduleTask{
		{Name: "work", Role: "work", Time: 100},
		{Name: "english", Role: "learn", Time: 20},
		{Name: "video", Role: "rest", Time: 5},
		{Name: "games", Role: "rest", Time: 30},
	}
	storage := &mockDigestStorage{
		schedule: entity.WeeklySchedule{
			Monday:  entity.DaySchedule{Tasks: tasks},
			Tuesday: entity.DaySchedule{Tasks: tasks},
		},
		records: []entity.TaskRecord{
			{Name: "work", Role: "work", TimeDuration: 150, Date: day(5)},
			{Name: "work", Role: "work", TimeDuration: 50, Date: day(11)},
			{Name: "video", Role: "rest", TimeDuration: 90, Date: day(6)},
			{Name: "telegram", Role: "rest", TimeDuration: 30, Date: day(6)},
			// The following week: must not leak into the digest of 5..11 October.
			{Name: "games", Role: "rest", TimeDuration: 500, Date: day(12)},
		},
	}
	srv := NewStatisticService(storage)
	srv.nowFunc = func() time.Time { return now }
	return srv
}

func TestWeeklyDigest_PreviousWeekOnMonday(t *testing.T) {
	// Monday morning, before `tracker clean`: the finished week is the previous one.
	srv := newDigestService(time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC))

	digest, err := srv.GetWeeklyDigest(WeekPrevious)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if digest.Week != WeekPrevious || digest.WeekStartDate != "5 October 2026" || digest.WeekEndDate != "11 October 2026" {
		t.Errorf("unexpected week window: %s %s..%s", digest.Week, digest.WeekStartDate, digest.WeekEndDate)
	}
	if digest.TotalDone != 320 {
		t.Errorf("expected 320 minutes done in the previous week, got %d", digest.TotalDone)
	}

	current, err := srv.GetWeeklyDigest("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if current.Week != WeekCurrent || current.WeekStartDate != "12 October 2026" || current.TotalDone != 500 {
		t.Errorf("expected the current week to hold only the new record, got %+v", current)
	}
}

func TestWeeklyDigest_Breakdown(t *testing.T) {
	// Sunday evening of the same week.
	srv := newDigestService(time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC))

	digest, err := srv.GetWeeklyDigest(WeekCurrent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if digest.TotalPlanned != 310 || digest.TotalDone != 320 {
		t.Errorf("expected planned 310 / done 320, got %d / %d", digest.TotalPlanned, digest.TotalDone)
	}

	wantRoles := []entity.WeeklyDigestRole{
		{Role: "work", Planned: 200, Done: 200, SharePercent: 62},
		{Role: "learn", Planned: 40, Done: 0, SharePercent: 0},
		{Role: "rest", Planned: 70, Done: 120, SharePercent: 37},
	}
	if len(digest.Roles) != len(wantRoles) {
		t.Fatalf("expected roles %+v, got %+v", wantRoles, digest.Roles)
	}
	for i, want := range wantRoles {
		if digest.Roles[i] != want {
			t.Errorf("role %d: expected %+v, got %+v", i, want, digest.Roles[i])
		}
	}

	wantStrict := []entity.WeeklyDigestTask{
		{Name: "work", Role: "work", Planned: 200, Done: 200, PlanPercent: 100, SharePercent: 100},
		{Name: "english", Role: "learn", Planned: 40, Done: 0, PlanPercent: 0, SharePercent: 0},
	}
	if len(digest.Strict) != 2 || digest.Strict[0] != wantStrict[0] || digest.Strict[1] != wantStrict[1] {
		t.Errorf("expected strict %+v, got %+v", wantStrict, digest.Strict)
	}

	// Free tasks: most time first; a 5-minute placeholder gives a huge plan percent, the share stays meaningful.
	wantFree := []entity.WeeklyDigestTask{
		{Name: "video", Role: "rest", Planned: 10, Done: 90, PlanPercent: 900, SharePercent: 75},
		{Name: "telegram", Role: "rest", Planned: 0, Done: 30, PlanPercent: 0, SharePercent: 25},
		{Name: "games", Role: "rest", Planned: 60, Done: 0, PlanPercent: 0, SharePercent: 0},
	}
	if digest.FreeDone != 120 || len(digest.Free) != len(wantFree) {
		t.Fatalf("expected free done 120 with %+v, got %d with %+v", wantFree, digest.FreeDone, digest.Free)
	}
	for i, want := range wantFree {
		if digest.Free[i] != want {
			t.Errorf("free %d: expected %+v, got %+v", i, want, digest.Free[i])
		}
	}

	if len(digest.OverPlan) != 1 || digest.OverPlan[0].Name != "video" || digest.OverPlan[0].Delta != 80 {
		t.Errorf("expected over_plan [video +80], got %+v", digest.OverPlan)
	}
	if len(digest.Untouched) != 2 || digest.Untouched[0].Name != "games" || digest.Untouched[1].Name != "english" {
		t.Errorf("expected untouched [games english] by planned time, got %+v", digest.Untouched)
	}
	if len(digest.Unscheduled) != 1 || digest.Unscheduled[0].Name != "telegram" || digest.Unscheduled[0].Done != 30 {
		t.Errorf("expected unscheduled [telegram 30], got %+v", digest.Unscheduled)
	}

	if len(digest.Days) != 7 {
		t.Fatalf("expected 7 days, got %d", len(digest.Days))
	}
	if digest.BusiestDay == nil || digest.BusiestDay.Day != "monday" || digest.BusiestDay.TotalDone != 150 {
		t.Errorf("expected busiest day monday with 150, got %+v", digest.BusiestDay)
	}
}

func TestWeeklyDigest_EmptyWeek(t *testing.T) {
	srv := newDigestService(time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC))

	digest, err := srv.GetWeeklyDigest(WeekCurrent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if digest.TotalDone != 0 || digest.BusiestDay != nil {
		t.Errorf("expected an empty week without a busiest day, got %+v", digest)
	}
	if digest.OverPlan == nil || digest.Unscheduled == nil || digest.Free == nil {
		t.Errorf("lists must be empty arrays, not null, for JSON clients")
	}
}

func TestWeeklyDigest_InvalidWeek(t *testing.T) {
	srv := newDigestService(time.Now())
	if _, err := srv.GetWeeklyDigest("next"); !errors.Is(err, ErrInvalidWeek) {
		t.Errorf("expected ErrInvalidWeek, got %v", err)
	}
}
