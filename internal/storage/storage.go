package storage

import "tracker-server/internal/domain/entity"

type Storage interface {
	RestSpend(restTime int) error
	GetRecords() ([]TaskRecord, error)
	GetDayTaskRecord(taskName string) (int, error)
	StatisticRolesGet() ([]RoleRecord, error)
	StatisticRolesGetToday() ([]RoleRecord, error)
	ShowTaskList() ([]TaskResult, error)
	SetTaskParams(params entity.TaskParams) error
	GetTaskParams(taskName string) (entity.TaskParams, error)
	RecheckRole() error
	CleanRecords()
	TimeListSetDB(count int) error
	TimeListDelDB(timeDuretion int) error
	TimeTasks() (int, error)
	TimerGlobalSet(timeScheduler int) error
	TimerGlobalGet() (int, error)
	TimeDurationGet() (int, error)
	AddTaskRecord(task entity.TaskRecord) error
	AddRoleMinutes(task entity.TaskRecord) error
	GetRole(taskName string) (string, error)
	AddRest(restTime int) error
	GetRest() (int, error)
	ProcentsSet(procentM Procents) error
	GetPlanProcents() (Procents, error)
	GetGroupPlanPercent() (int, error)
	ChangeGroupPlanPercent(groupPlan int) error
	GetGroupPercent(groupPlan int) (int, error)
	DelGroupPercent(groupPlan string) error
	RemovePlanPercent(group string, value int) error
	GetTaskNamePlanPercent(groupPlan string, groupPercent int) (string, error)
	CheckIfPlanPercentEmpty() error
	GetGroupName(groupNameOrdinal int) (string, error)
	GetTodayTaskDuration(taskName string) (int, error)
	GetTaskDurationForDate(taskName string, date string) (int, error)
	// GetTaskRecordToday(opts ...TaskRecordOption) ([]TaskRecord, error)
	// WithCheckBusinessDay(check bool) TaskRecordOption
	CreateTask(task entity.TaskDefinition) error
	GetTaskNamesForDate(date string) ([]string, error)
	MoveTaskToPreviousDate(taskName string, currentDate string) error

	// Schedule management
	CreateSchedule(schedule entity.WeeklySchedule) (string, error)
	GetSchedule(id string) (entity.WeeklySchedule, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetAllSchedules() ([]entity.WeeklySchedule, error)
	UpdateSchedule(id string, schedule entity.WeeklySchedule) error
	DeleteSchedule(id string) error
	SetActiveSchedule(id string) error
	GetDaySchedule(day string) (entity.DaySchedule, error)
}
