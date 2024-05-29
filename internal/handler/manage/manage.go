package manage

import (
	"fmt"
	"log/slog"
	"tracker-server/internal/notify"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

type Manage struct {
	st  storage.Storage
	ntf notify.Notify
}

func New(st storage.Storage, ntf notify.Notify) *Manage {
	return &Manage{st: st, ntf: ntf}
}

func (m *Manage) ProcentsSet(c *fiber.Ctx) error {

	log.Info("Get request ProcentsSet")

	var body struct {
		Procents []int  `json:"procents"`
		RoleName string `json:"role_name"`
	}

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	procentM, err := m.st.GetPlanProcents()
	if err != nil {
		if err == storage.ErrListEmpty {
			procentM = storage.Procents{}
		} else {
			slog.Error("Error getting plan procents", "err", err)
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
	}

	if body.RoleName != "" {
		switch body.RoleName {
		case "plan":
			procentM.Plan = body.Procents
		case "work":
			procentM.Work = body.Procents
		case "learn":
			procentM.Learn = body.Procents
		case "rest":
			procentM.Rest = body.Procents
		}
	} else {
		procentM.Plan = body.Procents
		procentM.Work = body.Procents
		procentM.Learn = body.Procents
		procentM.Rest = body.Procents
	}

	if err := m.st.ProcentsSet(procentM); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Procent set",
	})
}
