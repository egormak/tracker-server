package services

import (
    "tracker-server/internal/domain/entity"
    "tracker-server/internal/storage"
)

type StatisticStorage interface {
    ShowTaskList() ([]storage.TaskResult, error)
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
    out := make([]entity.TaskResult, 0, len(list))
    for _, t := range list {
        out = append(out, entity.TaskResult{
            Name:         t.Name,
            Role:         t.Role,
            TimeDuration: t.TimeDuration,
            TimeDone:     t.TimeDone,
            Priority:     t.Priority,
        })
    }
    return out, nil
}
