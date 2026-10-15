package handler

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"tracker-server/internal/domain/entity"

	"github.com/gofiber/fiber/v2"
)

type mockDebtReader struct {
	summary entity.DebtSummary
	err     error
}

func (m *mockDebtReader) Summary() (entity.DebtSummary, error) { return m.summary, m.err }

func TestDebtHandler_GetDebt(t *testing.T) {
	reader := &mockDebtReader{summary: entity.DebtSummary{
		TotalMinutes: 36,
		Items:        []entity.DebtItem{{TaskName: "home_task", Day: "wednesday", Remaining: 36}},
		OldestDay:    "wednesday",
	}}
	app := fiber.New()
	app.Get("/api/v1/debt", NewDebtHandler(reader).GetDebt)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/debt", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Status string             `json:"status"`
		Data   entity.DebtSummary `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "success" || body.Data.TotalMinutes != 36 || body.Data.OldestDay != "wednesday" {
		t.Errorf("unexpected body: %+v", body)
	}

	reader.err = errors.New("no active schedule")
	resp, err = app.Test(httptest.NewRequest("GET", "/api/v1/debt", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected 500 on error, got %d", resp.StatusCode)
	}
}
