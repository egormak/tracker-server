package services

import (
	"fmt"
	"log/slog"
	"time"
	"tracker-server/internal/domain/entity"
)

type TaskRecordStorage interface {
	GetRole(taskName string) (string, error)
	AddTaskRecord(task entity.TaskRecord) error
	AddRoleMinutes(task entity.TaskRecord) error
	AddRest(restTime int) error
	GetGroupPlanPercent() (int, error)
	GetGroupPercent(groupPlanOrdinal int) (int, error)
	CheckIfPlanPercentEmpty() error
	ChangeGroupPlanPercent(groupPlanOrdinal int) error
	GetGroupName(groupPlanOrdinal int) (string, error)
	GetTaskNamePlanPercent(groupName string, groupPercent int) (string, error)
	DelGroupPercent(groupName string) error
	GetTodayTaskDuration(taskName string) (int, error)
	GetTaskParams(taskName string) (entity.TaskParams, error)
}

type TaskRecordService struct {
	st TaskRecordStorage
}

func NewTaskRecordService(st TaskRecordStorage) *TaskRecordService {
	return &TaskRecordService{st: st}
}

func (s *TaskRecordService) AddRecord(taskRecordRequest entity.TaskRecordRequest) error {

	taskRole, err := s.st.GetRole(taskRecordRequest.TaskName)
	if err != nil {
		errMsg := fmt.Errorf("task role can't get: %s", err)
		slog.Error("task_record_service, add_record:get_role", "err", errMsg)
		return errMsg
	}

	record := entity.TaskRecord{
		Name:         taskRecordRequest.TaskName,
		Role:         taskRole,
		TimeDuration: taskRecordRequest.TimeDone,
		Date:         time.Now().Format("2 January 2006"),
	}

	if err := s.st.AddTaskRecord(record); err != nil {
		errMsg := fmt.Errorf("can't add record: %s", err)
		slog.Error("task_record_service, add_record:add_task_record", "err", errMsg)
		return errMsg
	}

	if err := s.st.AddRoleMinutes(record); err != nil {
		errMsg := fmt.Errorf("can't add role minutes: %s", err)
		slog.Error("task_record_service, add_record:add_role_minutes", "err", errMsg)
		return errMsg
	}

	if err := s.st.AddRest(record.TimeDuration); err != nil {
		errMsg := fmt.Errorf("can't add rest: %s", err)
		slog.Error("task_record_service, add_record:add_rest", "err", errMsg)
		return errMsg
	}

	return nil
}

// func (s *TaskRecordService) GetTasksNext() (entity.TasksNextResponse, error) {
// 	return entity.TasksNextResponse{}, nil
// }

// GetTodayTaskTimeLeft calculates the remaining time for a task for today
func (s *TaskRecordService) GetTodayTaskTimeLeft(taskName string) (int, error) {
	// Retrieve today's task duration
	timeDuration, err := s.st.GetTodayTaskDuration(taskName)
	if err != nil {
		errMsg := fmt.Errorf("unable to retrieve today's task duration: %s", err)
		slog.Error("task_record_service, get_today_task_left:get_today_task_duration", "err", errMsg)
		return 0, errMsg
	}

	// Retrieve task parameters
	taskParams, err := s.st.GetTaskParams(taskName)
	if err != nil {
		errMsg := fmt.Errorf("unable to retrieve task parameters: %s", err)
		slog.Error("task_record_service, get_today_task_left:get_task_params", "err", errMsg)
		return 0, errMsg
	}

	// Calculate the remaining time for the task
	taskTimeLeft := taskParams.Time - timeDuration

	return taskTimeLeft, nil
}
