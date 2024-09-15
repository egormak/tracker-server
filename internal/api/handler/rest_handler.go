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
		return err
	}

	// Call the 'RestSpend' function from the 'database' package
	// with the 'restTime' value from the 'body' map
	if err := r.srv.RestSpend(body.RestTime); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err,
		})
	}

	// Return a JSON response with the status and message
	return c.JSON(fiber.Map{
		"status":  "accept",
		"message": "Rest was spend",
	})
}

func (r *RestHandler) RestAdd(c *fiber.Ctx) error {

	// Receive JSON body and store it in the 'body' variable
	var body entity.RestRecordRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	slog.Info("Get request RestAdd", "rest_time", body.RestTime)
	if err := r.srv.AddRest(body.RestTime); err != nil {
		return err
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
