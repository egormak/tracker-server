package services

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

type fakeDebt struct {
	items     map[string][]entity.DebtItem
	closed    map[string]bool
	dayClosed []string
}

func (f *fakeDebt) TaskDebt(taskName string) ([]entity.DebtItem, error) {
	return f.items[taskName], nil
}

func (f *fakeDebt) DayClosed(day string) (bool, error) {
	f.dayClosed = append(f.dayClosed, day)
	return f.closed[day], nil
}

type splitCase struct {
	task      string
	target    int // today's schedule target; 0 = not scheduled today
	todayDone int
	strict    bool
	share     int
	session   int
	debt      []entity.DebtItem
	sourceDay string
}

func runSplit(t *testing.T, c splitCase) (entity.TaskRecord, *MockRunningTaskStorage, *mockRunningTaskNotify, *fakeDebt) {
	t.Helper()
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	debt := &fakeDebt{items: map[string][]entity.DebtItem{c.task: c.debt}, closed: map[string]bool{}}
	service := NewRunningTaskService(storage, nt)
	service.SetDebtProvider(debt)

	if c.target > 0 {
		today := strings.ToLower(time.Now().Weekday().String())
		var sched entity.WeeklySchedule
		setDaySchedule(&sched, today, entity.DaySchedule{Day: today, Tasks: []entity.ScheduleTask{{Name: c.task, Time: c.target}}})
		storage.activeSchedule = sched
	}
	storage.todayDurations[c.task] = c.todayDone
	storage.strictTasks = map[string]bool{c.task: c.strict}
	storage.tasks[c.task] = entity.RunningTask{
		TaskName:          c.task,
		Role:              "rest",
		StartTime:         time.Now().Add(-time.Duration(c.session)*time.Minute - 5*time.Second),
		IsRunning:         true,
		DebtShare:         c.share,
		SourceDay:         c.sourceDay,
		TelegramMessageID: 77,
	}

	record, err := service.Stop(c.task, "manual")
	if err != nil {
		t.Fatalf("unexpected error stopping task: %v", err)
	}
	return record, storage, nt, debt
}

// minutesByKind sums the split by kind, and the debt part by day.
func minutesByKind(parts []entity.SplitPart) map[string]int {
	out := map[string]int{}
	for _, p := range parts {
		if p.Kind == entity.SplitDebt {
			out["debt:"+p.Day] += p.Minutes
			continue
		}
		out[p.Kind] += p.Minutes
	}
	return out
}

func wednesdayDebt(min int) []entity.DebtItem {
	return []entity.DebtItem{{TaskName: "home_task", Day: "wednesday", Date: "14 October 2026", Remaining: min}}
}

// The examples agreed with the user (decision 0005): home_task, target 40, Wednesday debt 36.
func TestSplitSession_AgreedExamples(t *testing.T) {
	cases := []struct {
		name string
		c    splitCase
		want map[string]int
	}{
		{"no share, overtime pays debt", splitCase{task: "home_task", target: 40, todayDone: 35, session: 20, debt: wednesdayDebt(36)},
			map[string]int{"today": 5, "debt:wednesday": 15}},
		{"share 25 near the target", splitCase{task: "home_task", target: 40, todayDone: 35, share: 25, session: 20, debt: wednesdayDebt(36)},
			map[string]int{"today": 5, "debt:wednesday": 15}},
		{"share 25 early in the day", splitCase{task: "home_task", target: 40, todayDone: 0, share: 25, session: 20, debt: wednesdayDebt(36)},
			map[string]int{"today": 15, "debt:wednesday": 5}},
		{"share larger than the debt returns to today", splitCase{task: "home_task", target: 40, todayDone: 0, share: 25, session: 20, debt: wednesdayDebt(3)},
			map[string]int{"today": 17, "debt:wednesday": 3}},
		{"no debt, non-strict overtime stays today", splitCase{task: "home_task", target: 40, todayDone: 35, session: 20},
			map[string]int{"today": 20}},
		{"no target, share only", splitCase{task: "home_task", share: 50, session: 20, debt: wednesdayDebt(36)},
			map[string]int{"today": 10, "debt:wednesday": 10}},
		{"no target, no share: all today", splitCase{task: "home_task", session: 20, debt: wednesdayDebt(36)},
			map[string]int{"today": 20}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record, storage, _, _ := runSplit(t, tc.c)
			if got := minutesByKind(record.Split); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("split = %v, want %v", got, tc.want)
			}
			if record.TimeDuration != tc.c.session {
				t.Errorf("returned record should carry the whole session %d, got %d", tc.c.session, record.TimeDuration)
			}
			total := 0
			for _, r := range storage.records {
				total += r.TimeDuration
			}
			if total != tc.c.session {
				t.Errorf("stored records sum to %d, want the session %d", total, tc.c.session)
			}
		})
	}
}

// A strict task: the excess first pays its own debt, only the rest goes to tomorrow.
func TestSplitSession_StrictPaysDebtThenTomorrow(t *testing.T) {
	c := splitCase{task: "english", target: 60, todayDone: 55, strict: true, session: 30,
		debt: []entity.DebtItem{{TaskName: "english", Day: "tuesday", Date: "13 October 2026", Remaining: 20}}}
	record, storage, _, _ := runSplit(t, c)
	if got, want := minutesByKind(record.Split), map[string]int{"today": 5, "debt:tuesday": 20, "tomorrow": 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("split = %v, want %v", got, want)
	}
	for _, r := range storage.records {
		if r.TimeDuration == 20 && (r.SourceDay != "tuesday" || r.Date != "13 October 2026") {
			t.Errorf("debt record must be dated to the debt day, got %+v", r)
		}
		if r.TimeDuration == 5 && r.SourceDay == GetTomorrowDayName() && r.Date != CalculateDateForTomorrow() {
			t.Errorf("tomorrow record has the wrong date: %+v", r)
		}
	}
}

// Debt is paid oldest day first, each day at most its shortfall.
func TestSplitSession_OldestDayFirst(t *testing.T) {
	c := splitCase{task: "games", target: 40, todayDone: 40, session: 30, debt: []entity.DebtItem{
		{TaskName: "games", Day: "tuesday", Date: "13 October 2026", Remaining: 10},
		{TaskName: "games", Day: "wednesday", Date: "14 October 2026", Remaining: 36},
	}}
	record, _, _, _ := runSplit(t, c)
	if got, want := minutesByKind(record.Split), map[string]int{"debt:tuesday": 10, "debt:wednesday": 20}; !reflect.DeepEqual(got, want) {
		t.Errorf("split = %v, want %v", got, want)
	}
}

// An explicit backfill goes where it was sent and is never split.
func TestSplitSession_SourceDayIsNotSplit(t *testing.T) {
	c := splitCase{task: "home_task", target: 40, share: 50, session: 20, sourceDay: "monday", debt: wednesdayDebt(36)}
	record, storage, _, debt := runSplit(t, c)
	if len(storage.records) != 1 || storage.records[0].SourceDay != "monday" || storage.records[0].TimeDuration != 20 {
		t.Errorf("expected one 20-min record for monday, got %+v", storage.records)
	}
	if len(record.Split) != 1 || record.Split[0].Kind != entity.SplitToday {
		t.Errorf("expected a single non-debt part, got %+v", record.Split)
	}
	if len(debt.dayClosed) != 0 {
		t.Errorf("no debt day should be checked, got %v", debt.dayClosed)
	}
}

// Paying a day's last minutes closes it: the card says so.
func TestSplitSession_DayClosedNote(t *testing.T) {
	storageCase := splitCase{task: "home_task", target: 40, todayDone: 40, session: 10, debt: wednesdayDebt(10)}
	storage := NewMockRunningTaskStorage()
	nt := &mockRunningTaskNotify{}
	debt := &fakeDebt{items: map[string][]entity.DebtItem{"home_task": storageCase.debt}, closed: map[string]bool{"wednesday": true}}
	service := NewRunningTaskService(storage, nt)
	service.SetDebtProvider(debt)
	today := strings.ToLower(time.Now().Weekday().String())
	var sched entity.WeeklySchedule
	setDaySchedule(&sched, today, entity.DaySchedule{Day: today, Tasks: []entity.ScheduleTask{{Name: "home_task", Time: 40}}})
	storage.activeSchedule = sched
	storage.todayDurations["home_task"] = 40
	storage.tasks["home_task"] = entity.RunningTask{TaskName: "home_task", Role: "rest", StartTime: time.Now().Add(-10*time.Minute - 5*time.Second), IsRunning: true, TelegramMessageID: 5}

	if _, err := service.Stop("home_task", "manual"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nt.completionCalls) != 1 {
		t.Fatalf("expected a completion card, got %d", len(nt.completionCalls))
	}
	notes := strings.Join(nt.completionCalls[0].Notes, "\n")
	if !strings.Contains(notes, "↪️ Ср +10 мин") || !strings.Contains(notes, "🎉 День закрыт ✅ — Среда") {
		t.Errorf("expected debt and day-closed notes, got %q", notes)
	}
}

// The share is stored on a new start and clamped to 0–100.
func TestStartWith_StoresClampedDebtShare(t *testing.T) {
	storage := NewMockRunningTaskStorage()
	service := NewRunningTaskService(storage, nil)
	task, err := service.StartWith(StartOptions{TaskName: "video", Role: "rest", DebtShare: 140})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.DebtShare != 100 {
		t.Errorf("expected share clamped to 100, got %d", task.DebtShare)
	}
}
