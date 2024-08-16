package handler

import (
	"log/slog"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type StatisticService interface {
	GetTaskRecordToday() ([]entity.TaskResult, error)
}

type StatisticHandler struct {
	srv StatisticService
}

func NewStatisticHandler(srv StatisticService) *StatisticHandler {
	return &StatisticHandler{srv: srv}
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
