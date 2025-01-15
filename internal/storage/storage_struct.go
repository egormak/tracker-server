package storage

type TaskRecord struct {
	Name         string
	Role         string
	TimeDuration int
	Date         string
}

type TaskRecordOptions struct {
	CheckBusinessDay bool
}

type TaskRecordOption func(*TaskRecordOptions)

type TaskConfig struct {
	Name         string
	Role         string
	Date         string
	TimeSchedule int
	Priority     int
}

type TaskResult struct {
	Name         string `json:"name"`
	Role         string `json:"role"`
	TimeDuration int    `json:"time_duration"`
	TimeDone     int    `json:"time_done"`
	Priority     int    `json:"priority"`
}

type RoleRecord struct {
	Name          string
	Duration      int
	RecordDate    string
	DurationToday int
}

type DayList struct {
	Title    string
	Count    int
	ListTime []int
}

type Procents struct {
	Title         string
	Date          string
	CurrentChoice int
	Plans         []string
	Plan          []int
	Work          []int
	Learn         []int
	Rest          []int
}

type SchedulerInfo struct {
	Name        string
	Date        string
	ScheduleAll int
}

// type StatisticRecord struct {
// 	CompletionTime    int
// 	CompletionPercent int
// }
