package services

import "tracker-server/internal/domain/entity"

type TaskStorage interface {
	GetTaskParams(taskName string) (entity.TaskParams, error)
}

type TaskNotify interface {
	SendMessageStart(taskName string) (int, error)
	SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error
}

type TaskService struct {
	st TaskStorage
	nt TaskNotify
}

func NewTaskService(st TaskStorage, nt TaskNotify) *TaskService {
	return &TaskService{st: st, nt: nt}
}

func (t *TaskService) GetTaskParams(taskName string) (entity.TaskParams, error) {
	return t.st.GetTaskParams(taskName)
}
