package storage

type TaskRecord struct {
	Name         string
	Role         string
	TimeDuration int
	Date         string
}

type TaskConfig struct {
	Name         string
	Role         string
	Date         string
	TimeSchedule int
	Priority     int
}

type TaskParams struct {
	Name     string
	Time     int
	Priority int
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

type Storage interface {
	RestSpend(restTime int) error
	GetRecords() ([]TaskRecord, error)
	GetDayTaskRecord(taskName string) (int, error)
	StatisticRolesGet() ([]RoleRecord, error)
	StatisticRolesGetToday() ([]RoleRecord, error)
	ShowTaskList() ([]TaskResult, error)
	SetTaskParams(params TaskParams) error
	GetTaskParams(taskName string) (TaskParams, error)
	RecheckRole() error
	CleanRecords()
	TimeListSetDB(count int) error
	TimeListDelDB(timeDuretion int) error
	TimeTasks() (int, error)
	TimerGlobalSet(timeScheduler int) error
	TimerGlobalGet() (int, error)
	TimeDurationGet() (int, error)
	AddTaskRecord(task TaskRecord) error
	AddRoleMinutes(task TaskRecord) error
	GetRole(taskName string) (string, error)
	AddRest(restTime int) error
	GetRest() (int, error)
	ProcentsSet(procentM Procents) error
	GetPlanProcents() (Procents, error)
	GetGroupPlanPercent() (int, error)
	ChangeGroupPlanPercent(groupPlan int) error
	GetGroupPercent(groupPlan int) (int, error)
	DelGroupPercent(groupPlan string) error
	GetTaskNamePlanPercent(groupPlan string, groupPercent int) (string, error)
	GetGroupName(groupNameOrdinal int) (string, error)
}
