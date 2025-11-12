package service

import (
	"log/slog"
	"strings"
	"time"
)

// IsWeekendNow checks if the current day is a weekend day.
// It returns true if the current day is Sunday or Saturday, and false otherwise.
func IsWeekendNow() bool {
	today := time.Now().Weekday().String()
	return today == "Sunday" || today == "Saturday"
}

// CalculateDateForDay calculates the date for a given day of the week in the current week.
// If the specified day is in the future (e.g., asking for Friday on Monday), it returns the date from the previous week.
// Returns the date in "2 January 2006" format, or today's date if the day name is invalid.
func CalculateDateForDay(sourceDay string) string {
	now := time.Now()

	slog.Info("=== CalculateDateForDay START ===",
		"source_day_input", sourceDay,
		"current_date", now.Format("2 January 2006"),
		"current_weekday", now.Weekday().String(),
		"current_weekday_int", int(now.Weekday()))

	// If sourceDay is empty, return today
	if sourceDay == "" {
		slog.Info("Source day is empty, returning today")
		return now.Format("2 January 2006")
	}

	currentWeekday := now.Weekday()

	dayOrder := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
	}

	targetWeekday, ok := dayOrder[strings.ToLower(sourceDay)]
	if !ok {
		// Invalid day name, return today
		slog.Warn("Invalid day name, returning today",
			"source_day", sourceDay,
			"lowercased", strings.ToLower(sourceDay))
		return now.Format("2 January 2006")
	}

	slog.Info("Target weekday found",
		"source_day", sourceDay,
		"target_weekday", targetWeekday.String(),
		"target_weekday_int", int(targetWeekday))

	// Calculate days to subtract from today to get to target day in current/previous week
	// If target is Monday (1) and today is Wednesday (3), daysBack = 3 - 1 = 2 days ago
	daysBack := int(currentWeekday) - int(targetWeekday)
	if daysBack < 0 {
		// Target day is in the previous week (e.g., Friday when today is Monday)
		daysBack += 7
		slog.Info("Days back was negative, adjusted to previous week", "days_back_adjusted", daysBack)
	}

	targetDate := now.AddDate(0, 0, -daysBack)
	formattedDate := targetDate.Format("2 January 2006")

	slog.Info("=== CalculateDateForDay RESULT ===",
		"source_day", sourceDay,
		"days_back", daysBack,
		"target_date", formattedDate,
		"target_weekday", targetDate.Weekday().String())

	return formattedDate
}
