package services

import (
	"fmt"
	"log/slog"
	"time"
	"tracker-server/internal/domain/entity"
)

type StatisticStorage interface {
	ShowTaskList() ([]entity.TaskResult, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetRecordsForDates(dates []string) ([]entity.TaskRecord, error)
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
	return list, nil
}

// GetWeeklyStats returns aggregated task and role metrics for the current calendar week (Monday to Sunday)
func (s *StatisticService) GetWeeklyStats() (entity.WeeklyStatsResponse, error) {
	// 1. Get dates for the current week (Monday to Sunday)
	now := time.Now()
	weekday := now.Weekday()
	daysSinceMonday := int(weekday) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday is day 6 relative to Monday
	}

	mondayDate := now.AddDate(0, 0, -daysSinceMonday)

	weekdays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	dates := make([]string, 7)
	dayToDate := make(map[string]string)

	for i, day := range weekdays {
		dates[i] = mondayDate.AddDate(0, 0, i).Format("2 January 2006")
		dayToDate[day] = dates[i]
	}

	// 2. Fetch active schedule (gracefully degrade if missing)
	schedule, err := s.st.GetActiveSchedule()
	if err != nil {
		slog.Warn("GetWeeklyStats: no active schedule found, proceeding with empty schedule", "err", err)
		schedule = entity.WeeklySchedule{}
	}

	// 3. Fetch records for the current week's dates
	records, err := s.st.GetRecordsForDates(dates)
	if err != nil {
		return entity.WeeklyStatsResponse{}, fmt.Errorf("failed to get records for week: %w", err)
	}

	// Group records by date
	recordsByDate := make(map[string][]entity.TaskRecord)
	for _, rec := range records {
		recordsByDate[rec.Date] = append(recordsByDate[rec.Date], rec)
	}

	// 4. Process daily stats and aggregate weekly targets
	daysStats := make([]entity.WeeklyStatsDay, 7)
	roleTargets := make(map[string]*entity.WeeklyTarget)
	taskTargets := make(map[string]*entity.WeeklyTaskTarget)

	// Initialize role targets map with standard roles
	for _, r := range []string{"work", "learn", "rest"} {
		roleTargets[r] = &entity.WeeklyTarget{Role: r, TimeDuration: 0, TimeDone: 0}
	}

	for i, day := range weekdays {
		dateStr := dayToDate[day]

		// Get day's schedule
		var daySchedule entity.DaySchedule
		switch day {
		case "monday":
			daySchedule = schedule.Monday
		case "tuesday":
			daySchedule = schedule.Tuesday
		case "wednesday":
			daySchedule = schedule.Wednesday
		case "thursday":
			daySchedule = schedule.Thursday
		case "friday":
			daySchedule = schedule.Friday
		case "saturday":
			daySchedule = schedule.Saturday
		case "sunday":
			daySchedule = schedule.Sunday
		}

		// Map of task name -> TaskResult for this day
		dayTasks := make(map[string]*entity.TaskResult)

		// Fill map with scheduled tasks for this day
		for _, task := range daySchedule.Tasks {
			dayTasks[task.Name] = &entity.TaskResult{
				Name:         task.Name,
				Role:         string(task.Role),
				TimeDuration: task.Time,
				TimeDone:     0,
				Priority:     task.Priority,
			}

			// Accumulate to weekly targets
			roleStr := string(task.Role)
			if rt, exists := roleTargets[roleStr]; exists {
				rt.TimeDuration += task.Time
			} else {
				roleTargets[roleStr] = &entity.WeeklyTarget{Role: roleStr, TimeDuration: task.Time, TimeDone: 0}
			}

			if tt, exists := taskTargets[task.Name]; exists {
				tt.TimeDuration += task.Time
			} else {
				taskTargets[task.Name] = &entity.WeeklyTaskTarget{
					Name:         task.Name,
					Role:         roleStr,
					TimeDuration: task.Time,
					TimeDone:     0,
				}
			}
		}

		// Add actual records for this day
		dayRecords := recordsByDate[dateStr]
		dayRolesDone := make(map[string]int)
		totalDone := 0

		for _, rec := range dayRecords {
			totalDone += rec.TimeDuration
			dayRolesDone[rec.Role] += rec.TimeDuration

			// Accumulate to weekly roles
			if rt, exists := roleTargets[rec.Role]; exists {
				rt.TimeDone += rec.TimeDuration
			} else {
				roleTargets[rec.Role] = &entity.WeeklyTarget{Role: rec.Role, TimeDuration: 0, TimeDone: rec.TimeDuration}
			}

			// Accumulate to daily task list
			if tr, exists := dayTasks[rec.Name]; exists {
				tr.TimeDone += rec.TimeDuration
			} else {
				dayTasks[rec.Name] = &entity.TaskResult{
					Name:         rec.Name,
					Role:         rec.Role,
					TimeDuration: 0,
					TimeDone:     rec.TimeDuration,
					Priority:     0,
				}
			}

			// Accumulate to weekly tasks
			if tt, exists := taskTargets[rec.Name]; exists {
				tt.TimeDone += rec.TimeDuration
			} else {
				taskTargets[rec.Name] = &entity.WeeklyTaskTarget{
					Name:         rec.Name,
					Role:         rec.Role,
					TimeDuration: 0,
					TimeDone:     rec.TimeDuration,
				}
			}
		}

		// Convert dayTasks map to slice
		tasksList := make([]entity.TaskResult, 0, len(dayTasks))
		for _, tr := range dayTasks {
			tasksList = append(tasksList, *tr)
		}

		daysStats[i] = entity.WeeklyStatsDay{
			Day:       day,
			Date:      dateStr,
			TotalDone: totalDone,
			Roles:     dayRolesDone,
			Tasks:     tasksList,
		}
	}

	// Convert role targets map to slice
	weeklyTargets := make([]entity.WeeklyTarget, 0, len(roleTargets))
	for _, rt := range roleTargets {
		weeklyTargets = append(weeklyTargets, *rt)
	}

	// Convert task targets map to slice
	weeklyTasks := make([]entity.WeeklyTaskTarget, 0, len(taskTargets))
	for _, tt := range taskTargets {
		weeklyTasks = append(weeklyTasks, *tt)
	}

	return entity.WeeklyStatsResponse{
		WeekStartDate: dates[0],
		WeekEndDate:   dates[6],
		Days:          daysStats,
		WeeklyTargets: weeklyTargets,
		WeeklyTasks:   weeklyTasks,
	}, nil
}

