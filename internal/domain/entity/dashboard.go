package entity

type DashboardStateResponse struct {
	RunningTask   *RunningTask          `json:"running_task"`
	TodayTasks    []TaskResult          `json:"today_tasks"`
	RestPool      int                   `json:"rest_pool"`
	EveningFocus  *EveningFocusResponse `json:"evening_focus,omitempty"`
	TotalPlanned  int                   `json:"total_planned"`
	TotalDone     int                   `json:"total_done"`
	CompletionPct int                   `json:"completion_pct"`
}
