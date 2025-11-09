package handler

import (
	"fmt"
	"log/slog"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/services"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type scheduleService interface {
	GetTodaySchedule() (*entity.ActiveSchedule, error)
}

type taskRecordService interface {
	AddRecord(body entity.TaskRecordRequest) error
	// GetTasksNext() (entity.TasksNextResponse, error)
	GetTaskPlanPercent() (entity.PlanPercentResponse, error)
	GetTaskPlanPercentWithSchedule(scheduleService services.ScheduleServiceProvider) (entity.PlanPercentResponse, error)
}

type TaskRecordHandler struct {
	srv             taskRecordService
	scheduleService scheduleService
}

func NewTaskRecordHandler(srv taskRecordService) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv, scheduleService: nil}
}

// NewTaskRecordHandlerWithSchedule creates handler with schedule integration
func NewTaskRecordHandlerWithSchedule(srv taskRecordService, scheduleSrv scheduleService) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv, scheduleService: scheduleSrv}
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

func (t *TaskRecordHandler) GetTaskPlanPercent(c *fiber.Ctx) error {
	slog.Info("Get request GetTaskPlanPercent")

	answer, err := t.srv.GetTaskPlanPercent()
	if err != nil {
		statusCode := 500
		if err == storage.ErrAllEmpty {
			statusCode = 404
		}
		slog.Error("Error getting task plan percent", "err", err)
		return c.Status(statusCode).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	slog.Info("Sent answer", "request", "GetTaskPlanPercent", "answer", answer)
	return c.Status(200).JSON(answer)
}

// GetTaskPlanPercentWithSchedule returns next task considering weekly schedule and rollovers
func (t *TaskRecordHandler) GetTaskPlanPercentWithSchedule(c *fiber.Ctx) error {
	slog.Info("Get request GetTaskPlanPercentWithSchedule")

	// If no schedule service is configured, fall back to regular plan percent
	if t.scheduleService == nil {
		slog.Info("No schedule service configured, using regular plan percent")
		return t.GetTaskPlanPercent(c)
	}

	answer, err := t.srv.GetTaskPlanPercentWithSchedule(t.scheduleService)
	if err != nil {
		statusCode := 500
		if err == storage.ErrAllEmpty {
			statusCode = 404
		}
		slog.Error("Error getting task plan percent with schedule", "err", err)
		return c.Status(statusCode).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	slog.Info("Sent answer", "request", "GetTaskPlanPercentWithSchedule", "answer", answer)
	return c.Status(200).JSON(answer)
}
