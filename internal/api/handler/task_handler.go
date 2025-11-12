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

type taskStorage interface {
	GetDayTaskRecord(taskName string) (int, error)
}

type TaskHandler struct {
	srv taskService
	st  taskStorage
}

func NewTaskHandler(srv taskService) *TaskHandler {
	return &TaskHandler{srv: srv, st: nil}
}

// NewTaskHandlerWithStorage creates handler with direct storage access for legacy endpoints
func NewTaskHandlerWithStorage(srv taskService, st taskStorage) *TaskHandler {
	return &TaskHandler{srv: srv, st: st}
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

	if t.st == nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Internal configuration error",
		})
	}

	result, err := t.st.GetDayTaskRecord(taskName)
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
