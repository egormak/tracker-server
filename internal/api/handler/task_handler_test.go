package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type mockTaskService struct {
	getTaskParamsFunc func(taskName string) (entity.TaskParams, error)
	setTaskParamsFunc func(params entity.TaskParams) error
	getDayRecordFunc  func(taskName string) (int, error)
}

func (m *mockTaskService) GetTaskParams(taskName string) (entity.TaskParams, error) {
	if m.getTaskParamsFunc != nil {
		return m.getTaskParamsFunc(taskName)
	}
	return entity.TaskParams{}, nil
}

func (m *mockTaskService) SetTaskParams(params entity.TaskParams) error {
	if m.setTaskParamsFunc != nil {
		return m.setTaskParamsFunc(params)
	}
	return nil
}

func (m *mockTaskService) GetDayTaskRecord(taskName string) (int, error) {
	if m.getDayRecordFunc != nil {
		return m.getDayRecordFunc(taskName)
	}
	return 0, nil
}

func TestTaskHandler_TaskParams(t *testing.T) {
	tests := []struct {
		name           string
		taskName       string
		mockFunc       func(taskName string) (entity.TaskParams, error)
		expectedStatus int
	}{
		{
			name:     "success",
			taskName: "work",
			mockFunc: func(taskName string) (entity.TaskParams, error) {
				return entity.TaskParams{Name: "work", Time: 120, Priority: 1}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "task not found",
			taskName: "unknown",
			mockFunc: func(taskName string) (entity.TaskParams, error) {
				return entity.TaskParams{}, storage.ErrTaskNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "wrapped task not found",
			taskName: "unknown_wrapped",
			mockFunc: func(taskName string) (entity.TaskParams, error) {
				return entity.TaskParams{}, errors.New("task not found: " + storage.ErrTaskNotFound.Error())
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:     "wrapped with %w task not found",
			taskName: "wrapped_not_found",
			mockFunc: func(taskName string) (entity.TaskParams, error) {
				return entity.TaskParams{}, errors.Join(errors.New("prefix"), storage.ErrTaskNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "params old",
			taskName: "old_task",
			mockFunc: func(taskName string) (entity.TaskParams, error) {
				return entity.TaskParams{}, storage.ErrParamsOld
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "internal error",
			taskName: "err_task",
			mockFunc: func(taskName string) (entity.TaskParams, error) {
				return entity.TaskParams{}, errors.New("db error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			h := NewTaskHandler(&mockTaskService{getTaskParamsFunc: tt.mockFunc})
			app.Get("/api/v1/task/params", h.TaskParams)

			req := httptest.NewRequest("GET", "/api/v1/task/params?task_name="+tt.taskName, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestTaskHandler_SetTaskParams(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		mockFunc       func(params entity.TaskParams) error
		expectedStatus int
	}{
		{
			name: "success with time_duration",
			body: map[string]interface{}{
				"name":          "study",
				"time_duration": 60,
				"priority":      2,
			},
			mockFunc: func(params entity.TaskParams) error {
				if params.Name != "study" || params.Time != 60 || params.Priority != 2 {
					return errors.New("params mismatch")
				}
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success with time field alias",
			body: map[string]interface{}{
				"name":     "exercise",
				"time":     45,
				"priority": 3,
			},
			mockFunc: func(params entity.TaskParams) error {
				if params.Name != "exercise" || params.Time != 45 {
					return errors.New("params mismatch")
				}
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty task name",
			body: map[string]interface{}{
				"name":          "",
				"time_duration": 60,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: map[string]interface{}{
				"name":          "study",
				"time_duration": 60,
			},
			mockFunc: func(params entity.TaskParams) error {
				return errors.New("write failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			h := NewTaskHandler(&mockTaskService{setTaskParamsFunc: tt.mockFunc})
			app.Post("/api/v1/task/params", h.SetTaskParams)

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/task/params", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}
