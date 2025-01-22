package services

import "tracker-server/internal/domain/entity"

type StatisticStorage interface {
}

type StatisticService struct {
	st StatisticStorage
}

func NewStatisticService(st StatisticStorage) *StatisticService {
	return &StatisticService{st: st}
}

// func (s *Service) StatCompletionTimeDone() ([]storage.TaskRecord, error) {
// 	return s.srv.GetTaskRecordToday()
// }

func (s *StatisticService) GetTaskRecordToday() ([]entity.TaskResult, error) {
	return []entity.TaskResult{}, nil
}
