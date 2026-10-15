package entity

// Debt is the unfinished schedule of the previous days of the current week (Monday up to
// yesterday). It is never capped or forgiven; the user limits it only through the schedule,
// and `tracker clean` wipes it with the week's records.

// DebtItem is one task's shortfall on one previous day.
type DebtItem struct {
	TaskName  string `json:"task_name"`
	Role      string `json:"role"`
	Day       string `json:"day"`  // lowercase weekday, e.g. "tuesday"
	Date      string `json:"date"` // "2 January 2006"
	Remaining int    `json:"remaining"`
}

// Debt day statuses for the week strip.
const (
	DebtDayClosed  = "closed"  // a past day whose schedule is fully done
	DebtDayPartial = "partial" // a past day with some of its schedule done
	DebtDayOpen    = "open"    // a past day with nothing of its schedule done
	DebtDayToday   = "today"
	DebtDayFuture  = "future"
	DebtDayNone    = "none" // nothing scheduled that day
)

// DebtDay is one cell of the week strip.
type DebtDay struct {
	Day       string `json:"day"`
	Date      string `json:"date"`
	Planned   int    `json:"planned"`
	Remaining int    `json:"remaining"`
	Status    string `json:"status"`
}

// DebtSummary is the response of GET /api/v1/debt.
type DebtSummary struct {
	TotalMinutes   int        `json:"total_minutes"`
	Items          []DebtItem `json:"items"`
	OldestDay      string     `json:"oldest_day,omitempty"`
	TodayRemaining int        `json:"today_remaining"`
	Days           []DebtDay  `json:"days"`
}

// Split kinds: where the minutes of a stopped timer went.
const (
	SplitToday    = "today"
	SplitDebt     = "debt"
	SplitTomorrow = "tomorrow"
)

// SplitPart is one destination of a stopped timer's minutes.
type SplitPart struct {
	Kind    string `json:"kind"`
	Day     string `json:"day"`
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
}

// DayClosedEvent is the payload of the DAY_CLOSED realtime event.
type DayClosedEvent struct {
	Day  string `json:"day"`
	Date string `json:"date"`
}
