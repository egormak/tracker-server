package welcome

import "github.com/gofiber/fiber/v2"

func Welcome(c *fiber.Ctx) error {
	return c.SendString("Hello, World!")
}
