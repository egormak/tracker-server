package services

import "tracker-server/internal/storage"

func (s *Service) StatCompletionTimeDone() ([]storage.TaskRecord, error) {
	return s.st.GetTaskRecordToday()
}
