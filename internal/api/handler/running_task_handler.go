package handler

import (
	"log/slog"
	"tracker-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type RunningTaskHandler struct {
	service *services.RunningTaskService
}

func NewRunningTaskHandler(service *services.RunningTaskService) *RunningTaskHandler {
	return &RunningTaskHandler{service: service}
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

	return c.Status(200).JSON(fiber.Map{"status": "success", "data": task})
}

func (h *RunningTaskHandler) Stop(c *fiber.Ctx) error {
	slog.Info("RunningTaskHandler.Stop")

	record, err := h.service.Stop()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"status": "success", "data": record})
}

func (h *RunningTaskHandler) Pause(c *fiber.Ctx) error {
	slog.Info("RunningTaskHandler.Pause")

	task, err := h.service.Pause()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"status": "success", "data": task})
}

func (h *RunningTaskHandler) Resume(c *fiber.Ctx) error {
	slog.Info("RunningTaskHandler.Resume")

	task, err := h.service.Resume()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"status": "success", "data": task})
}

func (h *RunningTaskHandler) Status(c *fiber.Ctx) error {
	task, err := h.service.GetStatus()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	return c.Status(200).JSON(fiber.Map{"status": "success", "data": task})
}
