package handler

import (
	"log/slog"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type StatisticService interface {
	GetTaskRecordToday() ([]entity.TaskResult, error)
	GetWeeklyStats() (entity.WeeklyStatsResponse, error)
}

type RunningTaskProvider interface {
	GetStatus(taskName string) (entity.RunningTask, error)
}

type RestProvider interface {
	GetRest() (int, error)
}

type EveningProvider interface {
	GetEveningFocus(category string, timeOverride int) (entity.EveningFocusResponse, error)
}

type StatisticHandler struct {
	srv         StatisticService
	runningTask RunningTaskProvider
	rest        RestProvider
	evening     EveningProvider
}

func NewStatisticHandler(
	srv StatisticService,
	runningTask RunningTaskProvider,
	rest RestProvider,
	evening EveningProvider,
) *StatisticHandler {
	return &StatisticHandler{
		srv:         srv,
		runningTask: runningTask,
		rest:        rest,
		evening:     evening,
	}
}

// GetWeeklyStats returns task completion metrics and targets for the current week
func (s *StatisticHandler) GetWeeklyStats(c *fiber.Ctx) error {
	slog.Info("Get request GetWeeklyStats")

	statsData, err := s.srv.GetWeeklyStats()
	if err != nil {
		slog.Error("Failed to get weekly stats", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(statsData)
}

// TODO: Finish this function
func (s *StatisticHandler) StatCompletionTimeDone(c *fiber.Ctx) error {
	slog.Info("Get request StatCompletionTimeDone")

	// var answer StatisticCompletion

	statsData, err := s.srv.GetTaskRecordToday()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(statsData)

}

// ShowTaskList returns today's tasks with scheduled time and actual time done (legacy endpoint)
func (s *StatisticHandler) ShowTaskList(c *fiber.Ctx) error {
	slog.Info("Get request ShowTaskList")

	taskList, err := s.srv.GetTaskRecordToday()
	if err != nil {
		slog.Error("Failed to get task list", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(taskList)
}

// GetDashboardState aggregates running task, today's tasks, rest pool, and evening focus in a single response
func (s *StatisticHandler) GetDashboardState(c *fiber.Ctx) error {
	slog.Info("Get request GetDashboardState")

	resp := entity.DashboardStateResponse{
		TodayTasks: []entity.TaskResult{},
	}

	// 1. Running Task
	if s.runningTask != nil {
		active, err := s.runningTask.GetStatus("")
		if err == nil && active.TaskName != "" {
			resp.RunningTask = &active
		}
	}

	// 2. Today's Tasks & totals
	todayTasks, err := s.srv.GetTaskRecordToday()
	if err != nil {
		slog.Error("Failed to get today tasks for dashboard state", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	if todayTasks != nil {
		resp.TodayTasks = todayTasks
	}

	totalPlanned := 0
	totalDone := 0
	for _, t := range resp.TodayTasks {
		totalPlanned += t.TimeDuration
		totalDone += t.TimeDone
	}
	resp.TotalPlanned = totalPlanned
	resp.TotalDone = totalDone
	if totalPlanned > 0 {
		resp.CompletionPct = int(float64(totalDone) / float64(totalPlanned) * 100)
	}

	// 3. Rest Pool balance (raw units: units = minutes * 100)
	if s.rest != nil {
		restUnits, err := s.rest.GetRest()
		if err != nil {
			slog.Error("Failed to get rest units for dashboard state", "err", err)
		} else {
			resp.RestPool = restUnits
		}
	}

	// 4. Evening Focus (optional, top 3 candidates)
	if s.evening != nil {
		focus, err := s.evening.GetEveningFocus("", 0)
		if err != nil {
			slog.Warn("Failed to get evening focus for dashboard state", "err", err)
		} else {
			if len(focus.Candidates) > 3 {
				focus.Candidates = focus.Candidates[:3]
			}
			resp.EveningFocus = &focus
		}
	}

	return c.Status(200).JSON(resp)
}
