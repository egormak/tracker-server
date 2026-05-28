package services

import (
	"tracker-server/internal/domain/entity"
)

type StatisticStorage interface {
	ShowTaskList() ([]entity.TaskResult, error)
}

type StatisticService struct {
	st StatisticStorage
}

func NewStatisticService(st StatisticStorage) *StatisticService {
	return &StatisticService{st: st}
}

// GetTaskRecordToday returns today's tasks with planned (time_duration) and done (time_done)
func (s *StatisticService) GetTaskRecordToday() ([]entity.TaskResult, error) {
	list, err := s.st.ShowTaskList()
	if err != nil {
		return nil, err
	}
	return list, nil
}
