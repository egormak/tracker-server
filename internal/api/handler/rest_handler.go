package handler

import (
	"log/slog"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type RestService interface {
	RestSpend(restTime int) error
	AddRest(restTime int) error
	RestGet() (int, error)
	ResetRest() error
}

type RestHandler struct {
	srv RestService
}

func NewRestHandler(srv RestService) *RestHandler {
	return &RestHandler{srv: srv}
}

func (r *RestHandler) RestSpend(c *fiber.Ctx) error {
	// Receive JSON body and store it in the 'body' variable
	var body entity.RestRecordRequest
	if err := c.BodyParser(&body); err != nil {
		slog.Error("Failed to parse request body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	// Validate restTime
	if body.RestTime <= 0 {
		slog.Error("Invalid rest time", "rest_time", body.RestTime)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Rest time must be a positive integer",
		})
	}

	// Call the 'RestSpend' function from the 'database' package
	// with the 'restTime' value from the 'body' map
	if err := r.srv.RestSpend(body.RestTime); err != nil {
		slog.Error("Failed to spend rest time", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Failed to spend rest time",
		})
	}

	// Return a JSON response with the status and message
	return c.JSON(fiber.Map{
		"status":  "accept",
		"message": "Rest was spent",
	})
}

func (r *RestHandler) RestAdd(c *fiber.Ctx) error {
	var body entity.RestRecordRequest
	if err := c.BodyParser(&body); err != nil {
		slog.Error("Failed to parse request body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	if body.RestTime <= 0 {
		slog.Error("Invalid rest time", "rest_time", body.RestTime)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Rest time must be a positive integer",
		})
	}

	slog.Info("Get request RestAdd", "rest_time", body.RestTime)
	if err := r.srv.AddRest(body.RestTime); err != nil {
		slog.Error("Failed to add rest time", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": "Failed to add rest time",
		})
	}

	// Return a JSON response with the status and message
	return c.JSON(fiber.Map{
		"status":  "accept",
		"message": "Rest was Add",
	})
}

func (r *RestHandler) RestGet(c *fiber.Ctx) error {

	// Call the 'RestSpend' function from the 'database' package
	// with the 'restTime' value from the 'body' map
	restTime, err := r.srv.RestGet()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Return a JSON response with the status and message
	return c.Status(200).JSON(fiber.Map{
		"rest_time": restTime,
	})
}

func (r *RestHandler) RestReset(c *fiber.Ctx) error {
	if err := r.srv.ResetRest(); err != nil {
		slog.Error("Failed to reset rest time", "error", err)
		return c.Status(500).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to reset rest time",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "accept",
		"message": "Rest was reset",
	})
}
