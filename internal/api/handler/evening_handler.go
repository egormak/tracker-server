package handler

import (
	"log/slog"
	"strconv"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type EveningHandler struct {
	srv *services.EveningService
}

func NewEveningHandler(srv *services.EveningService) *EveningHandler {
	return &EveningHandler{srv: srv}
}

// GetEveningFocus handles GET /api/v1/mode/evening-focus
func (h *EveningHandler) GetEveningFocus(c *fiber.Ctx) error {
	category := c.Query("category")
	timeStr := c.Query("time")

	timeOverride := 20
	if timeStr != "" {
		if t, err := strconv.Atoi(timeStr); err == nil && t > 0 {
			timeOverride = t
		}
	}

	res, err := h.srv.GetEveningFocus(category, timeOverride)
	if err != nil {
		slog.Error("GetEveningFocus handler error", "err", err)
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"status": "success",
		"data":   res,
	})
}

// SkipTask handles POST /api/v1/mode/evening-focus/skip
func (h *EveningHandler) SkipTask(c *fiber.Ctx) error {
	var req entity.EveningFocusSkipRequest
	if err := c.BodyParser(&req); err != nil {
		// Try query param if body is empty
		req.TaskName = c.Query("task_name")
	}

	if req.TaskName == "" {
		return c.Status(400).JSON(fiber.Map{
			"status":  "error",
			"message": "task_name is required",
		})
	}

	if err := h.srv.SkipTask(req.TaskName); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Immediately return updated candidate queue
	category := c.Query("category")
	timeStr := c.Query("time")
	timeOverride := 20
	if timeStr != "" {
		if t, err := strconv.Atoi(timeStr); err == nil && t > 0 {
			timeOverride = t
		}
	}

	res, err := h.srv.GetEveningFocus(category, timeOverride)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"status": "success",
		"data":   res,
	})
}
