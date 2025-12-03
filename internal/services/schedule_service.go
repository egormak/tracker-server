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

// dayDeficitInfo tracks deficit information for a specific day
type dayDeficitInfo struct {
	deficit  int
	role     string
	priority int
	percent  int
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
	GetTaskNamesForDate(date string) ([]string, error)
	MoveTaskToPreviousDate(taskName string, currentDate string) error
	TimerGlobalSet(timeScheduler int) error
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

	normalizedDay := strings.ToLower(currentDay)
	currentDayOrdinal, ok := dayOrder[normalizedDay]
	if !ok {
		normalizedDay = strings.ToLower(time.Now().Weekday().String())
		currentDayOrdinal = dayOrder[normalizedDay]
	}
	previousDays := s.getPreviousDays(normalizedDay)
	referenceDate := s.resolveReferenceDate(normalizedDay, currentDayOrdinal)

	// Build per-day deficits and convert to rollover list
	return s.buildRolloverList(schedule, previousDays, referenceDate), nil
}

// buildTaskDayDeficits tracks deficits per task per day
// Returns: map[taskName]map[day]deficitInfo
func (s *ScheduleService) buildTaskDayDeficits(schedule entity.WeeklySchedule, previousDays []string, referenceDate time.Time) map[string]map[string]*dayDeficitInfo {
	taskDayDeficits := make(map[string]map[string]*dayDeficitInfo)

	// Process each day separately
	for _, day := range previousDays {
		daySchedule := s.getDayScheduleFromWeekly(schedule, day)
		dayDate := s.getDateForDay(day, referenceDate)

		// For each task scheduled on this day
		for _, task := range daySchedule.Tasks {
			// Get actual work done on this specific day
			timeDone, err := s.st.GetTaskDurationForDate(task.Name, dayDate)
			if err != nil {
				timeDone = 0
			}

			// Calculate deficit for this specific day
			dayDeficit := task.Time - timeDone

			slog.Info("Day task analysis",
				"task", task.Name,
				"day", day,
				"date", dayDate,
				"scheduled", task.Time,
				"done", timeDone,
				"deficit", dayDeficit)

			if dayDeficit > 0 {
				// This day has incomplete work
				if taskDayDeficits[task.Name] == nil {
					taskDayDeficits[task.Name] = make(map[string]*dayDeficitInfo)
				}

				percent := 100
				if len(task.Percents) > 0 {
					percent = task.Percents[0]
				}

				taskDayDeficits[task.Name][day] = &dayDeficitInfo{
					deficit:  dayDeficit,
					role:     task.Role,
					priority: task.Priority,
					percent:  percent,
				}
			}
		}
	}

	return taskDayDeficits
}

// buildRolloverList creates separate rollover tasks for each day with deficits
// Prioritized by: oldest day first, then by task priority
func (s *ScheduleService) buildRolloverList(schedule entity.WeeklySchedule, previousDays []string, referenceDate time.Time) []entity.RolloverTask {
	// Build per-day deficits
	taskDayDeficits := s.buildTaskDayDeficits(schedule, previousDays, referenceDate)

	var rollovers []entity.RolloverTask

	// Create separate rollover task for each task-day combination
	for taskName, dayDeficits := range taskDayDeficits {
		for day, info := range dayDeficits {
			rollovers = append(rollovers, entity.RolloverTask{
				TaskName:      taskName,
				Role:          info.role,
				Priority:      info.priority,
				RemainingTime: info.deficit,
				SourceDay:     day,
				Percent:       info.percent,
			})

			slog.Info("Rollover task created",
				"task", taskName,
				"day", day,
				"deficit", info.deficit,
				"priority", info.priority)
		}
	}

	slog.Info("Calculated rollover tasks", "count", len(rollovers))

	// Sort by oldest day first, then by priority within each day
	sort.Slice(rollovers, func(i, j int) bool {
		dayI := dayOrder[rollovers[i].SourceDay]
		dayJ := dayOrder[rollovers[j].SourceDay]

		// First: order by day (oldest first)
		if dayI != dayJ {
			return dayI < dayJ
		}

		// Second: order by priority (highest first) within same day
		return rollovers[i].Priority > rollovers[j].Priority
	})

	return rollovers
}

// ApplyScheduleToday creates TaskDefinitions for today based on the active schedule
// It only applies tasks that are scheduled for today (does not include rollover tasks)
// Tasks not in today's schedule are moved to the previous day (their date is updated)
// It also sets the global timer to today's total_time
func (s *ScheduleService) ApplyScheduleToday() error {
	today := strings.ToLower(time.Now().Weekday().String())
	todayDate := time.Now().Format("2 January 2006")

	daySchedule, err := s.st.GetDaySchedule(today)
	if err != nil {
		return fmt.Errorf("failed to get today's schedule: %w", err)
	}

	// Set the global timer to today's total time
	if err := s.st.TimerGlobalSet(daySchedule.TotalTime); err != nil {
		slog.Error("Failed to set global timer", "total_time", daySchedule.TotalTime, "error", err)
		// Continue even if this fails - don't block task creation
	} else {
		slog.Info("Set global timer from schedule", "total_time", daySchedule.TotalTime)
	}

	// Get existing tasks for today
	existingTaskNames, err := s.st.GetTaskNamesForDate(todayDate)
	if err != nil {
		slog.Warn("Failed to get existing tasks", "error", err)
		existingTaskNames = []string{}
	}

	// Build a set of scheduled task names for quick lookup
	scheduledTaskNames := make(map[string]bool)
	for _, scheduleTask := range daySchedule.Tasks {
		scheduledTaskNames[scheduleTask.Name] = true
	}

	// Remove tasks that are not in today's schedule
	for _, existingTaskName := range existingTaskNames {
		if !scheduledTaskNames[existingTaskName] {
			if err := s.st.MoveTaskToPreviousDate(existingTaskName, todayDate); err != nil {
				slog.Error("Failed to move task to previous date", "task", existingTaskName, "error", err)
			} else {
				slog.Info("Moved task to previous date (not in today's schedule)", "task", existingTaskName)
			}
		}
	}

	// Apply only scheduled tasks for today (no rollovers)
	for _, scheduleTask := range daySchedule.Tasks {
		taskDef := entity.TaskDefinition{
			Name:         scheduleTask.Name,
			Role:         scheduleTask.Role,
			TimeSchedule: scheduleTask.Time,
			Priority:     scheduleTask.Priority,
			Date:         todayDate,
		}

		if err := s.st.CreateTask(taskDef); err != nil {
			slog.Error("Failed to create/update task from schedule", "task", scheduleTask.Name, "error", err)
			continue
		}

		slog.Info("Applied task from today's schedule", "task", scheduleTask.Name, "time", scheduleTask.Time, "priority", scheduleTask.Priority)
	}

	slog.Info("Applied schedule for today", "day", today, "scheduled_tasks", len(daySchedule.Tasks), "total_time", daySchedule.TotalTime)
	return nil
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
func (s *ScheduleService) getDateForDay(day string, referenceDate time.Time) string {
	dayOrdinal := dayOrder[strings.ToLower(day)]
	referenceOrdinal := dayOrder[strings.ToLower(referenceDate.Weekday().String())]
	daysAgo := referenceOrdinal - dayOrdinal

	// Handle negative case (shouldn't happen if previousDays is filtered correctly)
	if daysAgo < 0 {
		daysAgo += 7 // Wrap around to previous week
	}

	date := referenceDate.AddDate(0, 0, -daysAgo)
	return date.Format("2 January 2006")
}

// resolveReferenceDate returns a time aligned to the requested current day within the current week
func (s *ScheduleService) resolveReferenceDate(currentDay string, currentDayOrdinal int) time.Time {
	now := time.Now()
	todayOrdinal := dayOrder[strings.ToLower(now.Weekday().String())]
	delta := currentDayOrdinal - todayOrdinal
	return now.AddDate(0, 0, delta)
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
