package record

import (
	"log/slog"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

func (r *Record) GetTaskPlanPercent(c *fiber.Ctx) error {

	slog.Info("Get request GetTaskPlanPercent")

	var TaskNamePlanPercent string
	var GroupPercent int

	for {
		GroupPlanOrdinal, err := r.st.GetGroupPlanPercent()
		if err != nil {
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		GroupPercent, err = r.st.GetGroupPercent(GroupPlanOrdinal)
		if err != nil {
			if err == storage.ErrListEmpty {
				r.st.ChangeGroupPlanPercent(GroupPlanOrdinal)
			} else {
				return c.Status(500).JSON(&fiber.Map{
					"status":  "error",
					"message": err.Error(),
				})
			}
		}
		groupName, err := r.st.GetGroupName(GroupPlanOrdinal)
		if err != nil {
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		TaskNamePlanPercent, err = r.st.GetTaskNamePlanPercent(groupName, GroupPercent)
		if err != nil {
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
		if TaskNamePlanPercent != "" {
			break
		} else if TaskNamePlanPercent == "" {
			if err := r.st.CheckIfPlanPercentEmpty(); err != nil {
				return c.Status(404).JSON(&fiber.Map{
					"status":  "status",
					"message": err.Error(),
				})
			}
			if err := r.st.DelGroupPercent(groupName); err != nil {
				return c.Status(500).JSON(&fiber.Map{
					"status":  "error",
					"message": err.Error(),
				})
			}
		}
	}

	slog.Info("Sent answer", "request", "GetTaskPlanPercent", "task_name", TaskNamePlanPercent, "percent", GroupPercent)

	return c.Status(200).JSON(&fiber.Map{
		"status":    "accept",
		"task_name": TaskNamePlanPercent,
		"percent":   GroupPercent,
	})
}

func (r *Record) ChangeGroupPlanPercent(c *fiber.Ctx) error {

	GroupPlan, err := r.st.GetGroupPlanPercent()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if err := r.st.ChangeGroupPlanPercent(GroupPlan); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status": "accept",
	})
}
