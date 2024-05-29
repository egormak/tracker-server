package manage

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

func (m *Manage) TelegramSendStart(c *fiber.Ctx) error {

	slog.Info("Get request TelegramSend")

	var body struct {
		TaskName string `json:"task_name"`
	}

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		slog.Error("can't parse body", "error", errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if body.TaskName == "" {
		errMsg := "Task name is not Set"
		slog.Error("TaskName is not Set", "error", errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg,
		})
	}

	msgID, err := m.ntf.SendMessageStart(body.TaskName)
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status": "accept",
		"msg_id": msgID,
	})
}

func (m *Manage) TelegramSendStop(c *fiber.Ctx) error {

	slog.Info("Get request TelegramSend")

	var body struct {
		TaskName string `json:"task_name"`
		MsgID    int    `json:"msg_id"`
		TimeDone int    `json:"time_done"`
		TimeEnd  string `json:"time_end"`
	}

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		slog.Error("can't parse body", "error", errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if body.TaskName == "" {
		errMsg := "Task name is not Set"
		slog.Error("TaskName is not Set", "error", errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg,
		})
	}

	slog.Info("Send Telegram Stop", "task_name", body.TaskName, "time_done", body.TimeDone, "msg_id", body.MsgID)

	if err := m.ntf.SendMessageStop(body.TaskName, body.TimeDone, body.MsgID, body.TimeEnd); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Message Telegram Stop send",
	})
}
