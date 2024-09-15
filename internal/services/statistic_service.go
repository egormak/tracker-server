package services

import "tracker-server/internal/domain/entity"

// func (s *Service) StatCompletionTimeDone() ([]storage.TaskRecord, error) {
// 	return s.srv.GetTaskRecordToday()
// }

func (s *Service) GetTaskRecordToday() ([]entity.TaskResult, error) {
	return []entity.TaskResult{}, nil
}
