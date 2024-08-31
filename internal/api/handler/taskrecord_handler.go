package handler

import (
	"fmt"
	"log/slog"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type taskRecordService interface {
	AddRecord(body entity.TaskRecordRequest) error
}

type TaskRecordHandler struct {
	srv taskRecordService
}

func NewTaskRecordHandler(srv taskRecordService) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv}
}

func (t *TaskRecordHandler) AddRecord(c *fiber.Ctx) error {
	slog.Info("Get request addRecord")

	var body entity.TaskRecordRequest

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		slog.Error("Can't parse body", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if body.TaskName == "" {
		errMsg := "Task name is not Set"
		slog.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg,
		})
	}

	if body.TimeDone == 0 {
		errMsg := "Task duration is not Set"
		slog.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg,
		})
	}

	if err := t.srv.AddRecord(body); err != nil {
		errMsg := fmt.Errorf("can't add record: %s", err)
		slog.Error("Can't add record", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	slog.Info("Record was added")
	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Record was added",
	})
}
