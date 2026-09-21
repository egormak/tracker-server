package handler

import (
	"log/slog"
	"tracker-server/internal/notify"

	"github.com/gofiber/fiber/v2"
)

type TelegramHandler struct {
	ntf notify.Notify
}

func NewTelegramHandler(ntf notify.Notify) *TelegramHandler {
	return &TelegramHandler{ntf: ntf}
}

// TelegramSendStart sends a task start notification to Telegram
func (h *TelegramHandler) TelegramSendStart(c *fiber.Ctx) error {
	var body struct {
		TaskName string `json:"task_name"`
	}

	if err := c.BodyParser(&body); err != nil {
		slog.Error("TelegramSendStart: can't parse body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if body.TaskName == "" {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Task name is not set",
		})
	}

	if h.ntf == nil {
		return c.Status(200).JSON(&fiber.Map{
			"status": "accept",
			"msg_id": 0,
		})
	}

	msgID, err := h.ntf.SendMessageStart(body.TaskName)
	if err != nil {
		slog.Error("TelegramSendStart: SendMessageStart failed", "error", err)
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

// TelegramSendStop sends a task completion notification to Telegram
func (h *TelegramHandler) TelegramSendStop(c *fiber.Ctx) error {
	var body struct {
		TaskName string `json:"task_name"`
		MsgID    int    `json:"msg_id"`
		TimeDone int    `json:"time_done"`
		TimeEnd  string `json:"time_end"`
	}

	if err := c.BodyParser(&body); err != nil {
		slog.Error("TelegramSendStop: can't parse body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if body.TaskName == "" {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Task name is not set",
		})
	}

	slog.Info("TelegramSendStop", "task_name", body.TaskName, "time_done", body.TimeDone, "msg_id", body.MsgID)

	if h.ntf != nil {
		if err := h.ntf.SendMessageStop(body.TaskName, body.TimeDone, body.MsgID, body.TimeEnd); err != nil {
			slog.Error("TelegramSendStop: SendMessageStop failed", "error", err)
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Message Telegram Stop send",
	})
}

// TelegramSendCustom sends an arbitrary notification message to Telegram
func (h *TelegramHandler) TelegramSendCustom(c *fiber.Ctx) error {
	var body struct {
		Message string `json:"message"`
	}

	if err := c.BodyParser(&body); err != nil {
		slog.Error("TelegramSendCustom: can't parse body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	if body.Message == "" {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Message is empty",
		})
	}

	if h.ntf != nil {
		if err := h.ntf.SendCustomMessage(body.Message); err != nil {
			slog.Error("TelegramSendCustom: SendCustomMessage failed", "error", err)
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		}
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Message send to telegram",
	})
}
