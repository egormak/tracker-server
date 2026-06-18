package services

import (
	"fmt"
	"sync"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/notify"
)

type RunningTaskStorage interface {
	GetRunningTask(taskName string) (entity.RunningTask, error)
	GetActiveRunningTask() (entity.RunningTask, error)
	GetAllRunningTasks() ([]entity.RunningTask, error)
	UpsertRunningTask(task entity.RunningTask) error
	DeleteRunningTask(taskName string) error
	GetRole(taskName string) (string, error)
	AddTaskRecord(task entity.TaskRecord) error
	AddRoleMinutes(task entity.TaskRecord) error
	AddRest(restTime int) error
	TimeListDelDB(timeDuretion int) error
}

type RunningTaskService struct {
	st RunningTaskStorage
	nt notify.Notify
	mu sync.Mutex
}

func NewRunningTaskService(st RunningTaskStorage, nt notify.Notify) *RunningTaskService {
	return &RunningTaskService{st: st, nt: nt}
}

func (s *RunningTaskService) Start(taskName string, role string, targetDuration int, sourceDay string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if this task already exists
	existing, err := s.st.GetRunningTask(taskName)
	if err == nil && existing.TaskName != "" {
		if existing.IsRunning {
			return existing, nil // Already running
		}
		// Resume paused task
		// First, pause any other currently running task
		active, errActive := s.st.GetActiveRunningTask()
		if errActive == nil && active.TaskName != "" && active.TaskName != taskName {
			active.Accumulated += int(time.Since(active.StartTime).Minutes())
			active.IsRunning = false
			_ = s.st.UpsertRunningTask(active)
		}
		existing.StartTime = time.Now()
		existing.IsRunning = true
		if err := s.st.UpsertRunningTask(existing); err != nil {
			return entity.RunningTask{}, err
		}
		return existing, nil
	}

	// Starting a new task. Pause any currently active running task first.
	active, errActive := s.st.GetActiveRunningTask()
	if errActive == nil && active.TaskName != "" {
		active.Accumulated += int(time.Since(active.StartTime).Minutes())
		active.IsRunning = false
		_ = s.st.UpsertRunningTask(active)
	}

	if role == "" {
		r, err := s.st.GetRole(taskName)
		if err == nil && r != "" {
			role = r
		} else {
			role = "work"
		}
	}

	task := entity.RunningTask{
		TaskName:       taskName,
		Role:           role,
		StartTime:      time.Now(),
		IsRunning:      true,
		TargetDuration: targetDuration,
		SourceDay:      sourceDay,
	}

	if s.nt != nil {
		if msgID, err := s.nt.SendMessageStart(taskName); err == nil {
			task.TelegramMessageID = msgID
		}
	}

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}
	return task, nil
}

func (s *RunningTaskService) Stop(taskName string) (entity.TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var task entity.RunningTask
	var err error

	if taskName != "" {
		task, err = s.st.GetRunningTask(taskName)
	} else {
		task, err = s.st.GetActiveRunningTask()
		if err != nil || task.TaskName == "" {
			tasks, errAll := s.st.GetAllRunningTasks()
			if errAll == nil && len(tasks) > 0 {
				task = tasks[0]
			}
		}
	}

	if err != nil {
		return entity.TaskRecord{}, fmt.Errorf("failed to get task: %w", err)
	}
	if task.TaskName == "" {
		return entity.TaskRecord{}, fmt.Errorf("no running task found")
	}

	// Calculate total duration
	duration := task.Accumulated
	if task.IsRunning {
		duration += int(time.Since(task.StartTime).Minutes())
	}

	if duration == 0 && task.IsRunning {
		duration = 1
	}

	recordDate := time.Now().Format("2 January 2006")
	if task.SourceDay != "" {
		recordDate = CalculateDateForDay(task.SourceDay)
	}

	record := entity.TaskRecord{
		Name:         task.TaskName,
		Role:         task.Role,
		TimeDuration: duration,
		Date:         recordDate,
		SourceDay:    task.SourceDay,
	}

	if err := s.st.AddTaskRecord(record); err != nil {
		return entity.TaskRecord{}, fmt.Errorf("failed to save task record: %w", err)
	}

	if err := s.st.AddRoleMinutes(record); err != nil {
		fmt.Printf("failed to add role minutes: %v\n", err)
	}
	if err := s.st.AddRest(record.TimeDuration); err != nil {
		fmt.Printf("failed to add rest: %v\n", err)
	}

	if s.nt != nil && task.TelegramMessageID != 0 {
		timeEnd := time.Now().Format("2 January 2006 15:04")
		_ = s.nt.SendMessageStop(task.TaskName, record.TimeDuration, task.TelegramMessageID, timeEnd)
	}

	if task.TargetDuration > 0 && record.TimeDuration >= task.TargetDuration {
		_ = s.st.TimeListDelDB(task.TargetDuration)
	}

	if err := s.st.DeleteRunningTask(task.TaskName); err != nil {
		return entity.TaskRecord{}, fmt.Errorf("failed to clear running task: %w", err)
	}

	return record, nil
}

func (s *RunningTaskService) Pause(taskName string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var task entity.RunningTask
	var err error

	if taskName != "" {
		task, err = s.st.GetRunningTask(taskName)
	} else {
		task, err = s.st.GetActiveRunningTask()
	}

	if err != nil {
		return entity.RunningTask{}, fmt.Errorf("failed to get task: %w", err)
	}
	if task.TaskName == "" {
		return entity.RunningTask{}, fmt.Errorf("no running task found")
	}

	if !task.IsRunning {
		return task, nil // Already paused
	}

	task.Accumulated += int(time.Since(task.StartTime).Minutes())
	task.IsRunning = false

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}
	return task, nil
}

func (s *RunningTaskService) Resume(taskName string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var task entity.RunningTask

	if taskName != "" {
		var err error
		task, err = s.st.GetRunningTask(taskName)
		if err != nil {
			return entity.RunningTask{}, err
		}
	} else {
		tasks, errAll := s.st.GetAllRunningTasks()
		if errAll == nil {
			for _, t := range tasks {
				if !t.IsRunning {
					task = t
					break
				}
			}
		}
	}

	if task.TaskName == "" {
		return entity.RunningTask{}, fmt.Errorf("no paused task found to resume")
	}

	if task.IsRunning {
		return task, nil // Already running
	}

	// Before resuming, pause any other active running task
	active, errActive := s.st.GetActiveRunningTask()
	if errActive == nil && active.TaskName != "" && active.TaskName != task.TaskName {
		active.Accumulated += int(time.Since(active.StartTime).Minutes())
		active.IsRunning = false
		_ = s.st.UpsertRunningTask(active)
	}

	task.StartTime = time.Now()
	task.IsRunning = true

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}
	return task, nil
}

func (s *RunningTaskService) GetStatus(taskName string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if taskName != "" {
		return s.st.GetRunningTask(taskName)
	}
	task, err := s.st.GetActiveRunningTask()
	if err == nil && task.TaskName != "" {
		return task, nil
	}
	tasks, errAll := s.st.GetAllRunningTasks()
	if errAll == nil && len(tasks) > 0 {
		return tasks[0], nil
	}
	return entity.RunningTask{}, nil
}

func (s *RunningTaskService) GetAllTasks() ([]entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.GetAllRunningTasks()
}
