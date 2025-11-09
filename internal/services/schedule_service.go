package services

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"tracker-server/internal/domain/entity"
)

// Day of week ordering for rollover calculations
var dayOrder = map[string]int{
	"monday":    0,
	"tuesday":   1,
	"wednesday": 2,
	"thursday":  3,
	"friday":    4,
	"saturday":  5,
	"sunday":    6,
}

// taskAggregate tracks scheduled vs completed time for a task across multiple days
type taskAggregate struct {
	totalScheduled int
	totalDone      int
	role           string
	priority       int
	percent        int
	oldestDay      string
}

// ScheduleStorage defines the storage interface for schedule operations
type ScheduleStorage interface {
	CreateSchedule(schedule entity.WeeklySchedule) (string, error)
	GetSchedule(id string) (entity.WeeklySchedule, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetAllSchedules() ([]entity.WeeklySchedule, error)
	UpdateSchedule(id string, schedule entity.WeeklySchedule) error
	DeleteSchedule(id string) error
	SetActiveSchedule(id string) error
	GetDaySchedule(day string) (entity.DaySchedule, error)

	// Task operations needed for rollover calculations
	GetTodayTaskDuration(taskName string) (int, error)
	GetTaskDurationForDate(taskName string, date string) (int, error)
	GetTaskParams(taskName string) (entity.TaskParams, error)
	CreateTask(taskDefinition entity.TaskDefinition) error
}

// ScheduleService handles business logic for schedule management
type ScheduleService struct {
	st ScheduleStorage
}

// NewScheduleService creates a new instance of ScheduleService
func NewScheduleService(st ScheduleStorage) *ScheduleService {
	return &ScheduleService{st: st}
}

// CreateSchedule creates a new weekly schedule
func (s *ScheduleService) CreateSchedule(request entity.ScheduleRequest, setActive bool) (string, error) {
	// Validate the schedule
	if err := s.validateScheduleRequest(request); err != nil {
		return "", fmt.Errorf("invalid schedule: %w", err)
	}

	schedule := entity.WeeklySchedule{
		Title:     "Weekly Schedule",
		IsActive:  setActive,
		Monday:    request.Monday,
		Tuesday:   request.Tuesday,
		Wednesday: request.Wednesday,
		Thursday:  request.Thursday,
		Friday:    request.Friday,
		Saturday:  request.Saturday,
		Sunday:    request.Sunday,
	}

	id, err := s.st.CreateSchedule(schedule)
	if err != nil {
		return "", fmt.Errorf("failed to create schedule: %w", err)
	}

	slog.Info("Schedule created", "id", id, "active", setActive)
	return id, nil
}

// GetActiveSchedule retrieves the currently active schedule
func (s *ScheduleService) GetActiveSchedule() (*entity.WeeklySchedule, error) {
	schedule, err := s.st.GetActiveSchedule()
	if err != nil {
		return nil, fmt.Errorf("failed to get active schedule: %w", err)
	}

	return &schedule, nil
}

// GetSchedule retrieves a specific schedule by ID
func (s *ScheduleService) GetSchedule(id string) (*entity.WeeklySchedule, error) {
	schedule, err := s.st.GetSchedule(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// UpdateSchedule updates an existing schedule
func (s *ScheduleService) UpdateSchedule(id string, request entity.ScheduleRequest) error {
	// Validate the schedule
	if err := s.validateScheduleRequest(request); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	// Get existing schedule to preserve metadata
	existing, err := s.st.GetSchedule(id)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	// Update schedule with new data
	existing.Monday = request.Monday
	existing.Tuesday = request.Tuesday
	existing.Wednesday = request.Wednesday
	existing.Thursday = request.Thursday
	existing.Friday = request.Friday
	existing.Saturday = request.Saturday
	existing.Sunday = request.Sunday

	if err := s.st.UpdateSchedule(id, existing); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	slog.Info("Schedule updated", "id", id)
	return nil
}

// DeleteSchedule deletes a schedule
func (s *ScheduleService) DeleteSchedule(id string) error {
	if err := s.st.DeleteSchedule(id); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	slog.Info("Schedule deleted", "id", id)
	return nil
}

// SetActiveSchedule activates a specific schedule
func (s *ScheduleService) SetActiveSchedule(id string) error {
	if err := s.st.SetActiveSchedule(id); err != nil {
		return fmt.Errorf("failed to activate schedule: %w", err)
	}

	slog.Info("Schedule activated", "id", id)
	return nil
}

// GetTodaySchedule returns today's schedule with rollover tasks
func (s *ScheduleService) GetTodaySchedule() (*entity.ActiveSchedule, error) {
	today := strings.ToLower(time.Now().Weekday().String())

	// Get today's base schedule
	daySchedule, err := s.st.GetDaySchedule(today)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's schedule: %w", err)
	}

	// Get rollover tasks from previous days
	rollovers, err := s.GetRolloverTasks(today)
	if err != nil {
		slog.Warn("Failed to get rollover tasks", "error", err)
		// Continue without rollovers rather than failing
		rollovers = []entity.RolloverTask{}
	}

	activeSchedule := &entity.ActiveSchedule{
		Day:           today,
		TotalTime:     daySchedule.TotalTime,
		Tasks:         daySchedule.Tasks,
		RolloverTasks: rollovers,
		PlanGroup:     daySchedule.PlanGroup,
	}

	return activeSchedule, nil
}

// GetRolloverTasks calculates incomplete tasks from previous days
// It aggregates by task name: total scheduled - total done across all previous days
func (s *ScheduleService) GetRolloverTasks(currentDay string) ([]entity.RolloverTask, error) {
	schedule, err := s.st.GetActiveSchedule()
	if err != nil {
		return nil, fmt.Errorf("no active schedule: %w", err)
	}

	currentDayOrdinal := dayOrder[strings.ToLower(currentDay)]
	previousDays := s.getPreviousDays(currentDay)

	// Build task aggregates from all previous days
	taskMap := s.buildTaskAggregates(schedule, previousDays, currentDayOrdinal)

	// Convert aggregates to rollover list
	return s.buildRolloverList(taskMap), nil
}

// buildTaskAggregates processes all previous days and builds task aggregation map
func (s *ScheduleService) buildTaskAggregates(schedule entity.WeeklySchedule, previousDays []string, currentDayOrdinal int) map[string]*taskAggregate {
	taskMap := make(map[string]*taskAggregate)

	// First pass: collect scheduled tasks from all days to initialize taskMap
	for _, day := range previousDays {
		daySchedule := s.getDayScheduleFromWeekly(schedule, day)

		for _, task := range daySchedule.Tasks {
			if agg, exists := taskMap[task.Name]; exists {
				// Task seen before, add scheduled time
				agg.totalScheduled += task.Time
			} else {
				// First time seeing this task
				percent := 100
				if len(task.Percents) > 0 {
					percent = task.Percents[0]
				}

				taskMap[task.Name] = &taskAggregate{
					totalScheduled: task.Time,
					totalDone:      0, // Will be calculated in second pass
					role:           task.Role,
					priority:       task.Priority,
					percent:        percent,
					oldestDay:      day,
				}
			}
		}
	}

	// Second pass: check actual completed work for ALL tasks on ALL previous days
	// This handles both scheduled and unscheduled days uniformly
	for _, day := range previousDays {
		dayDate := s.getDateForDay(day, currentDayOrdinal)

		for taskName, agg := range taskMap {
			timeDone, err := s.st.GetTaskDurationForDate(taskName, dayDate)
			if err != nil {
				// Not an error - task might not have been worked on this day
				continue
			}

			if timeDone > 0 {
				agg.totalDone += timeDone
				slog.Info("Task work found",
					"task", taskName,
					"day", day,
					"date", dayDate,
					"done", timeDone,
					"running_total_done", agg.totalDone)
			}
		}
	}

	return taskMap
}

// buildRolloverList converts task aggregates into rollover task list
func (s *ScheduleService) buildRolloverList(taskMap map[string]*taskAggregate) []entity.RolloverTask {
	var rollovers []entity.RolloverTask

	for taskName, agg := range taskMap {
		deficit := agg.totalScheduled - agg.totalDone
		if deficit > 0 {
			rollovers = append(rollovers, entity.RolloverTask{
				TaskName:      taskName,
				Role:          agg.role,
				Priority:      agg.priority,
				RemainingTime: deficit,
				SourceDay:     agg.oldestDay,
				Percent:       agg.percent,
			})

			slog.Info("Task deficit calculated",
				"task", taskName,
				"total_scheduled", agg.totalScheduled,
				"total_done", agg.totalDone,
				"deficit", deficit)
		}
	}

	slog.Info("Calculated rollover tasks", "count", len(rollovers))
	return rollovers
}

// ApplyScheduleToday creates TaskDefinitions for today based on the active schedule
// It accumulates incomplete time from previous days for recurring tasks
func (s *ScheduleService) ApplyScheduleToday() error {
	today := strings.ToLower(time.Now().Weekday().String())
	todayDate := time.Now().Format("2 January 2006")

	daySchedule, err := s.st.GetDaySchedule(today)
	if err != nil {
		return fmt.Errorf("failed to get today's schedule: %w", err)
	}

	// Get rollovers and build lookup maps
	rolloverMap, rolloverTaskInfo := s.buildRolloverMaps(today)

	// Apply scheduled tasks with rollovers
	processedTasks := s.applyScheduledTasks(daySchedule, todayDate, rolloverMap)

	// Apply rollover-only tasks
	s.applyRolloverOnlyTasks(todayDate, rolloverMap, rolloverTaskInfo, processedTasks)

	slog.Info("Applied schedule for today", "day", today, "scheduled_tasks", len(daySchedule.Tasks), "rollover_tasks", len(rolloverMap))
	return nil
}

// buildRolloverMaps creates lookup maps for rollover tasks
func (s *ScheduleService) buildRolloverMaps(today string) (map[string]int, map[string]entity.RolloverTask) {
	rollovers, err := s.GetRolloverTasks(today)
	if err != nil {
		slog.Warn("Failed to get rollover tasks", "error", err)
		return make(map[string]int), make(map[string]entity.RolloverTask)
	}

	rolloverMap := make(map[string]int)
	rolloverTaskInfo := make(map[string]entity.RolloverTask)

	for _, rollover := range rollovers {
		rolloverMap[rollover.TaskName] += rollover.RemainingTime
		if _, exists := rolloverTaskInfo[rollover.TaskName]; !exists {
			rolloverTaskInfo[rollover.TaskName] = rollover
		}
	}

	return rolloverMap, rolloverTaskInfo
}

// applyScheduledTasks creates task definitions for scheduled tasks (with rollovers if any)
func (s *ScheduleService) applyScheduledTasks(daySchedule entity.DaySchedule, todayDate string, rolloverMap map[string]int) map[string]bool {
	processedTasks := make(map[string]bool)

	for _, scheduleTask := range daySchedule.Tasks {
		processedTasks[scheduleTask.Name] = true
		totalTime := scheduleTask.Time

		if deficit, exists := rolloverMap[scheduleTask.Name]; exists {
			totalTime += deficit
			slog.Info("Adding rollover time to task",
				"task", scheduleTask.Name,
				"today_scheduled", scheduleTask.Time,
				"rollover_deficit", deficit,
				"total_time", totalTime)
		}

		taskDef := entity.TaskDefinition{
			Name:         scheduleTask.Name,
			Role:         scheduleTask.Role,
			TimeSchedule: totalTime,
			Priority:     scheduleTask.Priority,
			Date:         todayDate,
		}

		if err := s.st.CreateTask(taskDef); err != nil {
			slog.Error("Failed to create/update task from schedule", "task", scheduleTask.Name, "error", err)
			continue
		}

		slog.Info("Applied task from schedule", "task", scheduleTask.Name, "time", totalTime, "priority", scheduleTask.Priority)
	}

	return processedTasks
}

// applyRolloverOnlyTasks creates task definitions for rollover tasks not in today's schedule
func (s *ScheduleService) applyRolloverOnlyTasks(todayDate string, rolloverMap map[string]int, rolloverTaskInfo map[string]entity.RolloverTask, processedTasks map[string]bool) {
	for taskName, totalDeficit := range rolloverMap {
		if processedTasks[taskName] {
			continue
		}

		rolloverInfo := rolloverTaskInfo[taskName]

		slog.Info("Creating task from rollover only (not in today's schedule)",
			"task", taskName,
			"rollover_deficit", totalDeficit,
			"role", rolloverInfo.Role,
			"priority", rolloverInfo.Priority)

		taskDef := entity.TaskDefinition{
			Name:         taskName,
			Role:         rolloverInfo.Role,
			TimeSchedule: totalDeficit,
			Priority:     rolloverInfo.Priority,
			Date:         todayDate,
		}

		if err := s.st.CreateTask(taskDef); err != nil {
			slog.Error("Failed to create rollover task", "task", taskName, "error", err)
			continue
		}

		slog.Info("Applied rollover task", "task", taskName, "time", totalDeficit, "priority", rolloverInfo.Priority)
	}
}

// Helper: validateScheduleRequest validates a schedule request
func (s *ScheduleService) validateScheduleRequest(request entity.ScheduleRequest) error {
	days := []entity.DaySchedule{
		request.Monday, request.Tuesday, request.Wednesday,
		request.Thursday, request.Friday, request.Saturday, request.Sunday,
	}

	for _, day := range days {
		if day.TotalTime < 0 {
			return fmt.Errorf("total time cannot be negative for %s", day.Day)
		}

		for _, task := range day.Tasks {
			if task.Name == "" {
				return fmt.Errorf("task name cannot be empty")
			}
			if task.Role == "" {
				return fmt.Errorf("task role cannot be empty")
			}
			if task.Time < 0 {
				return fmt.Errorf("task time cannot be negative: %s", task.Name)
			}
			if task.Role != "work" && task.Role != "learn" && task.Role != "rest" {
				return fmt.Errorf("invalid role '%s' for task %s (must be work/learn/rest)", task.Role, task.Name)
			}
		}
	}

	return nil
}

// Helper: getPreviousDays returns all days before the current day in the week
func (s *ScheduleService) getPreviousDays(currentDay string) []string {
	currentOrdinal := dayOrder[strings.ToLower(currentDay)]
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

	var previousDays []string
	for _, day := range days {
		if dayOrder[day] < currentOrdinal {
			previousDays = append(previousDays, day)
		}
	}

	return previousDays
}

// Helper: getDayScheduleFromWeekly extracts a specific day's schedule from weekly schedule
func (s *ScheduleService) getDayScheduleFromWeekly(schedule entity.WeeklySchedule, day string) entity.DaySchedule {
	switch strings.ToLower(day) {
	case "monday":
		return schedule.Monday
	case "tuesday":
		return schedule.Tuesday
	case "wednesday":
		return schedule.Wednesday
	case "thursday":
		return schedule.Thursday
	case "friday":
		return schedule.Friday
	case "saturday":
		return schedule.Saturday
	case "sunday":
		return schedule.Sunday
	default:
		return entity.DaySchedule{}
	}
}

// Helper: getDateForDay calculates the date for a given day in the current week
func (s *ScheduleService) getDateForDay(day string, currentDayOrdinal int) string {
	dayOrdinal := dayOrder[strings.ToLower(day)]
	daysAgo := currentDayOrdinal - dayOrdinal

	// Handle negative case (shouldn't happen if previousDays is filtered correctly)
	if daysAgo < 0 {
		daysAgo += 7 // Wrap around to previous week
	}

	date := time.Now().AddDate(0, 0, -daysAgo)
	return date.Format("2 January 2006")
}

// GetRolloverTasksForGroup returns rollover tasks filtered by role/group
func (s *ScheduleService) GetRolloverTasksForGroup(currentDay string, groupName string) ([]entity.RolloverTask, error) {
	allRollovers, err := s.GetRolloverTasks(currentDay)
	if err != nil {
		return nil, err
	}

	var filtered []entity.RolloverTask
	for _, rollover := range allRollovers {
		// Match rollovers to group (plan matches all, others match by role)
		if groupName == "plan" || rollover.Role == groupName {
			filtered = append(filtered, rollover)
		}
	}

	// Sort by source day (older first), then priority (higher first)
	sort.Slice(filtered, func(i, j int) bool {
		dayI := dayOrder[filtered[i].SourceDay]
		dayJ := dayOrder[filtered[j].SourceDay]

		if dayI != dayJ {
			return dayI < dayJ // Older days first
		}
		return filtered[i].Priority > filtered[j].Priority // Higher priority first
	})

	return filtered, nil
}
