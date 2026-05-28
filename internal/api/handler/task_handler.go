package handler

import (
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type taskService interface {
	GetTaskParams(taskName string) (entity.TaskParams, error)
	GetDayTaskRecord(taskName string) (int, error)
}

type TaskHandler struct {
	srv taskService
}

func NewTaskHandler(srv taskService) *TaskHandler {
	return &TaskHandler{srv: srv}
}

func (t *TaskHandler) TaskParams(c *fiber.Ctx) error {
	taskName := c.Query("task_name")

	result, err := t.srv.GetTaskParams(taskName)
	if err != nil {
		status := 500
		message := "error"

		switch err {
		case storage.ErrTaskNotFound:
			status = 404
			message = "Task Not Found"
		case storage.ErrParamsOld:
			status = 404
			message = "params old"
		}

		return c.Status(status).JSON(&fiber.Map{
			"status":  message,
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(result)
}

// GetDayTaskRecord returns the total time spent on a task today (legacy endpoint for CLI)
func (t *TaskHandler) GetDayTaskRecord(c *fiber.Ctx) error {
	taskName := c.Query("task_name")

	result, err := t.srv.GetDayTaskRecord(taskName)
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":        "Done",
		"task_duration": result,
	})
}
