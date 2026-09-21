package handler

import (
	"log/slog"
	"time"
	"tracker-server/internal/notify"
	"tracker-server/internal/services"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type RoleHandler struct {
	st  storage.Storage
	ntf notify.Notify
}

func NewRoleHandler(st storage.Storage, ntf notify.Notify) *RoleHandler {
	return &RoleHandler{st: st, ntf: ntf}
}

// ShowRolesRecords returns the total time recorded for each role
func (r *RoleHandler) ShowRolesRecords(c *fiber.Ctx) error {
	records, err := r.st.StatisticRolesGet()
	if err != nil {
		slog.Error("ShowRolesRecords failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	result := make(map[string]int)
	for _, v := range records {
		result[v.Name] = v.Duration
	}

	return c.JSON(result)
}

// StatCompletionTimeDone returns today's completed time across non-rest roles (or all roles on weekend)
func (r *RoleHandler) StatCompletionTimeDone(c *fiber.Ctx) error {
	rolesData, err := r.st.StatisticRolesGetToday()
	if err != nil {
		slog.Error("StatCompletionTimeDone failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	today := time.Now().Format("2 January 2006")
	var timeDone int
	for _, roleData := range rolesData {
		if roleData.RecordDate == today {
			if services.IsWeekendNow() || roleData.Name != "rest" {
				timeDone += roleData.DurationToday
			}
		}
	}

	return c.Status(200).JSON(&fiber.Map{
		"time_done": timeDone,
	})
}

// RecheckRole triggers role duration re-aggregation in the database
func (r *RoleHandler) RecheckRole(c *fiber.Ctx) error {
	slog.Info("Run Recheck Role")
	if err := r.st.RecheckRole(); err != nil {
		slog.Error("Error in Recheck Role", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(&fiber.Map{
		"status":  "accept",
		"message": "RecheckRole was done",
	})
}

// TaskRoleGet returns the role assigned to a specific task name
func (r *RoleHandler) TaskRoleGet(c *fiber.Ctx) error {
	taskName := c.Query("task_name")
	result, err := r.st.GetRole(taskName)
	if err != nil {
		slog.Error("TaskRoleGet failed", "task_name", taskName, "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(&fiber.Map{
		"status": "accept",
		"role":   result,
	})
}
