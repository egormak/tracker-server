package record

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (r *Record) GetTaskDayByPercent(c *fiber.Ctx) error {

	percent, err := strconv.Atoi(c.Query("percent"))
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	if err := r.st.CheckIfPlanPercentEmpty(); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	taskNamePlanByPercent, err := r.st.GetTaskNamePlanPercent("plan", percent)
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}
	return c.Status(200).JSON(&fiber.Map{
		"task_name": taskNamePlanByPercent,
	})
}
