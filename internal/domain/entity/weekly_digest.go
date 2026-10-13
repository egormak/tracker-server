package entity

// WeeklyDigestRole is a role's weekly total. SharePercent is its share of all logged time.
type WeeklyDigestRole struct {
	Role         string `json:"role"`
	Planned      int    `json:"planned"`
	Done         int    `json:"done"`
	SharePercent int    `json:"share_percent"`
}

// WeeklyDigestTask is a task's weekly total. PlanPercent is done/planned (0 when nothing
// was planned); SharePercent is the task's share of the time logged in its group.
type WeeklyDigestTask struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	Planned      int    `json:"planned"`
	Done         int    `json:"done"`
	PlanPercent  int    `json:"plan_percent"`
	SharePercent int    `json:"share_percent"`
}

// WeeklyDigestDelta is a task whose logged time diverged from the schedule (Delta = done - planned).
type WeeklyDigestDelta struct {
	Name    string `json:"name"`
	Role    string `json:"role"`
	Planned int    `json:"planned"`
	Done    int    `json:"done"`
	Delta   int    `json:"delta"`
}

type WeeklyDigestDay struct {
	Day       string `json:"day"`
	Date      string `json:"date"`
	TotalDone int    `json:"total_done"`
}

// WeeklyDigest is the end-of-week debrief: what the week was spent on and where
// the choices diverged from the schedule. Nothing in it is stored.
type WeeklyDigest struct {
	Week          string `json:"week"` // "current" or "previous"
	WeekStartDate string `json:"week_start_date"`
	WeekEndDate   string `json:"week_end_date"`
	TotalPlanned  int    `json:"total_planned"`
	TotalDone     int    `json:"total_done"`

	Roles []WeeklyDigestRole `json:"roles"`
	// Strict are the schedule-driven tasks (work, english); Free is everything else, most time first.
	Strict   []WeeklyDigestTask `json:"strict"`
	Free     []WeeklyDigestTask `json:"free"`
	FreeDone int                `json:"free_done"`

	// Choice vs. schedule: done above the plan, planned but never started, done with no plan.
	OverPlan    []WeeklyDigestDelta `json:"over_plan"`
	Untouched   []WeeklyDigestDelta `json:"untouched"`
	Unscheduled []WeeklyDigestDelta `json:"unscheduled"`

	Days       []WeeklyDigestDay `json:"days"`
	BusiestDay *WeeklyDigestDay  `json:"busiest_day,omitempty"`
}
