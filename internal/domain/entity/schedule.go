package entity

// DaySchedule represents tasks scheduled for a specific day of the week
type DaySchedule struct {
	Day       string         `json:"day" bson:"day"`               // "monday", "tuesday", etc.
	TotalTime int            `json:"total_time" bson:"total_time"` // Total time available for the day in minutes
	Tasks     []ScheduleTask `json:"tasks" bson:"tasks"`           // Tasks scheduled for this day
	PlanGroup []string       `json:"plan_group" bson:"plan_group"` // Order of plan groups: ["plan", "work", "learn", "rest"]
}

// ScheduleTask represents a task configuration for a schedule
type ScheduleTask struct {
	Name     string `json:"name" bson:"name"`
	Role     string `json:"role" bson:"role"`                   // "work", "learn", "rest"
	Time     int    `json:"time" bson:"time"`                   // Time allocated in minutes
	Priority int    `json:"priority" bson:"priority"`           // Priority (higher = more important)
	Percents []int  `json:"percents" bson:"percents,omitempty"` // Percent values for plan rotation
}

// WeeklySchedule represents the full weekly schedule configuration
type WeeklySchedule struct {
	ID        string      `json:"id,omitempty" bson:"_id,omitempty"`
	Title     string      `json:"title" bson:"title"`           // "Weekly Schedule"
	CreatedAt string      `json:"created_at" bson:"created_at"` // Date created
	UpdatedAt string      `json:"updated_at" bson:"updated_at"` // Date last updated
	IsActive  bool        `json:"is_active" bson:"is_active"`   // Whether this schedule is active
	Monday    DaySchedule `json:"monday" bson:"monday"`
	Tuesday   DaySchedule `json:"tuesday" bson:"tuesday"`
	Wednesday DaySchedule `json:"wednesday" bson:"wednesday"`
	Thursday  DaySchedule `json:"thursday" bson:"thursday"`
	Friday    DaySchedule `json:"friday" bson:"friday"`
	Saturday  DaySchedule `json:"saturday" bson:"saturday"`
	Sunday    DaySchedule `json:"sunday" bson:"sunday"`
}

// ScheduleRequest represents the request to create or update a schedule
type ScheduleRequest struct {
	Monday    DaySchedule `json:"monday"`
	Tuesday   DaySchedule `json:"tuesday"`
	Wednesday DaySchedule `json:"wednesday"`
	Thursday  DaySchedule `json:"thursday"`
	Friday    DaySchedule `json:"friday"`
	Saturday  DaySchedule `json:"saturday"`
	Sunday    DaySchedule `json:"sunday"`
}

// ScheduleResponse represents the response containing schedule data
type ScheduleResponse struct {
	Status string          `json:"status"`
	Data   *WeeklySchedule `json:"data,omitempty"`
}

// RolloverTask represents a task that needs to be completed from previous days
type RolloverTask struct {
	TaskName      string `json:"task_name"`
	Role          string `json:"role"`
	Priority      int    `json:"priority"`
	RemainingTime int    `json:"remaining_time"` // Time remaining to complete
	SourceDay     string `json:"source_day"`     // Which day this task was originally scheduled
	Percent       int    `json:"percent"`        // The percent allocation it belongs to
}

// ActiveSchedule represents the current day's active schedule with rollovers
type ActiveSchedule struct {
	Day           string         `json:"day"`
	TotalTime     int            `json:"total_time"`
	Tasks         []ScheduleTask `json:"tasks"`          // Today's scheduled tasks
	RolloverTasks []RolloverTask `json:"rollover_tasks"` // Incomplete tasks from previous days
	PlanGroup     []string       `json:"plan_group"`
}
