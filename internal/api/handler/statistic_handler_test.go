package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type mockStatService struct {
	todayTasks []entity.TaskResult
	todayErr   error
	weeklyResp entity.WeeklyStatsResponse
	weeklyErr  error
}

func (m *mockStatService) GetTaskRecordToday() ([]entity.TaskResult, error) {
	return m.todayTasks, m.todayErr
}

func (m *mockStatService) GetWeeklyStats() (entity.WeeklyStatsResponse, error) {
	return m.weeklyResp, m.weeklyErr
}

type mockRunningTaskProvider struct {
	task entity.RunningTask
	err  error
}

func (m *mockRunningTaskProvider) GetStatus(taskName string) (entity.RunningTask, error) {
	return m.task, m.err
}

type mockRestProvider struct {
	units int
	err   error
}

func (m *mockRestProvider) GetRest() (int, error) {
	return m.units, m.err
}

type mockEveningProvider struct {
	focus entity.EveningFocusResponse
	err   error
}

func (m *mockEveningProvider) GetEveningFocus(category string, timeOverride int) (entity.EveningFocusResponse, error) {
	return m.focus, m.err
}

func TestStatisticHandler_GetDashboardState(t *testing.T) {
	now := time.Now()

	t.Run("success with running task and evening focus", func(t *testing.T) {
		statSvc := &mockStatService{
			todayTasks: []entity.TaskResult{
				{Name: "work", Role: "work", TimeDuration: 120, TimeDone: 60, Priority: 1},
				{Name: "english", Role: "learn", TimeDuration: 60, TimeDone: 60, Priority: 2},
			},
		}
		runningSvc := &mockRunningTaskProvider{
			task: entity.RunningTask{
				TaskName:  "work",
				Role:      "work",
				IsRunning: true,
				StartTime: now,
			},
		}
		restSvc := &mockRestProvider{
			units: 3000, // 30 minutes in raw units (minutes * 100)
		}
		eveningSvc := &mockEveningProvider{
			focus: entity.EveningFocusResponse{
				CurrentTask: entity.EveningFocusCandidate{TaskName: "reading", Role: "learn", WeeklyGap: 45},
				Candidates: []entity.EveningFocusCandidate{
					{TaskName: "reading", Role: "learn", WeeklyGap: 45},
					{TaskName: "guitar", Role: "rest", WeeklyGap: 30},
					{TaskName: "workout", Role: "sport", WeeklyGap: 20},
					{TaskName: "extra", Role: "other", WeeklyGap: 10},
				},
				SprintTime: 20,
				RestPool:   30,
			},
		}

		h := NewStatisticHandler(statSvc, runningSvc, restSvc, eveningSvc)
		app := fiber.New()
		app.Get("/api/v1/dashboard/state", h.GetDashboardState)

		req := httptest.NewRequest("GET", "/api/v1/dashboard/state", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		var dashResp entity.DashboardStateResponse
		if err := json.Unmarshal(body, &dashResp); err != nil {
			t.Fatalf("failed to unmarshal dashboard response: %v", err)
		}

		// Verify RunningTask
		if dashResp.RunningTask == nil || dashResp.RunningTask.TaskName != "work" {
			t.Errorf("expected running task 'work', got %+v", dashResp.RunningTask)
		}

		// Verify TodayTasks
		if len(dashResp.TodayTasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(dashResp.TodayTasks))
		}

		// Verify Totals and Percentage: totalPlanned = 180, totalDone = 120 -> 66%
		if dashResp.TotalPlanned != 180 {
			t.Errorf("expected TotalPlanned 180, got %d", dashResp.TotalPlanned)
		}
		if dashResp.TotalDone != 120 {
			t.Errorf("expected TotalDone 120, got %d", dashResp.TotalDone)
		}
		if dashResp.CompletionPct != 66 {
			t.Errorf("expected CompletionPct 66, got %d", dashResp.CompletionPct)
		}

		// Verify RestPool (units = minutes * 100)
		if dashResp.RestPool != 3000 {
			t.Errorf("expected RestPool 3000, got %d", dashResp.RestPool)
		}

		// Verify EveningFocus candidates capped at 3
		if dashResp.EveningFocus == nil {
			t.Fatalf("expected EveningFocus not nil")
		}
		if len(dashResp.EveningFocus.Candidates) != 3 {
			t.Errorf("expected 3 evening candidates (capped), got %d", len(dashResp.EveningFocus.Candidates))
		}
	})

	t.Run("error getting today tasks", func(t *testing.T) {
		statSvc := &mockStatService{
			todayErr: errors.New("db error"),
		}
		h := NewStatisticHandler(statSvc, nil, nil, nil)
		app := fiber.New()
		app.Get("/api/v1/dashboard/state", h.GetDashboardState)

		req := httptest.NewRequest("GET", "/api/v1/dashboard/state", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != fiber.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.StatusCode)
		}
	})
}
