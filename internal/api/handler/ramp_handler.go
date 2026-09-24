package handler

import (
	"context"
	"log/slog"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

// RampService defines the service contract needed by RampHandler.
type RampService interface {
	GetStatus(ctx context.Context) (entity.RampStatus, error)
	GetDurationForTask(ctx context.Context, taskName string) (int, error)
	Reset(ctx context.Context) (entity.RampStatus, error)
	Advance(ctx context.Context) (entity.RampStatus, error)
	GetConfig(ctx context.Context) (entity.RampConfig, error)
	UpdateConfig(ctx context.Context, cfg entity.RampConfig) (entity.RampStatus, error)
}

// RampHandler handles HTTP requests for the linear warm-up ramp.
type RampHandler struct {
	srv RampService
}

// NewRampHandler creates a new RampHandler.
func NewRampHandler(srv RampService) *RampHandler {
	return &RampHandler{srv: srv}
}

// GetStatus handles GET /api/v1/ramp/status.
func (h *RampHandler) GetStatus(c *fiber.Ctx) error {
	status, err := h.srv.GetStatus(c.UserContext())
	if err != nil {
		slog.Error("Failed to get ramp status", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	return c.JSON(status)
}

// Reset handles POST /api/v1/ramp/reset.
func (h *RampHandler) Reset(c *fiber.Ctx) error {
	status, err := h.srv.Reset(c.UserContext())
	if err != nil {
		slog.Error("Failed to reset ramp", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	return c.JSON(status)
}

// Advance handles POST /api/v1/ramp/advance.
func (h *RampHandler) Advance(c *fiber.Ctx) error {
	status, err := h.srv.Advance(c.UserContext())
	if err != nil {
		slog.Error("Failed to advance ramp", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	return c.JSON(status)
}

// GetConfig handles GET /api/v1/ramp/config.
func (h *RampHandler) GetConfig(c *fiber.Ctx) error {
	cfg, err := h.srv.GetConfig(c.UserContext())
	if err != nil {
		slog.Error("Failed to get ramp config", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	return c.JSON(cfg)
}

// UpdateConfig handles PUT /api/v1/ramp/config.
func (h *RampHandler) UpdateConfig(c *fiber.Ctx) error {
	var cfg entity.RampConfig
	if err := c.BodyParser(&cfg); err != nil {
		slog.Error("Failed to parse ramp config body", "err", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	if cfg.CapMinutes < 5 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "cap_minutes must be at least 5",
		})
	}
	if cfg.DefaultRestFallback < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "default_rest_fallback must be at least 1",
		})
	}

	status, err := h.srv.UpdateConfig(c.UserContext(), cfg)
	if err != nil {
		slog.Error("Failed to update ramp config", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(status)
}
