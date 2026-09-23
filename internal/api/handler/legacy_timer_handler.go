package handler

import (
	"log/slog"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type LegacyTimerHandler struct {
	st storage.Storage
}

func NewLegacyTimerHandler(st storage.Storage) *LegacyTimerHandler {
	return &LegacyTimerHandler{st: st}
}

// TimerSet sets the legacy countdown timer duration
func (h *LegacyTimerHandler) TimerSet(c *fiber.Ctx) error {
	var body struct {
		Count        int `json:"count"`
		TimeDuration int `json:"time_duration"`
	}

	if err := c.BodyParser(&body); err != nil {
		slog.Error("TimerSet: can't parse body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if body.Count <= 0 && body.TimeDuration > 0 {
		body.Count = body.TimeDuration
	}

	if err := h.st.TimeListSetDB(body.Count); err != nil {
		slog.Error("TimerSet: TimeListSetDB failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Timer was set",
	})
}

// TimerGet retrieves the current legacy countdown timer duration
func (h *LegacyTimerHandler) TimerGet(c *fiber.Ctx) error {
	count, err := h.st.TimeDurationGet()
	if err != nil {
		slog.Error("TimerGet: TimeDurationGet failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"time_duration": count,
		"count":         count,
	})
}

// TimerDel deletes/resets the legacy countdown timer duration
func (h *LegacyTimerHandler) TimerDel(c *fiber.Ctx) error {
	var body struct {
		Count        int `json:"count"`
		TimeDuration int `json:"time_duration"`
	}
	_ = c.BodyParser(&body)

	if body.Count <= 0 && body.TimeDuration > 0 {
		body.Count = body.TimeDuration
	}

	if err := h.st.TimeListDelDB(body.Count); err != nil {
		slog.Error("TimerDel: TimeListDelDB failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Timer was deleted",
	})
}

// TimerGlobalSet sets the legacy global scheduled time
func (h *LegacyTimerHandler) TimerGlobalSet(c *fiber.Ctx) error {
	var body struct {
		TimeGlobal int `json:"time_scheduler"`
	}

	if err := c.BodyParser(&body); err != nil {
		slog.Error("TimerGlobalSet: can't parse body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if err := h.st.TimerGlobalSet(body.TimeGlobal); err != nil {
		slog.Error("TimerGlobalSet: failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Timer set",
	})
}

// TimerGlobalGet retrieves the legacy global scheduled time
func (h *LegacyTimerHandler) TimerGlobalGet(c *fiber.Ctx) error {
	timerGlobal, err := h.st.TimerGlobalGet()
	if err != nil {
		slog.Error("TimerGlobalGet: failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"timer_global": timerGlobal,
	})
}
