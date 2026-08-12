package entity

type TaskRecord struct {
	Name         string
	Role         string
	TimeDuration int
	Date         string
	SourceDay    string
}

type TaskRecordRequest struct {
	TaskName        string `json:"task_name"`
	TimeDone        int    `json:"time_done"`
	SourceDay       string `json:"source_day,omitempty"`        // Optional: "monday", "tuesday", etc. If empty, uses today
	ManageByService bool   `json:"manage_by_service,omitempty"` // If true, distributes time to past unfilled schedules
}

type TaskResult struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	TimeDuration int    `json:"time_duration"`
	TimeDone     int    `json:"time_done"`
	Priority     int    `json:"priority"`
}

type TaskDefinition struct {
	Name         string `bson:"name" json:"name"`
	Role         string `bson:"role" json:"role"`
	TimeSchedule int    `bson:"timeschedule" json:"timeschedule"`
	Priority     int    `bson:"priority" json:"priority"`
	Date         string `bson:"date" json:"date"`
	TimeStrictly bool   `bson:"timestrictly" json:"timestrictly"`
}

type TaskParams struct {
	Name     string
	Time     int
	Priority int
}

// WebResponse is a struct for response
type PlanPercentResponse struct {
	TaskName  string `json:"task_name"`
	Percent   int    `json:"percent"`
	TimeLeft  int    `json:"time_left"`
	SourceDay string `json:"source_day,omitempty"` // Optional: which day this task is from (for rollover tasks)
}

type WeeklyStatsDay struct {
	Day       string         `json:"day"`
	Date      string         `json:"date"`
	TotalDone int            `json:"total_done"`
	Roles     map[string]int `json:"roles"`
	Tasks     []TaskResult   `json:"tasks"`
}

type WeeklyTarget struct {
	Role         string `json:"role"`
	TimeDuration int    `json:"time_duration"`
	TimeDone     int    `json:"time_done"`
}

type WeeklyTaskTarget struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	TimeDuration int    `json:"time_duration"`
	TimeDone     int    `json:"time_done"`
}

type WeeklyStatsResponse struct {
	WeekStartDate string             `json:"week_start_date"`
	WeekEndDate   string             `json:"week_end_date"`
	Days          []WeeklyStatsDay   `json:"days"`
	WeeklyTargets []WeeklyTarget     `json:"weekly_targets"`
	WeeklyTasks   []WeeklyTaskTarget `json:"weekly_tasks"`
}
