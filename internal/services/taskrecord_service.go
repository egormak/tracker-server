package services

import (
	"fmt"
	"log/slog"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/service"
)

type TaskRecordStorage interface {
	GetRole(taskName string) (string, error)
	AddTaskRecord(task entity.TaskRecord) error
	AddRoleMinutes(task entity.TaskRecord) error
	AddRest(restTime int) error
	GetGroupPlanPercent() (int, error)
	GetGroupPercent(groupPlanOrdinal int) (int, error)
	CheckIfPlanPercentEmpty() error
	ChangeGroupPlanPercent(groupPlanOrdinal int) error
	GetGroupName(groupPlanOrdinal int) (string, error)
	GetTaskNamePlanPercent(groupName string, groupPercent int) (string, error)
	DelGroupPercent(groupName string) error
	GetTodayTaskDuration(taskName string) (int, error)
	GetTaskParams(taskName string) (entity.TaskParams, error)
	GetActiveSchedule() (entity.WeeklySchedule, error)
	GetTaskDurationForDate(taskName string, date string) (int, error)
}

type TaskRecordService struct {
	st TaskRecordStorage
}

func NewTaskRecordService(st TaskRecordStorage) *TaskRecordService {
	return &TaskRecordService{st: st}
}

func (s *TaskRecordService) AddRecord(taskRecordRequest entity.TaskRecordRequest) error {
	slog.Info("=== TaskRecordService.AddRecord START ===",
		"task_name", taskRecordRequest.TaskName,
		"time_done", taskRecordRequest.TimeDone,
		"source_day", taskRecordRequest.SourceDay)

	taskRole, err := s.st.GetRole(taskRecordRequest.TaskName)
	if err != nil {
		errMsg := fmt.Errorf("task role can't get: %s", err)
		slog.Error("task_record_service, add_record:get_role", "err", errMsg)
		return errMsg
	}

	slog.Info("Retrieved task role", "task", taskRecordRequest.TaskName, "role", taskRole)

	// Determine the record date
	// If source_day is provided, calculate the date for that day in the current week
	// Otherwise, use today's date
	todayDate := time.Now().Format("2 January 2006")
	recordDate := todayDate

	slog.Info("Date calculation START",
		"today", todayDate,
		"source_day_provided", taskRecordRequest.SourceDay,
		"source_day_empty", taskRecordRequest.SourceDay == "",
		"current_weekday", time.Now().Weekday().String())

	if taskRecordRequest.SourceDay != "" {
		recordDate = service.CalculateDateForDay(taskRecordRequest.SourceDay)
		slog.Info("✅ Recording task against SOURCE DAY",
			"task", taskRecordRequest.TaskName,
			"source_day", taskRecordRequest.SourceDay,
			"calculated_date", recordDate,
			"today_date", todayDate,
			"dates_different", recordDate != todayDate)
	} else if taskRecordRequest.ManageByService {
		// New Logic: Backfill Schedule
		// 1. Get Active Schedule
		schedule, err := s.st.GetActiveSchedule()
		if err == nil { // If no active schedule, skip this logic
			// 2. Iterate from Monday to Yesterday
			weekDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
			todayWeekday := time.Now().Weekday()
			// Convertible to our string format (Monday=1, Sunday=0 in Go's time, but let's just use string comparison or simple mapping if needed)
			// Actually simpler: iterate our weekDays list until we hit "today"

			// Map Go's Weekday (Sun=0, Mon=1...) to index in our list (Mon=0, Sun=6)
			// specific logic for "today" to stop before it
			todayIndex := -1
			// todayStr := time.Now().Format("Monday") // "Monday", "Tuesday"...
			for i, d := range weekDays {
				// simple case insensitive check or just use the fact they match mostly
				if d == "monday" && todayWeekday == time.Monday {
					todayIndex = i
				} else if d == "tuesday" && todayWeekday == time.Tuesday {
					todayIndex = i
				} else if d == "wednesday" && todayWeekday == time.Wednesday {
					todayIndex = i
				} else if d == "thursday" && todayWeekday == time.Thursday {
					todayIndex = i
				} else if d == "friday" && todayWeekday == time.Friday {
					todayIndex = i
				} else if d == "saturday" && todayWeekday == time.Saturday {
					todayIndex = i
				} else if d == "sunday" && todayWeekday == time.Sunday {
					todayIndex = i
				}
			}

			if todayIndex > 0 {
				slog.Info("Checking past days for backfill", "today_index", todayIndex)
				remainingTime := taskRecordRequest.TimeDone

				for i := 0; i < todayIndex; i++ {
					if remainingTime <= 0 {
						break
					}
					checkDay := weekDays[i]

					// Get scheduled time for this task on checkDay
					scheduledTime := 0
					var daySchedule entity.DaySchedule
					switch checkDay {
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

					for _, t := range daySchedule.Tasks {
						if t.Name == taskRecordRequest.TaskName {
							scheduledTime = t.Time
							break
						}
					}

					if scheduledTime > 0 {
						// Check how much is already done for checkDay
						// Need to calculate date for that day
						dateForDay := service.CalculateDateForDay(checkDay)
						doneTime, err := s.st.GetTaskDurationForDate(taskRecordRequest.TaskName, dateForDay)
						if err != nil {
							slog.Error("failed to get task duration for backfill", "day", checkDay, "err", err)
							continue
						}

						if doneTime < scheduledTime {
							needed := scheduledTime - doneTime
							fillAmount := 0
							if remainingTime >= needed {
								fillAmount = needed
							} else {
								fillAmount = remainingTime
							}

							if fillAmount > 0 {
								// Create record for this past day
								slog.Info("Backfilling task",
									"task", taskRecordRequest.TaskName,
									"day", checkDay,
									"amount", fillAmount)

								pastRecord := entity.TaskRecord{
									Name:         taskRecordRequest.TaskName,
									Role:         taskRole,
									TimeDuration: fillAmount,
									Date:         dateForDay,
								}
								if err := s.st.AddTaskRecord(pastRecord); err != nil {
									slog.Error("failed to add backfill record", "err", err)
								} else {
									// Update remaining time
									remainingTime -= fillAmount

									// Also add role minutes and rest for consistency?
									// Usually yes, as it counts towards that day's stats
									s.st.AddRoleMinutes(pastRecord)
									// Assuming rest is calculated daily or simply added to pool,
									// if pool is global, we add it.
									s.st.AddRest(fillAmount)
								}
							}
						}
					}
				}
				// Update the request with remaining time for today
				taskRecordRequest.TimeDone = remainingTime
				slog.Info("Finished backfill", "remaining_for_today", taskRecordRequest.TimeDone)
			}
		} else {
			slog.Warn("ManageByService requested but could not get active schedule", "err", err)
		}

	} else {
		slog.Info("⚠️  NO SOURCE DAY - Recording against TODAY",
			"task", taskRecordRequest.TaskName,
			"record_date", recordDate)
	}

	// If all time was used in backfill, we can return early or record 0?
	// The original logic records whatever is in taskRecordRequest.TimeDone.
	// If it's 0, we might strictly skip recording a 0-minute entry.
	if taskRecordRequest.TimeDone <= 0 {
		slog.Info("All time distributed to past schedules, nothing left for today.")
		return nil
	}

	record := entity.TaskRecord{
		Name:         taskRecordRequest.TaskName,
		Role:         taskRole,
		TimeDuration: taskRecordRequest.TimeDone,
		Date:         recordDate,
	}

	slog.Info("Creating task record",
		"name", record.Name,
		"role", record.Role,
		"duration", record.TimeDuration,
		"date", record.Date)

	if err := s.st.AddTaskRecord(record); err != nil {
		errMsg := fmt.Errorf("can't add record: %s", err)
		slog.Error("task_record_service, add_record:add_task_record", "err", errMsg)
		return errMsg
	}

	slog.Info("✅ Task record SAVED to database",
		"task", record.Name,
		"date", record.Date,
		"duration", record.TimeDuration)

	if err := s.st.AddRoleMinutes(record); err != nil {
		errMsg := fmt.Errorf("can't add role minutes: %s", err)
		slog.Error("task_record_service, add_record:add_role_minutes", "err", errMsg)
		return errMsg
	}

	if err := s.st.AddRest(record.TimeDuration); err != nil {
		errMsg := fmt.Errorf("can't add rest: %s", err)
		slog.Error("task_record_service, add_record:add_rest", "err", errMsg)
		return errMsg
	}

	return nil
}

// func (s *TaskRecordService) GetTasksNext() (entity.TasksNextResponse, error) {
// 	return entity.TasksNextResponse{}, nil
// }

// GetTodayTaskTimeLeft calculates the remaining time for a task for today
func (s *TaskRecordService) GetTodayTaskTimeLeft(taskName string) (int, error) {
	// Retrieve today's task duration
	timeDuration, err := s.st.GetTodayTaskDuration(taskName)
	if err != nil {
		errMsg := fmt.Errorf("unable to retrieve today's task duration: %s", err)
		slog.Error("task_record_service, get_today_task_left:get_today_task_duration", "err", errMsg)
		return 0, errMsg
	}

	// Retrieve task parameters
	taskParams, err := s.st.GetTaskParams(taskName)
	if err != nil {
		errMsg := fmt.Errorf("unable to retrieve task parameters: %s", err)
		slog.Error("task_record_service, get_today_task_left:get_task_params", "err", errMsg)
		return 0, errMsg
	}

	// Calculate the remaining time for the task
	taskTimeLeft := taskParams.Time - timeDuration

	return taskTimeLeft, nil
}
