package rest

import (
	"log/slog"
	"tracker-server/internal/notify"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type Rest struct {
	st  storage.Storage
	ntf notify.Notify
}

type RestHandler interface {
	RestSpend(restTime int) error
}

func New(st storage.Storage, ntf notify.Notify) *Rest {
	return &Rest{st: st, ntf: ntf}
}

func (r *Rest) RestSpend(c *fiber.Ctx) error {
	// Receive JSON body and store it in the 'body' variable
	var body map[string]int
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	// Call the 'RestSpend' function from the 'database' package
	// with the 'restTime' value from the 'body' map
	if err := r.st.RestSpend(body["rest_time"]); err != nil {
		return err
	}

	// Return a JSON response with the status and message
	return c.JSON(fiber.Map{
		"status":  "accept",
		"message": "Rest was spend",
	})
}

func (r *Rest) RestAdd(c *fiber.Ctx) error {

	// Receive JSON body and store it in the 'body' variable
	var body map[string]int
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	slog.Info("Get request RestAdd", "rest_time", body["rest_time"])
	// Call the 'RestSpend' function from the 'database' package
	// with the 'restTime' value from the 'body' map
	if err := r.st.AddRest(body["rest_time"]); err != nil {
		return err
	}

	// Return a JSON response with the status and message
	return c.JSON(fiber.Map{
		"status":  "accept",
		"message": "Rest was Add",
	})
}

func (r *Rest) RestGet(c *fiber.Ctx) error {

	// Call the 'RestSpend' function from the 'database' package
	// with the 'restTime' value from the 'body' map
	restTime, err := r.st.GetRest()
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
