package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type mockLegacyTimerStorage struct {
	storage.Storage

	timeDurationGetVal int
	timeDurationGetErr error

	timeListSetDBErr error
	lastSetCount     int

	timeListDelDBErr error
	lastDelCount     int

	timerGlobalSetErr error
	lastGlobalSet     int

	timerGlobalGetVal int
	timerGlobalGetErr error
}

func (m *mockLegacyTimerStorage) TimeDurationGet() (int, error) {
	return m.timeDurationGetVal, m.timeDurationGetErr
}

func (m *mockLegacyTimerStorage) TimeListSetDB(count int) error {
	m.lastSetCount = count
	return m.timeListSetDBErr
}

func (m *mockLegacyTimerStorage) TimeListDelDB(timeDuration int) error {
	m.lastDelCount = timeDuration
	return m.timeListDelDBErr
}

func (m *mockLegacyTimerStorage) TimerGlobalSet(timeScheduler int) error {
	m.lastGlobalSet = timeScheduler
	return m.timerGlobalSetErr
}

func (m *mockLegacyTimerStorage) TimerGlobalGet() (int, error) {
	return m.timerGlobalGetVal, m.timerGlobalGetErr
}

func setupTimerApp(mock *mockLegacyTimerStorage) *fiber.App {
	app := fiber.New()
	h := NewLegacyTimerHandler(mock)

	app.Post("/api/v1/timer/set", h.TimerSet)
	app.Get("/api/v1/timer/get", h.TimerGet)
	app.Post("/api/v1/timer/del", h.TimerDel)
	app.Post("/api/v1/manage/timer/global", h.TimerGlobalSet)
	app.Get("/api/v1/manage/timer/global", h.TimerGlobalGet)

	return app
}

func TestLegacyTimerHandler_TimerGet(t *testing.T) {
	t.Run("200 OK asserts both time_duration and count", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timeDurationGetVal: 25}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/timer/get", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		td, okTd := body["time_duration"].(float64)
		cnt, okCnt := body["count"].(float64)
		if !okTd || int(td) != 25 {
			t.Errorf("expected time_duration=25, got %v", body["time_duration"])
		}
		if !okCnt || int(cnt) != 25 {
			t.Errorf("expected count=25, got %v", body["count"])
		}
	})

	t.Run("500 error on storage failure", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timeDurationGetErr: errors.New("db error")}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/timer/get", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", resp.StatusCode)
		}
	})
}

func TestLegacyTimerHandler_TimerSet(t *testing.T) {
	t.Run("200 OK with count", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/set", strings.NewReader(`{"count": 30}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if mock.lastSetCount != 30 {
			t.Errorf("expected storage to receive count 30, got %d", mock.lastSetCount)
		}
	})

	t.Run("200 OK with time_duration", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/set", strings.NewReader(`{"time_duration": 45}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if mock.lastSetCount != 45 {
			t.Errorf("expected storage to receive count 45, got %d", mock.lastSetCount)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/set", strings.NewReader(`{"count": "not-a-number"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("500 on storage error", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timeListSetDBErr: errors.New("db error")}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/set", strings.NewReader(`{"count": 20}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", resp.StatusCode)
		}
	})
}

func TestLegacyTimerHandler_TimerDel(t *testing.T) {
	t.Run("200 OK with count", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/del", strings.NewReader(`{"count": 10}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if mock.lastDelCount != 10 {
			t.Errorf("expected storage to receive count 10, got %d", mock.lastDelCount)
		}
	})

	t.Run("200 OK with time_duration", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/del", strings.NewReader(`{"time_duration": 15}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if mock.lastDelCount != 15 {
			t.Errorf("expected storage to receive count 15, got %d", mock.lastDelCount)
		}
	})

	t.Run("500 on storage error", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timeListDelDBErr: errors.New("db error")}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/del", strings.NewReader(`{"count": 10}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", resp.StatusCode)
		}
	})
}

func TestLegacyTimerHandler_TimerGlobalSet(t *testing.T) {
	t.Run("200 OK", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/manage/timer/global", bytes.NewBufferString(`{"time_scheduler": 60}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if mock.lastGlobalSet != 60 {
			t.Errorf("expected storage to receive time_scheduler 60, got %d", mock.lastGlobalSet)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/manage/timer/global", bytes.NewBufferString(`{"time_scheduler": "invalid"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("500 on storage error", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timerGlobalSetErr: errors.New("db error")}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/manage/timer/global", bytes.NewBufferString(`{"time_scheduler": 60}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", resp.StatusCode)
		}
	})
}

func TestLegacyTimerHandler_TimerGlobalGet(t *testing.T) {
	t.Run("200 OK", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timerGlobalGetVal: 42}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/manage/timer/global", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}
		tg, ok := body["timer_global"].(float64)
		if !ok || int(tg) != 42 {
			t.Errorf("expected timer_global=42, got %v", body["timer_global"])
		}
	})

	t.Run("500 on storage error", func(t *testing.T) {
		mock := &mockLegacyTimerStorage{timerGlobalGetErr: errors.New("db error")}
		app := setupTimerApp(mock)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/manage/timer/global", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", resp.StatusCode)
		}
	})
}
