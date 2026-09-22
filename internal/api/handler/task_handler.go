package handler

import (
	"errors"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type taskService interface {
	GetTaskParams(taskName string) (entity.TaskParams, error)
	SetTaskParams(params entity.TaskParams) error
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

		if errors.Is(err, storage.ErrTaskNotFound) {
			status = 404
			message = "Task Not Found"
		} else if errors.Is(err, storage.ErrParamsOld) {
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

// SetTaskParams sets task parameters (supports name, time_duration or time, and priority)
func (t *TaskHandler) SetTaskParams(c *fiber.Ctx) error {
	var req struct {
		Name         string `json:"name"`
		TimeDuration int    `json:"time_duration"`
		Time         int    `json:"time"`
		Priority     int    `json:"priority"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "cannot parse json body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "task name is required",
		})
	}

	taskTime := req.TimeDuration
	if taskTime == 0 && req.Time != 0 {
		taskTime = req.Time
	}

	params := entity.TaskParams{
		Name:     req.Name,
		Time:     taskTime,
		Priority: req.Priority,
	}

	if err := t.srv.SetTaskParams(params); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "task params updated successfully",
	})
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
