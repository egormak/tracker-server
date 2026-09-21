package mongo

import (
	"reflect"
	"testing"
	"time"
)

func TestGetWeekDates(t *testing.T) {
	// Thursday, 17 September 2026
	dates := getWeekDates("17 September 2026")
	expected := []string{
		"14 September 2026", // Monday
		"15 September 2026", // Tuesday
		"16 September 2026", // Wednesday
		"17 September 2026", // Thursday
		"18 September 2026", // Friday
		"19 September 2026", // Saturday
		"20 September 2026", // Sunday
	}

	if !reflect.DeepEqual(dates, expected) {
		t.Fatalf("getWeekDates expected %v, got %v", expected, dates)
	}

	// Sunday, 20 September 2026 (boundary case: Sunday is last day of the Monday-Sunday week)
	datesSunday := getWeekDates("20 September 2026")
	if !reflect.DeepEqual(datesSunday, expected) {
		t.Fatalf("getWeekDates for Sunday expected %v, got %v", expected, datesSunday)
	}

	// Monday, 14 September 2026 (boundary case: Monday is first day of the week)
	datesMonday := getWeekDates("14 September 2026")
	if !reflect.DeepEqual(datesMonday, expected) {
		t.Fatalf("getWeekDates for Monday expected %v, got %v", expected, datesMonday)
	}

	// Fallback for unparseable date
	fallback := getWeekDates("invalid-date-format")
	if len(fallback) != 1 || fallback[0] != "invalid-date-format" {
		t.Fatalf("expected fallback [invalid-date-format], got %v", fallback)
	}
}

func TestDateBoundaryExcludesHistoricalRollovers(t *testing.T) {
	// Querying for Monday, 14 September 2026 with sourceDay="monday"
	weekDates := getWeekDates("14 September 2026")
	weekDatesSet := make(map[string]bool)
	for _, d := range weekDates {
		weekDatesSet[d] = true
	}

	// Record 1: Rollover completed within the same week (e.g. Thursday 17 September)
	withinWeekRecordDate := "17 September 2026"
	if !weekDatesSet[withinWeekRecordDate] {
		t.Errorf("expected within-week record date '%s' to be included in week dates", withinWeekRecordDate)
	}

	// Record 2: Stale rollover from a prior week (e.g. Monday 7 September)
	priorWeekRecordDate := "7 September 2026"
	if weekDatesSet[priorWeekRecordDate] {
		t.Errorf("CRITICAL BUG: prior week record date '%s' should be excluded from week dates to prevent unbounded over-crediting", priorWeekRecordDate)
	}

	// Record 3: Stale rollover from 1 year ago
	oldYearRecordDate := time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC).Format("2 January 2006")
	if weekDatesSet[oldYearRecordDate] {
		t.Errorf("CRITICAL BUG: historical year record date '%s' should be excluded", oldYearRecordDate)
	}
}
