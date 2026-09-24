package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type mockRampServiceForHandler struct {
	status entity.RampStatus
	config entity.RampConfig

	getStatusErr    error
	getDurationVal  int
	getDurationErr  error
	resetErr        error
	advanceErr      error
	getConfigErr    error
	updateConfigErr error

	resetCalled   bool
	advanceCalled bool
	lastConfig    entity.RampConfig
}

func (m *mockRampServiceForHandler) GetStatus(ctx context.Context) (entity.RampStatus, error) {
	if m.getStatusErr != nil {
		return entity.RampStatus{}, m.getStatusErr
	}
	return m.status, nil
}

func (m *mockRampServiceForHandler) GetDurationForTask(ctx context.Context, taskName string) (int, error) {
	if m.getDurationErr != nil {
		return 0, m.getDurationErr
	}
	return m.getDurationVal, nil
}

func (m *mockRampServiceForHandler) Reset(ctx context.Context) (entity.RampStatus, error) {
	if m.resetErr != nil {
		return entity.RampStatus{}, m.resetErr
	}
	m.resetCalled = true
	m.status.CurrentStep = 1
	return m.status, nil
}

func (m *mockRampServiceForHandler) Advance(ctx context.Context) (entity.RampStatus, error) {
	if m.advanceErr != nil {
		return entity.RampStatus{}, m.advanceErr
	}
	m.advanceCalled = true
	m.status.CurrentStep++
	return m.status, nil
}

func (m *mockRampServiceForHandler) GetConfig(ctx context.Context) (entity.RampConfig, error) {
	if m.getConfigErr != nil {
		return entity.RampConfig{}, m.getConfigErr
	}
	return m.config, nil
}

func (m *mockRampServiceForHandler) UpdateConfig(ctx context.Context, cfg entity.RampConfig) (entity.RampStatus, error) {
	if m.updateConfigErr != nil {
		return entity.RampStatus{}, m.updateConfigErr
	}
	m.lastConfig = cfg
	m.status.Config = cfg
	m.status.CapMinutes = cfg.CapMinutes
	return m.status, nil
}

func setupRampApp(srv RampService) *fiber.App {
	app := fiber.New()
	h := NewRampHandler(srv)
	app.Get("/api/v1/ramp/status", h.GetStatus)
	app.Post("/api/v1/ramp/reset", h.Reset)
	app.Post("/api/v1/ramp/advance", h.Advance)
	app.Get("/api/v1/ramp/config", h.GetConfig)
	app.Put("/api/v1/ramp/config", h.UpdateConfig)
	return app
}

func TestRampHandler_GetStatus(t *testing.T) {
	mockSvc := &mockRampServiceForHandler{
		status: entity.RampStatus{
			CurrentStep:       3,
			CapMinutes:        25,
			IsCapped:          false,
			TodayFocusMinutes: 6,
			Date:              "23 September 2026",
			Config: entity.RampConfig{
				CapMinutes:          25,
				EnabledRoles:        []string{"work", "learn"},
				EnabledTasks:        []string{"home_task"},
				ExcludedTasks:       []string{"video"},
				DefaultRestFallback: 15,
			},
		},
	}
	app := setupRampApp(mockSvc)

	t.Run("success returns 200 with RampStatus", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ramp/status", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var body entity.RampStatus
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if body.CurrentStep != 3 || body.CapMinutes != 25 {
			t.Errorf("unexpected status body: %+v", body)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockSvc.getStatusErr = errors.New("db failure")
		req := httptest.NewRequest("GET", "/api/v1/ramp/status", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected 500, got %d", resp.StatusCode)
		}
		mockSvc.getStatusErr = nil
	})
}

func TestRampHandler_Reset(t *testing.T) {
	mockSvc := &mockRampServiceForHandler{
		status: entity.RampStatus{CurrentStep: 7, CapMinutes: 25},
	}
	app := setupRampApp(mockSvc)

	t.Run("success returns 200", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/ramp/reset", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if !mockSvc.resetCalled {
			t.Errorf("expected Reset to be called on service")
		}

		var body entity.RampStatus
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.CurrentStep != 1 {
			t.Errorf("expected CurrentStep 1, got %d", body.CurrentStep)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockSvc.resetErr = errors.New("reset failed")
		req := httptest.NewRequest("POST", "/api/v1/ramp/reset", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected 500, got %d", resp.StatusCode)
		}
		mockSvc.resetErr = nil
	})
}

func TestRampHandler_Advance(t *testing.T) {
	mockSvc := &mockRampServiceForHandler{
		status: entity.RampStatus{CurrentStep: 3, CapMinutes: 25},
	}
	app := setupRampApp(mockSvc)

	t.Run("success returns 200", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/ramp/advance", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if !mockSvc.advanceCalled {
			t.Errorf("expected Advance to be called on service")
		}

		var body entity.RampStatus
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.CurrentStep != 4 {
			t.Errorf("expected CurrentStep 4, got %d", body.CurrentStep)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockSvc.advanceErr = errors.New("advance failed")
		req := httptest.NewRequest("POST", "/api/v1/ramp/advance", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected 500, got %d", resp.StatusCode)
		}
		mockSvc.advanceErr = nil
	})
}

func TestRampHandler_GetConfig(t *testing.T) {
	mockSvc := &mockRampServiceForHandler{
		config: entity.RampConfig{
			CapMinutes:          20,
			EnabledRoles:        []string{"work"},
			EnabledTasks:        []string{"home_task"},
			ExcludedTasks:       []string{"video"},
			DefaultRestFallback: 10,
		},
	}
	app := setupRampApp(mockSvc)

	t.Run("success returns 200 with RampConfig", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/ramp/config", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var cfg entity.RampConfig
		if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if cfg.CapMinutes != 20 || cfg.DefaultRestFallback != 10 {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockSvc.getConfigErr = errors.New("config load failed")
		req := httptest.NewRequest("GET", "/api/v1/ramp/config", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected 500, got %d", resp.StatusCode)
		}
		mockSvc.getConfigErr = nil
	})
}

func TestRampHandler_UpdateConfig(t *testing.T) {
	mockSvc := &mockRampServiceForHandler{
		status: entity.RampStatus{CurrentStep: 5, CapMinutes: 25},
	}
	app := setupRampApp(mockSvc)

	t.Run("valid config returns 200", func(t *testing.T) {
		cfg := entity.RampConfig{
			CapMinutes:          30,
			EnabledRoles:        []string{"work", "learn"},
			EnabledTasks:        []string{"home_task"},
			ExcludedTasks:       []string{"games"},
			DefaultRestFallback: 12,
		}
		bodyBytes, _ := json.Marshal(cfg)
		req := httptest.NewRequest("PUT", "/api/v1/ramp/config", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if mockSvc.lastConfig.CapMinutes != 30 {
			t.Errorf("expected CapMinutes 30 passed to service, got %d", mockSvc.lastConfig.CapMinutes)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/ramp/config", bytes.NewReader([]byte("{invalid-json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("CapMinutes < 5 returns 400", func(t *testing.T) {
		cfg := entity.RampConfig{CapMinutes: 4, DefaultRestFallback: 15}
		bodyBytes, _ := json.Marshal(cfg)
		req := httptest.NewRequest("PUT", "/api/v1/ramp/config", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("DefaultRestFallback < 1 returns 400", func(t *testing.T) {
		cfg := entity.RampConfig{CapMinutes: 20, DefaultRestFallback: 0}
		bodyBytes, _ := json.Marshal(cfg)
		req := httptest.NewRequest("PUT", "/api/v1/ramp/config", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		mockSvc.updateConfigErr = errors.New("failed update in service")
		cfg := entity.RampConfig{CapMinutes: 20, DefaultRestFallback: 15}
		bodyBytes, _ := json.Marshal(cfg)
		req := httptest.NewRequest("PUT", "/api/v1/ramp/config", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected 500, got %d", resp.StatusCode)
		}
		mockSvc.updateConfigErr = nil
	})
}
