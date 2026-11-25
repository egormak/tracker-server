package handler

import (
	"fmt"
	"log/slog"
	"strings"
	"tracker-server/internal/api/middleware"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type scheduleService interface {
	GetTodaySchedule() (*entity.ActiveSchedule, error)
}

type taskRecordService interface {
	AddRecord(body entity.TaskRecordRequest) error
	GetTaskPlanPercent() (entity.PlanPercentResponse, error)
	GetTaskPlanPercentWithSchedule(scheduleService services.ScheduleServiceProvider) (entity.PlanPercentResponse, error)
	GetTaskByNameSchedule(taskName string, scheduleService services.ScheduleServiceProvider) (entity.PlanPercentResponse, error)
	CleanRecords()
	RotatePlanPercent() error
}

type TaskRecordHandler struct {
	srv             taskRecordService
	scheduleService scheduleService
}

func NewTaskRecordHandler(srv taskRecordService, scheduleSrv scheduleService) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv, scheduleService: scheduleSrv}
}

func (t *TaskRecordHandler) AddRecord(c *fiber.Ctx) error {
	slog.Info("=== RECEIVED AddRecord Request ===")

	// Log raw body for debugging
	rawBody := string(c.Body())
	slog.Info("Raw request body", "body", rawBody)

	var body entity.TaskRecordRequest

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		slog.Error("Can't parse body", "err", err, "raw_body", rawBody)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	body.SourceDevice = middleware.Client(c)

	// Log parsed request details
	slog.Info("Parsed request body",
		"task_name", body.TaskName,
		"time_done", body.TimeDone,
		"source_day", body.SourceDay,
		"source_day_empty", body.SourceDay == "",
		"source_day_length", len(body.SourceDay))

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
// If task_name query parameter is provided, searches for that specific task
func (t *TaskRecordHandler) GetTaskPlanPercentWithSchedule(c *fiber.Ctx) error {
	taskName := c.Query("task_name")

	// If task_name is provided, use the task-specific search
	if taskName != "" {
		slog.Info("Get request GetTaskPlanPercentWithSchedule with task_name", "task_name", taskName)
		return t.GetTaskByNameSchedule(c)
	}

	slog.Info("Get request GetTaskPlanPercentWithSchedule")

	// If no schedule service is configured, fall back to regular plan percent
	if t.scheduleService == nil {
		slog.Info("No schedule service configured, using regular plan percent")
		return t.GetTaskPlanPercent(c)
	}

	answer, err := t.srv.GetTaskPlanPercentWithSchedule(t.scheduleService)
	if err != nil {
		statusCode := 500
		slog.Error("Error getting task plan percent with schedule", "err", err)
		return c.Status(statusCode).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	slog.Info("Sent answer", "request", "GetTaskPlanPercentWithSchedule", "answer", answer)
	return c.Status(200).JSON(answer)
}

// GetTaskByNameSchedule returns a specific task by name with schedule awareness
// Searches from Monday through today for available rollover tasks
func (t *TaskRecordHandler) GetTaskByNameSchedule(c *fiber.Ctx) error {
	taskName := c.Query("task_name")

	slog.Info("Get request GetTaskByNameSchedule", "task_name", taskName)

	if taskName == "" {
		slog.Error("Missing task_name parameter")
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "task_name parameter is required",
		})
	}

	// If no schedule service is configured, return error
	if t.scheduleService == nil {
		slog.Error("No schedule service configured")
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Schedule service not available",
		})
	}

	answer, err := t.srv.GetTaskByNameSchedule(taskName, t.scheduleService)
	if err != nil {
		statusCode := 500
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			statusCode = 404
		}
		slog.Error("Error getting task by name from schedule", "err", err, "task_name", taskName)
		return c.Status(statusCode).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	slog.Info("Sent answer", "request", "GetTaskByNameSchedule", "task_name", taskName, "answer", answer)
	return c.Status(200).JSON(answer)
}

// ChangeGroupPlanPercent rotates to the next plan percent group (legacy endpoint for CLI)
func (t *TaskRecordHandler) ChangeGroupPlanPercent(c *fiber.Ctx) error {
	slog.Info("Get request ChangeGroupPlanPercent")

	if err := t.srv.RotatePlanPercent(); err != nil {
		slog.Error("Failed to change group plan percent", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	slog.Info("Plan percent group rotated successfully")
	return c.Status(200).JSON(&fiber.Map{
		"status": "accept",
	})
}

// CleanRecords cleans all records (dev endpoint)
func (t *TaskRecordHandler) CleanRecords(c *fiber.Ctx) error {
	slog.Info("Start Clean Data")
	t.srv.CleanRecords()
	slog.Info("Finish Data Clean")

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Data was erased",
	})
}
