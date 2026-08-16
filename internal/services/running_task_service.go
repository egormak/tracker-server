package services

import (
	"fmt"
	"sync"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/notify"
	"tracker-server/internal/realtime"
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
	GetTodayTaskDuration(taskName string) (int, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetTaskParams(taskName string) (entity.TaskParams, error)
	IsTaskStrict(taskName string) (bool, error)
}

type RunningTaskService struct {
	st  RunningTaskStorage
	nt  notify.Notify
	hub *realtime.Hub
	mu  sync.Mutex
}

func NewRunningTaskService(st RunningTaskStorage, nt notify.Notify) *RunningTaskService {
	return &RunningTaskService{st: st, nt: nt}
}

func (s *RunningTaskService) SetHub(hub *realtime.Hub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hub = hub
}

func (s *RunningTaskService) Start(taskName string, role string, targetDuration int, sourceDay string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

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
			active.Accumulated += int(now.Sub(active.StartTime).Minutes())
			active.IsRunning = false
			active.DeadlineAt = time.Time{}
			_ = s.st.UpsertRunningTask(active)
			s.broadcastEvent(realtime.Event{
				Type:     realtime.EventTaskPaused,
				TaskName: active.TaskName,
				Role:     active.Role,
				Reason:   "switch_task",
				Data:     active,
			})
		}
		existing.StartTime = now
		existing.LastHeartbeatAt = now
		existing.IsRunning = true
		if existing.TargetDuration > 0 {
			remainingMin := existing.TargetDuration - existing.Accumulated
			if remainingMin > 0 {
				existing.DeadlineAt = now.Add(time.Duration(remainingMin) * time.Minute)
			}
		}
		if err := s.st.UpsertRunningTask(existing); err != nil {
			return entity.RunningTask{}, err
		}
		s.broadcastEvent(realtime.Event{
			Type:     realtime.EventTaskResumed,
			TaskName: existing.TaskName,
			Role:     existing.Role,
			Duration: existing.TargetDuration,
			Data:     existing,
		})
		return existing, nil
	}

	// Starting a new task. Pause any currently active running task first.
	active, errActive := s.st.GetActiveRunningTask()
	if errActive == nil && active.TaskName != "" {
		active.Accumulated += int(now.Sub(active.StartTime).Minutes())
		active.IsRunning = false
		active.DeadlineAt = time.Time{}
		_ = s.st.UpsertRunningTask(active)
		s.broadcastEvent(realtime.Event{
			Type:     realtime.EventTaskPaused,
			TaskName: active.TaskName,
			Role:     active.Role,
			Reason:   "switch_task",
			Data:     active,
		})
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
		TaskName:        taskName,
		Role:            role,
		StartTime:       now,
		LastHeartbeatAt: now,
		IsRunning:       true,
		TargetDuration:  targetDuration,
		SourceDay:       sourceDay,
	}

	if targetDuration > 0 {
		task.DeadlineAt = now.Add(time.Duration(targetDuration) * time.Minute)
	}

	if s.nt != nil {
		if msgID, err := s.nt.SendMessageStart(taskName); err == nil {
			task.TelegramMessageID = msgID
		}
	}

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}

	s.broadcastEvent(realtime.Event{
		Type:     realtime.EventTaskStarted,
		TaskName: task.TaskName,
		Role:     task.Role,
		Duration: task.TargetDuration,
		Data:     task,
	})

	return task, nil
}

func (s *RunningTaskService) Stop(taskName string, reasons ...string) (entity.TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reason := "manual"
	if len(reasons) > 0 && reasons[0] != "" {
		reason = reasons[0]
	}

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
				err = nil
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

	// Check if this task is strict and whether it exceeds today's schedule target time
	isStrict, _ := s.st.IsTaskStrict(task.TaskName)
	if isStrict && task.SourceDay == "" {
		targetMin := GetScheduledTargetTime(s.st, task.TaskName)
		if targetMin > 0 {
			todayDone, _ := s.st.GetTodayTaskDuration(task.TaskName)
			if todayDone+duration > targetMin {
				todayAllowed := targetMin - todayDone
				if todayAllowed < 0 {
					todayAllowed = 0
				}

				overtime := duration - todayAllowed
				if overtime > 0 {
					duration = todayAllowed

					// Create credit record for TOMORROW
					tomorrowDay := GetTomorrowDayName()
					tomorrowDate := CalculateDateForTomorrow()
					overtimeRecord := entity.TaskRecord{
						Name:         task.TaskName,
						Role:         task.Role,
						TimeDuration: overtime,
						Date:         tomorrowDate,
						SourceDay:    tomorrowDay,
					}

					if err := s.st.AddTaskRecord(overtimeRecord); err == nil {
						fmt.Printf("🎉 STRICT TASK OVERTIME CREDITED TO TOMORROW (from timer): %d min to %s (%s)\n", overtime, tomorrowDay, tomorrowDate)
						_ = s.st.AddRoleMinutes(overtimeRecord)
						_ = s.st.AddRest(overtime)
					}
				}
			}
		}
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

	if duration > 0 {
		if err := s.st.AddTaskRecord(record); err != nil {
			return entity.TaskRecord{}, fmt.Errorf("failed to save task record: %w", err)
		}

		if err := s.st.AddRoleMinutes(record); err != nil {
			fmt.Printf("failed to add role minutes: %v\n", err)
		}
		if err := s.st.AddRest(record.TimeDuration); err != nil {
			fmt.Printf("failed to add rest: %v\n", err)
		}
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

	s.broadcastEvent(realtime.Event{
		Type:     realtime.EventTaskStopped,
		TaskName: task.TaskName,
		Role:     task.Role,
		Duration: record.TimeDuration,
		Reason:   reason,
		Data:     record,
	})

	return record, nil
}

func (s *RunningTaskService) Pause(taskName string, reasons ...string) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reason := "manual"
	if len(reasons) > 0 && reasons[0] != "" {
		reason = reasons[0]
	}

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
	task.DeadlineAt = time.Time{}

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}

	s.broadcastEvent(realtime.Event{
		Type:     realtime.EventTaskPaused,
		TaskName: task.TaskName,
		Role:     task.Role,
		Reason:   reason,
		Data:     task,
	})

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

	now := time.Now()

	// Before resuming, pause any other active running task
	active, errActive := s.st.GetActiveRunningTask()
	if errActive == nil && active.TaskName != "" && active.TaskName != task.TaskName {
		active.Accumulated += int(now.Sub(active.StartTime).Minutes())
		active.IsRunning = false
		active.DeadlineAt = time.Time{}
		_ = s.st.UpsertRunningTask(active)
		s.broadcastEvent(realtime.Event{
			Type:     realtime.EventTaskPaused,
			TaskName: active.TaskName,
			Role:     active.Role,
			Reason:   "switch_task",
			Data:     active,
		})
	}

	task.StartTime = now
	task.LastHeartbeatAt = now
	task.IsRunning = true

	if task.TargetDuration > 0 {
		remainingMin := task.TargetDuration - task.Accumulated
		if remainingMin > 0 {
			task.DeadlineAt = now.Add(time.Duration(remainingMin) * time.Minute)
		}
	}

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}

	s.broadcastEvent(realtime.Event{
		Type:     realtime.EventTaskResumed,
		TaskName: task.TaskName,
		Role:     task.Role,
		Duration: task.TargetDuration,
		Data:     task,
	})

	return task, nil
}

func (s *RunningTaskService) Heartbeat(taskName string) (entity.RunningTask, error) {
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
		return entity.RunningTask{}, nil // No active task
	}

	task.LastHeartbeatAt = time.Now()
	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}

	s.broadcastEvent(realtime.Event{
		Type:     realtime.EventHeartbeatAck,
		TaskName: task.TaskName,
		Role:     task.Role,
		Data:     task,
	})

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

func (s *RunningTaskService) broadcastEvent(event realtime.Event) {
	if s.hub != nil {
		s.hub.Broadcast(event)
	}
}
