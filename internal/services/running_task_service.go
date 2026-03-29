package services

import (
	"fmt"
	"time"
	"tracker-server/internal/domain/entity"
)

type RunningTaskStorage interface {
	GetRunningTask() (entity.RunningTask, error)
	UpsertRunningTask(task entity.RunningTask) error
	DeleteRunningTask() error
	GetRole(taskName string) (string, error)
	AddTaskRecord(task entity.TaskRecord) error
	AddRoleMinutes(task entity.TaskRecord) error
	AddRest(restTime int) error
}

type RunningTaskService struct {
	st RunningTaskStorage
}

func NewRunningTaskService(st RunningTaskStorage) *RunningTaskService {
	return &RunningTaskService{st: st}
}

func (s *RunningTaskService) Start(taskName string, role string, targetDuration int, sourceDay string) (entity.RunningTask, error) {
	// Check if already running
	existing, err := s.st.GetRunningTask()
	if err == nil && existing.IsRunning {
		// Stop existing first? Or just error?
		// Let's implicitly stop and save the previous one to be user friendly, or just error.
		// For simplicity, let's error and tell user to stop previous task.
		return entity.RunningTask{}, fmt.Errorf("task '%s' is already running", existing.TaskName)
	}

	if role == "" {
		// Try to lookup role
		r, err := s.st.GetRole(taskName)
		if err == nil && r != "" {
			role = r
		} else {
			role = "work" // Default fallback
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

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}
	return task, nil
}

func (s *RunningTaskService) Stop() (entity.TaskRecord, error) {
	task, err := s.st.GetRunningTask()
	if err != nil {
		return entity.TaskRecord{}, err
	}
	if task.TaskName == "" {
		return entity.TaskRecord{}, fmt.Errorf("no running task found")
	}

	// Calculate total duration
	duration := task.Accumulated
	if task.IsRunning {
		duration += int(time.Since(task.StartTime).Minutes())
	}

	// Minimum 1 minute if it was running for less
	if duration == 0 && task.IsRunning {
		duration = 1
	}

	record := entity.TaskRecord{
		Name:         task.TaskName,
		Role:         task.Role,
		TimeDuration: duration,
		Date:         time.Now().Format("2 January 2006"),
		SourceDay:    task.SourceDay, // Apply the source day when stopping
	}

	// Save Record
	if err := s.st.AddTaskRecord(record); err != nil {
		return entity.TaskRecord{}, fmt.Errorf("failed to save task record: %w", err)
	}

	// Side effects (roles, rest) - similar to AddRecord
	if err := s.st.AddRoleMinutes(record); err != nil {
		// Log error but don't fail the whole stop operation
		fmt.Printf("failed to add role minutes: %v\n", err)
	}
	if err := s.st.AddRest(record.TimeDuration); err != nil {
		fmt.Printf("failed to add rest: %v\n", err)
	}

	// Delete running task
	if err := s.st.DeleteRunningTask(); err != nil {
		return entity.TaskRecord{}, fmt.Errorf("failed to clear running task: %w", err)
	}

	return record, nil
}

func (s *RunningTaskService) Pause() (entity.RunningTask, error) {
	task, err := s.st.GetRunningTask()
	if err != nil {
		return entity.RunningTask{}, err
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

func (s *RunningTaskService) Resume() (entity.RunningTask, error) {
	task, err := s.st.GetRunningTask()
	if err != nil {
		return entity.RunningTask{}, err
	}
	if task.IsRunning {
		return task, nil // Already running
	}

	task.StartTime = time.Now()
	task.IsRunning = true

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}
	return task, nil
}

func (s *RunningTaskService) GetStatus() (entity.RunningTask, error) {
	// On get status, we might want to return a "computed" duration valid for NOW
	// But the entity itself just stores the state. Handler can compute for frontend.
	return s.st.GetRunningTask()
}
