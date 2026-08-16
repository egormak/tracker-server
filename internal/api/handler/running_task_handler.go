package handler

import (
	"encoding/json"
	"log/slog"
	"time"
	"tracker-server/internal/realtime"
	"tracker-server/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type RunningTaskHandler struct {
	service *services.RunningTaskService
	hub     *realtime.Hub
}

func NewRunningTaskHandler(service *services.RunningTaskService, hub *realtime.Hub) *RunningTaskHandler {
	return &RunningTaskHandler{service: service, hub: hub}
}

func (h *RunningTaskHandler) Start(c *fiber.Ctx) error {
	var body struct {
		TaskName       string `json:"task_name"`
		Role           string `json:"role"`
		TargetDuration int    `json:"target_duration"`
		SourceDay      string `json:"source_day"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	slog.Info("RunningTaskHandler.Start", "task", body.TaskName, "role", body.Role, "target_duration", body.TargetDuration, "source_day", body.SourceDay)

	task, err := h.service.Start(body.TaskName, body.Role, body.TargetDuration, body.SourceDay)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        task,
	})
}

func (h *RunningTaskHandler) Stop(c *fiber.Ctx) error {
	slog.Info("RunningTaskHandler.Stop")

	var body struct {
		TaskName string `json:"task_name"`
		Reason   string `json:"reason"`
	}
	_ = c.BodyParser(&body)

	record, err := h.service.Stop(body.TaskName, body.Reason)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        record,
	})
}

func (h *RunningTaskHandler) Pause(c *fiber.Ctx) error {
	slog.Info("RunningTaskHandler.Pause")

	var body struct {
		TaskName string `json:"task_name"`
		Reason   string `json:"reason"`
	}
	_ = c.BodyParser(&body)

	task, err := h.service.Pause(body.TaskName, body.Reason)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        task,
	})
}

func (h *RunningTaskHandler) Resume(c *fiber.Ctx) error {
	slog.Info("RunningTaskHandler.Resume")

	var body struct {
		TaskName string `json:"task_name"`
	}
	_ = c.BodyParser(&body)

	task, err := h.service.Resume(body.TaskName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        task,
	})
}

func (h *RunningTaskHandler) Status(c *fiber.Ctx) error {
	taskName := c.Query("task_name")
	task, err := h.service.GetStatus(taskName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        task,
	})
}

func (h *RunningTaskHandler) List(c *fiber.Ctx) error {
	tasks, err := h.service.GetAllTasks()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        tasks,
	})
}

func (h *RunningTaskHandler) Heartbeat(c *fiber.Ctx) error {
	var body struct {
		TaskName string `json:"task_name"`
	}
	_ = c.BodyParser(&body)

	task, err := h.service.Heartbeat(body.TaskName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{
		"status":      "success",
		"server_time": time.Now().UnixMilli(),
		"data":        task,
	})
}

func (h *RunningTaskHandler) WebSocketUpgradeCheck(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *RunningTaskHandler) WebSocketHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		if h.hub == nil {
			c.Close()
			return
		}

		h.hub.Register(c)
		defer func() {
			h.hub.Unregister(c)
		}()

		// Send initial state sync immediately upon connection
		tasks, _ := h.service.GetAllTasks()
		_ = c.WriteJSON(realtime.Event{
			Type:       realtime.EventStateSync,
			ServerTime: time.Now().UnixMilli(),
			Data:       tasks,
		})

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				break
			}

			var msg struct {
				Type     string `json:"type"`
				TaskName string `json:"task_name"`
			}
			if json.Unmarshal(message, &msg) == nil {
				if msg.Type == "heartbeat" || msg.Type == "HEARTBEAT" {
					task, _ := h.service.Heartbeat(msg.TaskName)
					_ = c.WriteJSON(realtime.Event{
						Type:       realtime.EventHeartbeatAck,
						TaskName:   task.TaskName,
						Role:       task.Role,
						ServerTime: time.Now().UnixMilli(),
						Data:       task,
					})
				}
			}
		}
	})
}
