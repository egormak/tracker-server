package storage

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
	// GetTaskRecordToday(opts ...TaskRecordOption) ([]TaskRecord, error)
	// WithCheckBusinessDay(check bool) TaskRecordOption
}
