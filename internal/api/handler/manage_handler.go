package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/services"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

// ManageService defines the interface for the manage service
type ManageService interface {
	CreateTaskWithRole(taskName string, role string) error
	GetPlanPercents() (*entity.PlanPercents, error)
	SetPlanPercents(procentM entity.PlanPercents) error
	RemovePlanPercent(group string, value int) error
}

// ManageHandler handles task management operations
type ManageHandler struct {
	srv ManageService
}

// NewManageHandler creates a new instance of ManageHandler
func NewManageHandler(srv ManageService) *ManageHandler {
	return &ManageHandler{srv: srv}
}

// CreateTask handles the creation of new tasks
func (m *ManageHandler) CreateTask(c *fiber.Ctx) error {
	var request struct {
		TaskName string `json:"task_name"`
		Role     string `json:"role"`
	}

	if err := c.BodyParser(&request); err != nil {
		slog.Error("Failed to parse request body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Invalid request format",
		})
	}

	// Validate request data
	if request.TaskName == "" {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Task name is required",
		})
	}

	if request.Role == "" {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Role is required",
		})
	}

	// Call service to create task
	err := m.srv.CreateTaskWithRole(request.TaskName, request.Role)
	if err != nil {
		slog.Error("Failed to create task", "error", err, "task_name", request.TaskName)

		// Check for specific errors
		switch err {
		case storage.ErrTaskNotFound:
			return c.Status(404).JSON(&fiber.Map{
				"status":  "error",
				"message": "Task not found",
			})
		default:
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Failed to create task: %v", err),
			})
		}
	}

	slog.Info("Task created successfully", "task_name", request.TaskName)
	return c.Status(201).JSON(&fiber.Map{
		"status":  "success",
		"message": "Task created successfully",
	})
}

// GetPlanPercents handles retrieving the plan percent values for plan, work, learn and rest
func (m *ManageHandler) GetPlanPercents(c *fiber.Ctx) error {
	slog.Info("Request received: Get plan percents")

	percents, err := m.srv.GetPlanPercents()
	if err != nil {
		slog.Error("Failed to get plan percents", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to get plan percents: %v", err),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status": "success",
		"data":   percents,
	})
}

// DeletePlanPercent handles removing a specific percent value from a plan group
func (m *ManageHandler) DeletePlanPercent(c *fiber.Ctx) error {
	group := c.Params("group")
	valueStr := c.Params("value")

	if group == "" || valueStr == "" {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Group and value are required",
		})
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": "Value must be a number",
		})
	}

	if err := m.srv.RemovePlanPercent(group, value); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidPlanPercentGroup), errors.Is(err, services.ErrInvalidPlanPercentValue):
			return c.Status(400).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		case errors.Is(err, services.ErrPlanPercentValueNotFound):
			return c.Status(404).JSON(&fiber.Map{
				"status":  "error",
				"message": err.Error(),
			})
		default:
			slog.Error("Failed to remove plan percent", "error", err, "group", group, "value", value)
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Failed to remove plan percent: %v", err),
			})
		}
	}

	slog.Info("Removed plan percent", "group", group, "value", value)
	return c.Status(200).JSON(&fiber.Map{
		"status":  "success",
		"message": "Plan percent removed",
	})
}

// ProcentsSet sets the percentage distribution for a plan role or all roles
func (m *ManageHandler) ProcentsSet(c *fiber.Ctx) error {
	var body struct {
		Procents []int  `json:"procents"`
		RoleName string `json:"role_name"`
	}

	if err := c.BodyParser(&body); err != nil {
		slog.Error("ProcentsSet: can't parse body", "error", err)
		return c.Status(400).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	procentM, err := m.srv.GetPlanPercents()
	var currentPercents entity.PlanPercents
	if err == nil && procentM != nil {
		currentPercents = *procentM
	}

	if body.RoleName != "" {
		switch strings.ToLower(body.RoleName) {
		case "plan":
			currentPercents.Plan = body.Procents
		case "work":
			currentPercents.Work = body.Procents
		case "learn":
			currentPercents.Learn = body.Procents
		case "rest":
			currentPercents.Rest = body.Procents
		}
	} else {
		currentPercents.Plan = body.Procents
		currentPercents.Work = body.Procents
		currentPercents.Learn = body.Procents
		currentPercents.Rest = body.Procents
	}

	if err := m.srv.SetPlanPercents(currentPercents); err != nil {
		slog.Error("ProcentsSet: failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Plan procents was updated",
	})
}

// GetPlanProcentsLegacy returns plan percents in the legacy structure
func (m *ManageHandler) GetPlanProcentsLegacy(c *fiber.Ctx) error {
	procents, err := m.srv.GetPlanPercents()
	if err != nil {
		slog.Error("GetPlanProcentsLegacy: failed", "error", err)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"title":          procents.Title,
			"date":           procents.Date,
			"current_choice": procents.CurrentChoice,
			"plans":          procents.Plans,
			"plan":           procents.Plan,
			"work":           procents.Work,
			"learn":          procents.Learn,
			"rest":           procents.Rest,
		},
	})
}
