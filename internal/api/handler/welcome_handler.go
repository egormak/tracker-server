package handler

import "github.com/gofiber/fiber/v2"

// Welcome returns a welcome message for the API root
func Welcome(c *fiber.Ctx) error {
	return c.SendString("Welcome to Tracker Server API")
}
