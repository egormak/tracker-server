package manage

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func (m *Manage) TimerRecheck(c *fiber.Ctx) error {

	log.Info("Get request TimerRecheck")

	timeTasks, err := m.st.TimeTasks()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	i := 0
	for timeTasks > 0 {
		i++
		for x := 1; x <= i; x++ {
			timeTasks -= x
		}
	}

	if err := m.st.TimeListSetDB(i); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": fmt.Sprintf("Timer: %d", i),
	})
}

func (m *Manage) TimerGlobalSet(c *fiber.Ctx) error {

	log.Info("Get request TimerGlobalSet")

	var body struct {
		TimeGlobal int `json:"time_scheduler"`
	}

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if err := m.st.TimerGlobalSet(body.TimeGlobal); err != nil {
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

func (m *Manage) TimerGlobalGet(c *fiber.Ctx) error {

	log.Info("Get request TimerGlobalGet")

	timerGlobal, err := m.st.TimerGlobalGet()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"timer_global": timerGlobal,
	})
}

// TimerSet sets the timer value based on the count provided in the request body.
func (m *Manage) TimerSet(c *fiber.Ctx) error {
	// Define a struct to hold the request body
	var body struct {
		Count int `json:"count"`
	}

	// Parse the request body and store it in the 'body' variable
	if err := c.BodyParser(&body); err != nil {
		log.Errorf("Error parsing request body: %v", err)
		return err
	}

	// Log the start of the timer set process
	log.Infof("Start Timer Set")

	// Set the timer value in the database based on the count
	m.st.TimeListSetDB(body.Count)

	// Log the completion of the timer set process
	log.Info("Finish Timer Set")

	// Create a response map with the status and message
	response := fiber.Map{
		"status":  "accept",
		"message": "Timer was set",
	}

	// Log the response being returned
	log.Infof("Returning response: %v", response)

	// Return the response as a JSON with status 200
	return c.Status(200).JSON(&response)
}

func (m *Manage) TimerGet(c *fiber.Ctx) error {
	slog.Info("Get request TimerGet")

	result, err := m.st.TimeDurationGet()
	if err != nil {
		slog.Error("Error getting time duration", "err", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	slog.Info("Returning response", "result", result)

	return c.Status(200).JSON(&fiber.Map{
		"time_duration": result,
	})
}
