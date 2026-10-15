package handler

import (
	"log/slog"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type DebtReader interface {
	Summary() (entity.DebtSummary, error)
}

type DebtHandler struct {
	srv DebtReader
}

func NewDebtHandler(srv DebtReader) *DebtHandler {
	return &DebtHandler{srv: srv}
}

// GetDebt handles GET /api/v1/debt: the previous days' unfinished schedule of this week,
// today's remainder and the week strip.
func (h *DebtHandler) GetDebt(c *fiber.Ctx) error {
	summary, err := h.srv.Summary()
	if err != nil {
		slog.Error("GetDebt handler error", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   summary,
	})
}
