package handler

import (
	"fmt"
	"log/slog"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

// ManageService defines the interface for the manage service
type ManageService interface {
	CreateTaskWithRole(taskName string, role string) error
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

// // SetTaskParams handles setting task parameters
// func (m *ManageHandler) SetTaskParams(c *fiber.Ctx) error {
// 	var request entity.TaskParams

// 	if err := c.BodyParser(&request); err != nil {
// 		slog.Error("Failed to parse request body", "error", err)
// 		return c.Status(400).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": "Invalid request format",
// 		})
// 	}

// 	// Validate request data
// 	if request.Name == "" {
// 		return c.Status(400).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": "Task name is required",
// 		})
// 	}

// 	// Additional validation could be added here

// 	// This would typically call a service method
// 	// For now, return success
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status":  "success",
// 		"message": "Task parameters set successfully",
// 	})
// }

// // GetTasks retrieves a list of tasks
// func (m *ManageHandler) GetTasks(c *fiber.Ctx) error {
// 	// This would typically call a service method to get tasks
// 	// For now, return a placeholder response
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status": "success",
// 		"tasks":  []string{}, // Empty array as placeholder
// 	})
// }

// // DeleteTask handles task deletion
// func (m *ManageHandler) DeleteTask(c *fiber.Ctx) error {
// 	taskID := c.Params("id")
// 	if taskID == "" {
// 		return c.Status(400).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": "Task ID is required",
// 		})
// 	}

// 	// This would typically call a service method to delete the task
// 	// For now, return success
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status":  "success",
// 		"message": "Task deleted successfully",
// 	})
// }

// // UpdateTask handles task updates
// func (m *ManageHandler) UpdateTask(c *fiber.Ctx) error {
// 	taskID := c.Params("id")
// 	if taskID == "" {
// 		return c.Status(400).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": "Task ID is required",
// 		})
// 	}

// 	var request struct {
// 		TaskName string `json:"task_name"`
// 		Role     string `json:"role"`
// 		Priority int    `json:"priority"`
// 	}

// 	if err := c.BodyParser(&request); err != nil {
// 		return c.Status(400).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": "Invalid request format",
// 		})
// 	}

// 	// This would typically call a service method to update the task
// 	// For now, return success
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status":  "success",
// 		"message": "Task updated successfully",
// 	})
// }

// // RecheckRoles triggers a role rechecking process
// func (m *ManageHandler) RecheckRoles(c *fiber.Ctx) error {
// 	// This would typically call a service method to recheck roles
// 	// For now, return success
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status":  "success",
// 		"message": "Roles rechecked successfully",
// 	})
// }

// // CleanRecords handles cleaning of task records
// func (m *ManageHandler) CleanRecords(c *fiber.Ctx) error {
// 	// This would typically call a service method to clean records
// 	// For now, return success
// 	return c.Status(200).JSON(&fiber.Map{
// 		"status":  "success",
// 		"message": "Records cleaned successfully",
// 	})
// }
