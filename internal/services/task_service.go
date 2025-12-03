package services

import (
	"fmt"
	"strings"
	"time"
	"tracker-server/internal/domain/entity"
)

type TaskStorage interface {
	GetTaskParams(taskName string) (entity.TaskParams, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetTaskDurationForDate(taskName string, date string) (int, error)
}

type TaskNotify interface {
	SendMessageStart(taskName string) (int, error)
	SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error
}

type TaskService struct {
	st TaskStorage
	nt TaskNotify
}

func NewTaskService(st TaskStorage, nt TaskNotify) *TaskService {
	return &TaskService{st: st, nt: nt}
}

// GetTaskParams retrieves task parameters, checking rollover tasks from schedule if not found in today's task list
func (t *TaskService) GetTaskParams(taskName string) (entity.TaskParams, error) {
	// Try to get from today's task list first
	params, err := t.st.GetTaskParams(taskName)
	if err == nil {
		return params, nil
	}

	// If not found, check if it's a rollover task from the active schedule
	schedule, schedErr := t.st.GetActiveSchedule()
	if schedErr != nil {
		// No schedule available, return original error
		return entity.TaskParams{}, err
	}

	// Get today's day name
	today := strings.ToLower(time.Now().Weekday().String())

	// Get previous days
	previousDays := getPreviousDaysForTask(today)

	// Search for the task in previous days' schedules
	for _, day := range previousDays {
		daySchedule := getDayScheduleFromWeekly(schedule, day)

		// Look for the task in this day's schedule
		for _, task := range daySchedule.Tasks {
			if task.Name == taskName {
				// Found the task in a previous day - return its params
				return entity.TaskParams{
					Name:     task.Name,
					Time:     task.Time,
					Priority: task.Priority,
				}, nil
			}
		}
	}

	// Task not found in schedule either, return original error
	return entity.TaskParams{}, fmt.Errorf("task not found in today's list or schedule: %w", err)
}

// Helper functions
func getPreviousDaysForTask(currentDay string) []string {
	dayOrder := map[string]int{
		"monday":    0,
		"tuesday":   1,
		"wednesday": 2,
		"thursday":  3,
		"friday":    4,
		"saturday":  5,
		"sunday":    6,
	}

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

func getDayScheduleFromWeekly(schedule entity.WeeklySchedule, day string) entity.DaySchedule {
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
