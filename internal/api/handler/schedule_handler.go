package handler

import (
	"log/slog"
	"strings"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

// ScheduleHandler handles HTTP requests for schedule management
type ScheduleHandler struct {
	service *services.ScheduleService
}

// NewScheduleHandler creates a new ScheduleHandler
func NewScheduleHandler(service *services.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{service: service}
}

// CreateSchedule handles POST /api/v1/schedule
func (h *ScheduleHandler) CreateSchedule(c *fiber.Ctx) error {
	var request struct {
		Schedule  entity.ScheduleRequest `json:"schedule"`
		SetActive bool                   `json:"set_active"`
	}

	if err := c.BodyParser(&request); err != nil {
		slog.Error("Failed to parse schedule request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request format",
		})
	}

	scheduleID, err := h.service.CreateSchedule(request.Schedule, request.SetActive)
	if err != nil {
		slog.Error("Failed to create schedule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"schedule_id": scheduleID,
			"is_active":   request.SetActive,
		},
		"message": "Schedule created successfully",
	})
}

// GetActiveSchedule handles GET /api/v1/schedule/active
func (h *ScheduleHandler) GetActiveSchedule(c *fiber.Ctx) error {
	schedule, err := h.service.GetActiveSchedule()
	if err != nil {
		slog.Error("Failed to get active schedule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   schedule,
	})
}

// GetSchedule handles GET /api/v1/schedule/:id
func (h *ScheduleHandler) GetSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	if scheduleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Schedule ID is required",
		})
	}

	schedule, err := h.service.GetSchedule(scheduleID)
	if err != nil {
		slog.Error("Failed to get schedule", "id", scheduleID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   schedule,
	})
}

// UpdateSchedule handles PUT /api/v1/schedule/:id
func (h *ScheduleHandler) UpdateSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	if scheduleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Schedule ID is required",
		})
	}

	var request entity.ScheduleRequest
	if err := c.BodyParser(&request); err != nil {
		slog.Error("Failed to parse schedule request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request format",
		})
	}

	if err := h.service.UpdateSchedule(scheduleID, request); err != nil {
		slog.Error("Failed to update schedule", "id", scheduleID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Schedule updated successfully",
	})
}

// DeleteSchedule handles DELETE /api/v1/schedule/:id
func (h *ScheduleHandler) DeleteSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	if scheduleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Schedule ID is required",
		})
	}

	if err := h.service.DeleteSchedule(scheduleID); err != nil {
		slog.Error("Failed to delete schedule", "id", scheduleID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Schedule deleted successfully",
	})
}

// SetActiveSchedule handles PUT /api/v1/schedule/:id/activate
func (h *ScheduleHandler) SetActiveSchedule(c *fiber.Ctx) error {
	scheduleID := c.Params("id")
	if scheduleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Schedule ID is required",
		})
	}

	if err := h.service.SetActiveSchedule(scheduleID); err != nil {
		slog.Error("Failed to activate schedule", "id", scheduleID, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Schedule activated successfully",
	})
}

// GetTodaySchedule handles GET /api/v1/schedule/active/today
func (h *ScheduleHandler) GetTodaySchedule(c *fiber.Ctx) error {
	activeSchedule, err := h.service.GetTodaySchedule()
	if err != nil {
		slog.Error("Failed to get today's schedule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   activeSchedule,
	})
}

// GetRolloverTasks handles GET /api/v1/schedule/active/rollover
func (h *ScheduleHandler) GetRolloverTasks(c *fiber.Ctx) error {
	// Get current day from query param or use today
	day := strings.ToLower(c.Query("day", ""))
	if day == "" {
		if val := c.Locals("today"); val != nil {
			if localDay, ok := val.(string); ok {
				day = strings.ToLower(localDay)
			}
		}
	}
	if day == "" {
		day = strings.ToLower(time.Now().Weekday().String())
	}

	rollovers, err := h.service.GetRolloverTasks(day)
	if err != nil {
		slog.Error("Failed to get rollover tasks", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"day":            day,
			"rollover_tasks": rollovers,
			"count":          len(rollovers),
		},
	})
}

// ApplySchedule handles POST /api/v1/schedule/apply
func (h *ScheduleHandler) ApplySchedule(c *fiber.Ctx) error {
	if err := h.service.ApplyScheduleToday(); err != nil {
		slog.Error("Failed to apply schedule", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Schedule applied successfully for today",
	})
}
