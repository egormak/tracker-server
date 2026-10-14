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

// WeeklyDigestClient is the time logged through one client (the task record's
// source_device); "unknown" collects records with no client name.
type WeeklyDigestClient struct {
	Client       string `json:"client"`
	Records      int    `json:"records"`
	Minutes      int    `json:"minutes"`
	SharePercent int    `json:"share_percent"`
}

type WeeklyDigestUsageClient struct {
	Client string `json:"client"`
	Count  int    `json:"count"`
}

// WeeklyDigestUsage is how many times one kind of choice was made during the week.
type WeeklyDigestUsage struct {
	Event   string                    `json:"event"`
	Count   int                       `json:"count"`
	Clients []WeeklyDigestUsageClient `json:"clients"`
}

// Digest sources: computed from the live records, or the snapshot taken before the wipe.
const (
	DigestSourceLive     = "live"
	DigestSourceSnapshot = "snapshot"
)

// WeeklyDigest is the end-of-week debrief: what the week was spent on and where
// the choices diverged from the schedule. It is computed on request; the only stored
// copy is the single snapshot taken right before the weekly wipe.
type WeeklyDigest struct {
	Week string `json:"week"` // "current" or "previous"
	// Source is "snapshot" when the week's records are already wiped and the digest
	// comes from the copy taken before the wipe (SnapshotTakenAt, RFC 3339).
	Source          string `json:"source"`
	SnapshotTakenAt string `json:"snapshot_taken_at,omitempty"`
	WeekStartDate   string `json:"week_start_date"`
	WeekEndDate     string `json:"week_end_date"`
	TotalPlanned    int    `json:"total_planned"`
	TotalDone       int    `json:"total_done"`

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

	// Clients is the logged time by client, most time first; Usage is the week's
	// choice counters in a fixed event order (events never counted are omitted).
	Clients []WeeklyDigestClient `json:"clients"`
	Usage   []WeeklyDigestUsage  `json:"usage"`
}
