package handler

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
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
	GetTaskByNameSchedule(taskName string, scheduleService services.ScheduleServiceProvider) (entity.PlanPercentResponse, error)
}

type planPercentStorage interface {
	GetGroupPlanPercent() (int, error)
	ChangeGroupPlanPercent(groupPlan int) error
	GetRecords() ([]storage.TaskRecord, error)
	CleanRecords()
}

type TaskRecordHandler struct {
	srv             taskRecordService
	scheduleService scheduleService
	st              planPercentStorage
}

func NewTaskRecordHandler(srv taskRecordService) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv, scheduleService: nil, st: nil}
}

// NewTaskRecordHandlerWithSchedule creates handler with schedule integration
func NewTaskRecordHandlerWithSchedule(srv taskRecordService, scheduleSrv scheduleService) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv, scheduleService: scheduleSrv, st: nil}
}

// NewTaskRecordHandlerWithStorage creates handler with direct storage access for legacy endpoints
func NewTaskRecordHandlerWithStorage(srv taskRecordService, scheduleSrv scheduleService, st planPercentStorage) *TaskRecordHandler {
	return &TaskRecordHandler{srv: srv, scheduleService: scheduleSrv, st: st}
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

	if t.st == nil {
		slog.Error("Storage not initialized for ChangeGroupPlanPercent")
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Internal configuration error",
		})
	}

	groupPlan, err := t.st.GetGroupPlanPercent()
	if err != nil {
		slog.Error("Failed to get group plan percent", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if err := t.st.ChangeGroupPlanPercent(groupPlan); err != nil {
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

// ShowRecords returns a JSON response with task records for today, yesterday, and all time (legacy endpoint for web UI)
func (t *TaskRecordHandler) ShowRecords(c *fiber.Ctx) error {
	if t.st == nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Internal configuration error",
		})
	}

	// Get today's and yesterday's dates in the required format
	today := time.Now().Format("2 January 2006")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2 January 2006")

	// Retrieve task records from the storage
	records, err := t.st.GetRecords()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Initialize maps to store task records for today, yesterday, and all time
	taskRecordsToday := make(map[string]int)
	taskRecordsYesterday := make(map[string]int)
	taskRecordsAll := make(map[string]int)

	// Iterate over the records and update the corresponding maps based on the date
	for _, v := range records {
		if v.Date == today {
			taskRecordsToday[v.Name] += v.TimeDuration
		} else if v.Date == yesterday {
			taskRecordsYesterday[v.Name] += v.TimeDuration
		}
		taskRecordsAll[v.Name] += v.TimeDuration
	}

	// Create a response map with task records for today, yesterday, and all time
	answer := map[string]map[string]int{
		"today":     taskRecordsToday,
		"yesterday": taskRecordsYesterday,
		"all":       taskRecordsAll,
	}

	return c.JSON(answer)
}

// CleanRecords cleans all records (dev endpoint)
func (t *TaskRecordHandler) CleanRecords(c *fiber.Ctx) error {
	if t.st == nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Internal configuration error",
		})
	}

	slog.Info("Start Clean Data")
	t.st.CleanRecords()
	slog.Info("Finish Data Clean")

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Data was erased",
	})
}
