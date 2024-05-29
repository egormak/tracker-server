package role

import (
	"time"
	"tracker-server/internal/notify"
	"tracker-server/internal/service"
	"tracker-server/internal/storage"

	log "github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
)

type Role struct {
	st  storage.Storage
	ntf notify.Notify
}

func New(st storage.Storage, ntf notify.Notify) *Role {
	return &Role{st: st, ntf: ntf}
}

func (r *Role) ShowRolesRecords(c *fiber.Ctx) error {

	records, err := r.st.StatisticRolesGet()
	var result = make(map[string]int)

	if err != nil {
		c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	for _, v := range records {
		result[v.Name] = v.Duration
	}

	// var answer = make(map[string]map[string]int)
	// answer["role_records"] = result

	return c.JSON(result)

}

func (r *Role) StatCompletionTimeDone(c *fiber.Ctx) error {

	rolesData, err := r.st.StatisticRolesGetToday()
	var timeDone int

	if err != nil {
		c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Count Time Done
	for _, roleData := range rolesData {
		if roleData.RecordDate == time.Now().Format("2 January 2006") {
			if service.IsWeekendNow() || roleData.Name != "rest" {
				timeDone += roleData.DurationToday
			}
		}
	}

	// var answer = make(map[string]map[string]int)
	// answer["role_records"] = result

	return c.Status(200).JSON(&fiber.Map{
		"time_done": timeDone,
	})

}

func (r *Role) RecheckRole(c *fiber.Ctx) error {
	// Log the start of the Recheck Role operation
	log.Info("Run Recheck Role")

	// Call the RecheckRole function from the database package
	if err := r.st.RecheckRole(); err != nil {
		// If there is an error, log the error message and return an error response
		log.Error("Error in Recheck Role: ", err.Error())
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Log the end of the Recheck Role operation
	log.Info("End Recheck Role")

	// Return a success response
	return c.JSON(&fiber.Map{
		"status":  "accept",
		"message": "RecheckRole was done",
	})
}

func (r *Role) TaskRoleGet(c *fiber.Ctx) error {

	taskName := c.Query("task_name")

	result, err := r.st.GetRole(taskName)
	if err != nil {
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
