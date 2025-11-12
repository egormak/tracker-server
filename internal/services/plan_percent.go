package services

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"
)

func (s *TaskRecordService) GetTaskPlanPercent() (entity.PlanPercentResponse, error) {

	var planPercent entity.PlanPercentResponse
	// var GroupPlanOrdinal int
	// var GroupPercent int

	for {
		GroupPlanOrdinal, err := s.st.GetGroupPlanPercent()
		if err != nil {
			errMsg := fmt.Errorf("can't get group percent: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_group_percent", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}
		GroupPercent, err := s.st.GetGroupPercent(GroupPlanOrdinal)
		if err != nil {
			if err == storage.ErrListEmpty {
				if err := s.st.CheckIfPlanPercentEmpty(); err != nil {
					if err == storage.ErrAllEmpty {
						return entity.PlanPercentResponse{}, storage.ErrAllEmpty
					}
					errMsg := fmt.Errorf("can't check if plan percent empty: %s", err)
					slog.Error("task_record_service, get_task_plan_percent:check_if_plan_percent_empty", "err", errMsg)
					return entity.PlanPercentResponse{}, errMsg
				}
				if err := s.st.ChangeGroupPlanPercent(GroupPlanOrdinal); err != nil {
					errMsg := fmt.Errorf("can't advance group plan percent: %s", err)
					slog.Error("task_record_service, get_task_plan_percent:change_group_plan_percent", "err", errMsg)
					return entity.PlanPercentResponse{}, errMsg
				}
				continue
			}
			errMsg := fmt.Errorf("can't get group percent: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_group_percent", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}
		groupName, err := s.st.GetGroupName(GroupPlanOrdinal)
		if err != nil {
			errMsg := fmt.Errorf("can't get group name: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_group_name", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}
		TaskNamePlanPercent, err := s.st.GetTaskNamePlanPercent(groupName, GroupPercent)
		if err != nil {
			errMsg := fmt.Errorf("can't get task name plan percent: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_task_name_plan_percent", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}

		if TaskNamePlanPercent != "" {
			timeLeft, _ := s.GetTodayTaskTimeLeft(TaskNamePlanPercent)
			planPercent = entity.PlanPercentResponse{
				TaskName: TaskNamePlanPercent,
				Percent:  GroupPercent,
				TimeLeft: timeLeft,
			}
			break
		}
		if err := s.st.DelGroupPercent(groupName); err != nil {
			errMsg := fmt.Errorf("can't delete group percent: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:del_group_percent", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}
		// Try the next available percent within the same group.
		continue
	}

	return planPercent, nil

}

// ScheduleServiceProvider defines methods needed from ScheduleService for task selection
type ScheduleServiceProvider interface {
	GetTodaySchedule() (*entity.ActiveSchedule, error)
}

// GetTaskPlanPercentWithSchedule integrates the weekly schedule system with plan percent logic
// It prioritizes rollover tasks from previous days, then today's schedule, then falls back to plan percent
func (s *TaskRecordService) GetTaskPlanPercentWithSchedule(scheduleService ScheduleServiceProvider) (entity.PlanPercentResponse, error) {
	// Try to get active schedule
	activeSchedule, err := scheduleService.GetTodaySchedule()
	if err == nil && activeSchedule != nil {
		// We have an active schedule - use schedule-based task selection
		groupOrder := normalizePlanGroup(activeSchedule.PlanGroup)

		// 1. Check rollover tasks first (incomplete from previous days) respecting plan group order
		if len(activeSchedule.RolloverTasks) > 0 {
			sortedRollovers := make([]entity.RolloverTask, len(activeSchedule.RolloverTasks))
			copy(sortedRollovers, activeSchedule.RolloverTasks)

			sort.Slice(sortedRollovers, func(i, j int) bool {
				dayOrderI := dayOrder[strings.ToLower(sortedRollovers[i].SourceDay)]
				dayOrderJ := dayOrder[strings.ToLower(sortedRollovers[j].SourceDay)]

				if dayOrderI != dayOrderJ {
					return dayOrderI < dayOrderJ // Older day first
				}
				return sortedRollovers[i].Priority > sortedRollovers[j].Priority // Higher priority first
			})

			for _, group := range groupOrder {
				for _, rollover := range sortedRollovers {
					if !matchesPlanGroup(rollover.Role, group) {
						continue
					}
					if rollover.RemainingTime <= 0 {
						continue
					}

					slog.Info("Selected rollover task",
						"task", rollover.TaskName,
						"source_day", rollover.SourceDay,
						"priority", rollover.Priority,
						"remaining", rollover.RemainingTime,
						"group", group)

					return entity.PlanPercentResponse{
						TaskName:  rollover.TaskName,
						Percent:   rollover.Percent,
						TimeLeft:  rollover.RemainingTime,
						SourceDay: rollover.SourceDay, // Include source day for rollover tasks
					}, nil
				}
			}
		}

		// 2. No rollovers or all complete - use today's scheduled tasks
		if len(activeSchedule.Tasks) > 0 {
			sortedTasks := make([]entity.ScheduleTask, len(activeSchedule.Tasks))
			copy(sortedTasks, activeSchedule.Tasks)

			sort.Slice(sortedTasks, func(i, j int) bool {
				return sortedTasks[i].Priority > sortedTasks[j].Priority
			})

			timeLeftCache := make(map[string]int)
			getTimeLeft := func(taskName string) (int, error) {
				if cached, ok := timeLeftCache[taskName]; ok {
					return cached, nil
				}
				timeLeft, err := s.GetTodayTaskTimeLeft(taskName)
				if err != nil {
					return 0, err
				}
				timeLeftCache[taskName] = timeLeft
				return timeLeft, nil
			}

			for _, group := range groupOrder {
				for _, task := range sortedTasks {
					if !matchesPlanGroup(task.Role, group) {
						continue
					}
					timeLeft, err := getTimeLeft(task.Name)
					if err != nil {
						slog.Warn("Failed to get time left for scheduled task", "task", task.Name, "error", err)
						continue
					}

					if timeLeft > 0 {
						percent := 100
						if len(task.Percents) > 0 {
							percent = task.Percents[0]
						}

						slog.Info("Selected scheduled task",
							"task", task.Name,
							"priority", task.Priority,
							"time_left", timeLeft,
							"group", group)

						return entity.PlanPercentResponse{
							TaskName:  task.Name,
							Percent:   percent,
							TimeLeft:  timeLeft,
							SourceDay: "", // Empty for today's tasks (not a rollover)
						}, nil
					}
				}
			}
		}

		// All schedule tasks complete - fall through to plan percent logic
		slog.Info("All schedule tasks complete, falling back to plan percent logic")
	} else {
		// No active schedule or error getting it - use original plan percent logic
		if err != nil {
			slog.Debug("No active schedule found, using plan percent logic", "error", err)
		}
	}

	// Fallback to original plan percent logic
	return s.GetTaskPlanPercent()
}

func normalizePlanGroup(groups []string) []string {
	if len(groups) == 0 {
		return []string{"plan", "work", "learn", "rest"}
	}

	normalized := make([]string, 0, len(groups))
	for _, group := range groups {
		g := strings.TrimSpace(strings.ToLower(group))
		if g == "" {
			continue
		}
		normalized = append(normalized, g)
	}

	if len(normalized) == 0 {
		return []string{"plan", "work", "learn", "rest"}
	}

	return normalized
}

func matchesPlanGroup(role string, group string) bool {
	if group == "" || group == "plan" {
		return true
	}

	return strings.ToLower(role) == group
}
