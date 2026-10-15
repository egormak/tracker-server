package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	GetTaskDurationForDate(taskName string, date string, sourceDay string) (int, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetTaskParams(taskName string) (entity.TaskParams, error)
	IsTaskStrict(taskName string) (bool, error)
}

type BreakOptionsProvider interface {
	GetBreakOptions() (entity.BreakOptionsResponse, error)
}

// minRecordedRun is the shortest stopped run that becomes a task record.
const minRecordedRun = 30 * time.Second

type RunningTaskService struct {
	st           RunningTaskStorage
	nt           notify.Notify
	hub          *realtime.Hub
	rampService  RampRecalculator
	breakOptions BreakOptionsProvider
	usage        *UsageService
	mu           sync.Mutex
}

// StartOptions describes a start request. Client and Origin are optional: Client is the
// calling client's name, Origin says how the task was chosen (OriginBreak, OriginEvening,
// OriginCombo; anything else counts as a manual start).
type StartOptions struct {
	TaskName       string
	Role           string
	TargetDuration int
	SourceDay      string
	Client         string
	Origin         string
}

func NewRunningTaskService(st RunningTaskStorage, nt notify.Notify) *RunningTaskService {
	return &RunningTaskService{st: st, nt: nt}
}

func (s *RunningTaskService) SetHub(hub *realtime.Hub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hub = hub
}

// SetBreakOptionsProvider enables the break selector on the completion card.
func (s *RunningTaskService) SetBreakOptionsProvider(p BreakOptionsProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.breakOptions = p
}

// SetUsageService enables weekly usage counters for task starts.
func (s *RunningTaskService) SetUsageService(u *UsageService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage = u
}

func (s *RunningTaskService) SetRampService(r RampRecalculator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rampService = r
}

func (s *RunningTaskService) Start(taskName string, role string, targetDuration int, sourceDay string) (entity.RunningTask, error) {
	return s.StartWith(StartOptions{TaskName: taskName, Role: role, TargetDuration: targetDuration, SourceDay: sourceDay})
}

// startEvent maps a start origin to its usage event. A break start is "variety" when the
// task is the one the break selector currently puts first, otherwise "mood".
func (s *RunningTaskService) startEvent(origin, taskName string) string {
	switch origin {
	case OriginEvening:
		return UsageStartEvening
	case OriginCombo:
		return UsageStartCombo
	case OriginBreak:
		if s.breakOptions != nil {
			if resp, err := s.breakOptions.GetBreakOptions(); err == nil {
				for _, opt := range resp.Options {
					if opt.IsVariety && strings.EqualFold(opt.TaskName, taskName) {
						return UsageStartBreakVariety
					}
				}
			}
		}
		return UsageStartBreakMood
	}
	return UsageStartManual
}

// StartWith starts a task. The client and the origin are recorded only for a newly
// started task, not when it is already running or resumed from a pause.
func (s *RunningTaskService) StartWith(opts StartOptions) (entity.RunningTask, error) {
	taskName, role, targetDuration, sourceDay := opts.TaskName, opts.Role, opts.TargetDuration, opts.SourceDay

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// If this exact task is already running, return it immediately
	existing, err := s.st.GetRunningTask(taskName)
	if err == nil && existing.TaskName != "" && existing.IsRunning {
		return existing, nil // Already running
	}

	// Seamless 1-click task switching: if activeTask is running and activeTask.TaskName != taskName,
	// auto-stop the running task with reason "switched" before starting taskName!
	active, errActive := s.st.GetActiveRunningTask()
	if errActive == nil && active.TaskName != "" && active.TaskName != taskName && active.IsRunning {
		if _, err := s.stopLocked(active.TaskName, "switched"); err != nil {
			slog.Error("RunningTaskService: failed to auto-stop running task during seamless switch", "task", active.TaskName, "error", err)
			return entity.RunningTask{}, fmt.Errorf("failed to auto-stop task '%s' for seamless switch: %w", active.TaskName, err)
		}
	}

	// Resume existing paused task if it already exists
	if err == nil && existing.TaskName != "" {
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
		SourceDevice:    opts.Client,
	}

	// Classified before the task exists: the break selector ranks by logged time.
	usageEvent := s.startEvent(opts.Origin, taskName)

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
	s.usage.Track(usageEvent, opts.Client)

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
	return s.stopLocked(taskName, reasons...)
}

func (s *RunningTaskService) stopLocked(taskName string, reasons ...string) (entity.TaskRecord, error) {
	reason := "manual"
	if len(reasons) > 0 && reasons[0] != "" {
		reason = reasons[0]
	}

	task, err := s.resolveTaskOrActive(taskName)
	if err != nil {
		return entity.TaskRecord{}, fmt.Errorf("failed to get task: %w", err)
	}
	if task.TaskName == "" {
		return entity.TaskRecord{}, fmt.Errorf("no running task found")
	}

	// Calculate total duration
	elapsed := time.Duration(task.Accumulated) * time.Minute
	if task.IsRunning {
		elapsed += time.Since(task.StartTime)
	}
	duration := int(elapsed.Minutes())

	// A started minute counts as one, but a run shorter than minRecordedRun is a
	// mis-start (e.g. a stop key that also reached the next timer) and leaves no record.
	if duration == 0 && task.IsRunning && elapsed >= minRecordedRun {
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
						CreatedAt:    time.Now().UTC(),
						SourceDevice: task.SourceDevice,
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
		CreatedAt:    time.Now().UTC(),
		SourceDevice: task.SourceDevice,
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
		if s.rampService != nil {
			if err := s.rampService.AutoRecalculateOnRecord(context.Background(), record.Name, record.TimeDuration); err != nil {
				slog.Error("running_task_service, stopLocked:auto_recalculate_ramp", "err", err)
			}
		}
	}

	if s.nt != nil && task.TelegramMessageID != 0 && duration == 0 {
		// Nothing was recorded: drop the start card instead of reporting a 0-minute task.
		if err := s.nt.DeleteMessage(task.TelegramMessageID); err != nil {
			slog.Warn("RunningTaskService: failed to delete start card", "task", task.TaskName, "error", err)
		}
	} else if s.nt != nil && task.TelegramMessageID != 0 {
		// A backfilled session (SourceDay set) is recorded against that day, so its
		// progress and target must come from that day rather than today.
		todayDone := record.TimeDuration
		doneFn := func() (int, error) { return s.st.GetTodayTaskDuration(task.TaskName) }
		if task.SourceDay != "" {
			doneFn = func() (int, error) { return s.st.GetTaskDurationForDate(task.TaskName, recordDate, task.SourceDay) }
		}
		if td, err := doneFn(); err == nil && td > 0 {
			todayDone = td
		}

		// todayDone covers the whole day for this task, so the target is the task's day
		// target: the schedule (below), else its params time. Never task.TargetDuration —
		// that is the length of this one timer run.
		targetDuration := 0

		var remainingTasksList []string
		var nextTask string

		if sched, err := s.st.GetActiveSchedule(); err == nil {
			todayWeekday := strings.ToLower(time.Now().Weekday().String())
			daySchedule := getDayScheduleFromWeekly(sched, todayWeekday)

			targetSchedule := daySchedule
			if task.SourceDay != "" {
				targetSchedule = getDayScheduleFromWeekly(sched, strings.ToLower(task.SourceDay))
			}
			for _, st := range targetSchedule.Tasks {
				if strings.EqualFold(st.Name, task.TaskName) && st.Time > 0 {
					targetDuration = st.Time
					break
				}
			}

			seen := make(map[string]bool)
			for _, st := range daySchedule.Tasks {
				if strings.EqualFold(st.Name, task.TaskName) {
					continue
				}
				if seen[st.Name] {
					continue
				}
				done, _ := s.st.GetTodayTaskDuration(st.Name)
				target := st.Time
				if target == 0 {
					if p, err := s.st.GetTaskParams(st.Name); err == nil && p.Time > 0 {
						target = p.Time
					}
				}
				if done < target {
					seen[st.Name] = true
					remainingTasksList = append(remainingTasksList, st.Name)
				}
			}

			if len(remainingTasksList) > 0 {
				nextTask = remainingTasksList[0]
			}
		}
		if targetDuration == 0 {
			if params, err := s.st.GetTaskParams(task.TaskName); err == nil && params.Time > 0 {
				targetDuration = params.Time
			}
		}

		// The break selector follows a focus block only; after a rest task the card stays as is.
		var breakOptions []entity.BreakOption
		if s.breakOptions != nil && (strings.EqualFold(task.Role, "work") || strings.EqualFold(task.Role, "learn")) {
			if res, err := s.breakOptions.GetBreakOptions(); err != nil {
				slog.Warn("RunningTaskService: failed to get break options", "task", task.TaskName, "error", err)
			} else {
				breakOptions = res.Options
			}
		}

		if err := s.nt.SendMessageCompletion(task.TaskName, record.TimeDuration, todayDone, targetDuration, remainingTasksList, nextTask, task.TelegramMessageID, breakOptions); err != nil {
			slog.Error("RunningTaskService: failed to send telegram completion message", "task", task.TaskName, "error", err)
		}
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
		if err != nil {
			return entity.RunningTask{}, fmt.Errorf("failed to get task: %w", err)
		}
		if task.TaskName == "" {
			return entity.RunningTask{}, fmt.Errorf("task %q not found or not running", taskName)
		}
	} else {
		task, err = s.st.GetActiveRunningTask()
		if err != nil {
			return entity.RunningTask{}, fmt.Errorf("failed to get task: %w", err)
		}
		if task.TaskName == "" {
			return entity.RunningTask{}, fmt.Errorf("no running task found")
		}
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

// resolveTaskOrActive resolves the task a caller means by name, falling back to
// the currently active running task, and finally to any running task at all.
// Shared by Stop and Adjust so their fallback order can't drift apart.
func (s *RunningTaskService) resolveTaskOrActive(taskName string) (entity.RunningTask, error) {
	if taskName != "" {
		task, err := s.st.GetRunningTask(taskName)
		if err != nil {
			return entity.RunningTask{}, fmt.Errorf("failed to get task: %w", err)
		}
		if task.TaskName == "" {
			return entity.RunningTask{}, fmt.Errorf("task %q not found or not running", taskName)
		}
		return task, nil
	}

	task, err := s.st.GetActiveRunningTask()
	if err != nil || task.TaskName == "" {
		tasks, errAll := s.st.GetAllRunningTasks()
		if errAll == nil && len(tasks) > 0 {
			return tasks[0], nil
		}
	}
	return task, err
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

func (s *RunningTaskService) Adjust(taskName string, deltaMinutes int) (entity.RunningTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, err := s.resolveTaskOrActive(taskName)
	if err != nil {
		return entity.RunningTask{}, fmt.Errorf("failed to get task: %w", err)
	}
	if task.TaskName == "" {
		return entity.RunningTask{}, fmt.Errorf("no running task found to adjust")
	}

	newTarget := task.TargetDuration + deltaMinutes
	if newTarget < 0 {
		newTarget = 0
	}
	task.TargetDuration = newTarget

	now := time.Now()
	if task.IsRunning && task.TargetDuration > 0 {
		elapsedMin := task.Accumulated + int(now.Sub(task.StartTime).Minutes())
		remainingMin := task.TargetDuration - elapsedMin
		if remainingMin > 0 {
			task.DeadlineAt = now.Add(time.Duration(remainingMin) * time.Minute)
		}
		// remainingMin <= 0 (already overtime): leave DeadlineAt untouched, matching
		// Start's and Resume's identical remaining-time recompute.
	} else {
		task.DeadlineAt = time.Time{}
	}

	if err := s.st.UpsertRunningTask(task); err != nil {
		return entity.RunningTask{}, err
	}

	s.broadcastEvent(realtime.Event{
		Type:     realtime.EventTaskAdjusted,
		TaskName: task.TaskName,
		Role:     task.Role,
		Duration: task.TargetDuration,
		Reason:   fmt.Sprintf("adjusted_%d", deltaMinutes),
		Data:     task,
	})

	return task, nil
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
