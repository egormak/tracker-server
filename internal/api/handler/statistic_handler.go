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

type StatisticHandler struct {
	srv StatisticService
}

func NewStatisticHandler(srv StatisticService) *StatisticHandler {
	return &StatisticHandler{srv: srv}
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
