package handler

import (
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type taskService interface {
	GetTaskParams(taskName string) (entity.TaskParams, error)
	// AddRecord(body entity.TaskRecordRequest) error
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

// func (t *TaskRecordHandler) AddRecord(c *fiber.Ctx) error {
// 	slog.Info("Get request addRecord")

// 	var body entity.TaskRecordRequest

// 	if err := c.BodyParser(&body); err != nil {
// 		errMsg := fmt.Errorf("can't parse body: %s", err)
// 		slog.Error("Can't parse body", "err", err)
// 		return c.Status(500).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": errMsg.Error(),
// 		})
// 	}

// 	if body.TaskName == "" {
// 		errMsg := "Task name is not Set"
// 		slog.Error(errMsg)
// 		return c.Status(500).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": errMsg,
// 		})
// 	}

// 	if body.TimeDone == 0 {
// 		errMsg := "Task duration is not Set"
// 		slog.Error(errMsg)
// 		return c.Status(500).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": errMsg,
// 		})
// 	}

// 	if err := t.srv.AddRecord(body); err != nil {
// 		errMsg := fmt.Errorf("can't add record: %s", err)
// 		slog.Error("Can't add record", "err", err)
// 		return c.Status(500).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": errMsg.Error(),
// 		})
// 	}

// 	slog.Info("Record was added")
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status":  "accept",
// 		"message": "Record was added",
// 	})
// }
