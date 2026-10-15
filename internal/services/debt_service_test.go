package services

import (
	"strings"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

// debtFixture schedules games 40 + video 20 on every day. Monday is fully done, Tuesday
// half done, the other past days untouched. Needs today to be Wednesday or later.
func debtFixture(t *testing.T) (*DebtService, int) {
	t.Helper()
	todayOrd := dayOrder[strings.ToLower(time.Now().Weekday().String())]
	if todayOrd < 2 {
		t.Skip("needs two past days this week (runs Wednesday to Sunday)")
	}
	day := entity.DaySchedule{Tasks: []entity.ScheduleTask{
		{Name: "games", Role: "rest", Time: 40, Priority: 2},
		{Name: "video", Role: "rest", Time: 20, Priority: 4},
	}}
	sched := entity.WeeklySchedule{IsActive: true}
	for _, d := range weekDayNames {
		setDaySchedule(&sched, d, day)
	}
	st := NewMockScheduleStorage()
	if _, err := st.CreateSchedule(sched); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	monday := now.AddDate(0, 0, -todayOrd).Format("2 January 2006")
	tuesday := now.AddDate(0, 0, 1-todayOrd).Format("2 January 2006")
	st.records = []entity.TaskRecord{
		{Name: "games", TimeDuration: 40, Date: monday},
		{Name: "video", TimeDuration: 20, Date: monday},
		{Name: "games", TimeDuration: 30, Date: tuesday},
	}
	return NewDebtService(NewScheduleService(st)), todayOrd
}

func TestDebtSummary_WeekStripAndItems(t *testing.T) {
	srv, todayOrd := debtFixture(t)
	sum, err := srv.Summary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sum.Days) != 7 {
		t.Fatalf("expected 7 days, got %d", len(sum.Days))
	}
	if sum.Days[0].Status != entity.DebtDayClosed {
		t.Errorf("monday: expected closed, got %+v", sum.Days[0])
	}
	if sum.Days[1].Status != entity.DebtDayPartial || sum.Days[1].Remaining != 30 {
		t.Errorf("tuesday: expected partial with 30 left, got %+v", sum.Days[1])
	}
	for ord := 2; ord < todayOrd; ord++ {
		if sum.Days[ord].Status != entity.DebtDayOpen {
			t.Errorf("%s: expected open, got %+v", weekDayNames[ord], sum.Days[ord])
		}
	}
	if sum.Days[todayOrd].Status != entity.DebtDayToday {
		t.Errorf("today: expected today, got %+v", sum.Days[todayOrd])
	}
	for ord := todayOrd + 1; ord < 7; ord++ {
		if sum.Days[ord].Status != entity.DebtDayFuture {
			t.Errorf("%s: expected future, got %+v", weekDayNames[ord], sum.Days[ord])
		}
	}

	// Tuesday: games 10 + video 20; each later past day: 60.
	wantTotal := 30 + 60*(todayOrd-2)
	if sum.TotalMinutes != wantTotal {
		t.Errorf("expected total %d, got %d", wantTotal, sum.TotalMinutes)
	}
	if sum.OldestDay != "tuesday" {
		t.Errorf("expected oldest day tuesday, got %q", sum.OldestDay)
	}
	if sum.TodayRemaining != 60 {
		t.Errorf("expected today's remainder 60, got %d", sum.TodayRemaining)
	}
	for _, it := range sum.Items {
		if it.Day == strings.ToLower(time.Now().Weekday().String()) {
			t.Errorf("today must not be part of the debt items: %+v", it)
		}
	}
}

func TestDebtService_TaskDebtAndDayClosed(t *testing.T) {
	srv, _ := debtFixture(t)
	items, err := srv.TaskDebt("games")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || items[0].Day != "tuesday" || items[0].Remaining != 10 {
		t.Errorf("expected games debt starting with tuesday 10, got %+v", items)
	}
	if closed, _ := srv.DayClosed("monday"); !closed {
		t.Errorf("monday should be closed")
	}
	if closed, _ := srv.DayClosed("tuesday"); closed {
		t.Errorf("tuesday should not be closed")
	}

	toDate, err := srv.DebtToDate()
	if err != nil {
		t.Fatal(err)
	}
	if toDate["games"] <= toDate["video"] {
		t.Errorf("games should be more behind than video to date, got %v", toDate)
	}
}
