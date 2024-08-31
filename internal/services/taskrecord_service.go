package services

import (
	"fmt"
	"log/slog"
	"time"
	"tracker-server/internal/domain/entity"
)

type Storage interface {
	GetRole(taskName string) (string, error)
	AddTaskRecord(task entity.TaskRecord) error
	AddRoleMinutes(task entity.TaskRecord) error
	AddRest(restTime int) error
}

type TaskRecordService struct {
	st Storage
}

func NewTaskRecordService(st Storage) *TaskRecordService {
	return &TaskRecordService{st: st}
}

func (s *TaskRecordService) AddRecord(taskRecord entity.TaskRecordRequest) error {

	taskRole, err := s.st.GetRole(taskRecord.TaskName)
	if err != nil {
		errMsg := fmt.Errorf("task role can't get: %s", err)
		slog.Error("task_record_service, add_record:get_role", "err", errMsg)
		return errMsg
	}

	record := entity.TaskRecord{
		Name:         taskRecord.TaskName,
		Role:         taskRole,
		TimeDuration: taskRecord.TimeDone,
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
