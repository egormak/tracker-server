package entity

type TaskRecord struct {
	Name         string
	Role         string
	TimeDuration int
	Date         string
}

type TaskRecordRequest struct {
	TaskName  string `json:"task_name"`
	TimeDone  int    `json:"time_done"`
	SourceDay string `json:"source_day,omitempty"` // Optional: "monday", "tuesday", etc. If empty, uses today
}

type TaskResult struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	TimeDuration int    `json:"time_duration"`
	TimeDone     int    `json:"time_done"`
	Priority     int    `json:"priority"`
}

type TaskDefinition struct {
	Name         string
	Role         string
	TimeSchedule int
	Priority     int
	Date         string
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
