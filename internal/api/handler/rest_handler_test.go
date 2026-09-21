package handler

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type mockRestHandlerService struct {
	spendErr error
	addErr   error
	getVal   int
	getErr   error
	resetErr error
}

func (m *mockRestHandlerService) RestSpend(restTime int) error { return m.spendErr }
func (m *mockRestHandlerService) AddRest(restTime int) error   { return m.addErr }
func (m *mockRestHandlerService) RestGet() (int, error)        { return m.getVal, m.getErr }
func (m *mockRestHandlerService) ResetRest() error             { return m.resetErr }

func TestRestHandler_RestReset(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockRestHandlerService{}
	h := NewRestHandler(mockSvc)
	app.Post("/api/v1/rest/reset", h.RestReset)

	t.Run("success", func(t *testing.T) {
		mockSvc.resetErr = nil
		req := httptest.NewRequest("POST", "/api/v1/rest/reset", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("server error", func(t *testing.T) {
		mockSvc.resetErr = errors.New("failed to reset")
		req := httptest.NewRequest("POST", "/api/v1/rest/reset", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.StatusCode)
		}
	})
}

func TestRestHandler_RestAdd(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockRestHandlerService{}
	h := NewRestHandler(mockSvc)
	app.Post("/api/v1/rest/add", h.RestAdd)

	t.Run("invalid rest time <= 0", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/rest/add", strings.NewReader(`{"rest_time": 0}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid json body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/rest/add", strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/rest/add", strings.NewReader(`{"rest_time": 15}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}
