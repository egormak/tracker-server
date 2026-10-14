package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type trackedEvent struct{ event, client string }

type fakeTracker struct{ events []trackedEvent }

func (f *fakeTracker) Track(event, client string) {
	f.events = append(f.events, trackedEvent{event, client})
}

func TestClientID_KeepsOnlyKnownClients(t *testing.T) {
	app := fiber.New()
	app.Get("/", ClientID(), func(c *fiber.Ctx) error { return c.SendString(Client(c)) })

	cases := map[string]string{
		"cli":       "cli",
		" MiniApp ": "miniapp",
		"curl":      "",
		"":          "",
	}
	for header, want := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			req.Header.Set(ClientHeader, header)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("header %q: %v", header, err)
		}
		buf := make([]byte, 32)
		n, _ := resp.Body.Read(buf)
		if got := string(buf[:n]); got != want {
			t.Errorf("header %q: client = %q, want %q", header, got, want)
		}
	}
}

func TestTrackUsage_CountsOnlySuccessfulMatchingRequests(t *testing.T) {
	tracker := &fakeTracker{}
	app := fiber.New()
	app.Use(ClientID())
	app.Post("/ok", TrackUsage(tracker, "ok_event"), func(c *fiber.Ctx) error { return c.Status(200).JSON(fiber.Map{}) })
	app.Post("/fail", TrackUsage(tracker, "fail_event"), func(c *fiber.Ctx) error { return c.Status(500).JSON(fiber.Map{}) })
	app.Post("/adjust", TrackUsage(tracker, "extend", PositiveDelta), func(c *fiber.Ctx) error { return c.Status(200).JSON(fiber.Map{}) })

	post := func(path, body string) {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(ClientHeader, "web")
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	post("/ok", "")
	post("/fail", "")
	post("/adjust", `{"delta_minutes": 10}`)
	post("/adjust", `{"delta_minutes": -5}`)
	post("/adjust", `not json`)

	want := []trackedEvent{{"ok_event", "web"}, {"extend", "web"}}
	if len(tracker.events) != len(want) {
		t.Fatalf("events = %v, want %v", tracker.events, want)
	}
	for i := range want {
		if tracker.events[i] != want[i] {
			t.Errorf("events[%d] = %v, want %v", i, tracker.events[i], want[i])
		}
	}
}
