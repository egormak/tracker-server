package service

import "time"

// IsWeekendNow checks if the current day is a weekend day.
// It returns true if the current day is Sunday or Saturday, and false otherwise.
func IsWeekendNow() bool {
	today := time.Now().Weekday().String()
	return today == "Sunday" || today == "Saturday"
}
