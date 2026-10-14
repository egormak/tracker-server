package middleware

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ClientHeader names the client a request comes from.
const ClientHeader = "X-Tracker-Client"

const clientLocalsKey = "tracker_client"

var knownClients = map[string]bool{
	"cli":     true,
	"web":     true,
	"miniapp": true,
	"bot":     true,
	"app":     true,
	"mcp":     true,
}

// ClientID stores the calling client's name for the handlers. Anything outside the
// known set is treated as absent, so the header cannot grow the set of stored values.
func ClientID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := strings.ToLower(strings.TrimSpace(c.Get(ClientHeader)))
		if knownClients[name] {
			c.Locals(clientLocalsKey, name)
		}
		return c.Next()
	}
}

// Client returns the calling client's name, or "" when it did not identify itself.
func Client(c *fiber.Ctx) string {
	name, _ := c.Locals(clientLocalsKey).(string)
	return name
}

// UsageTracker counts a user choice.
type UsageTracker interface {
	Track(event, client string)
}

// TrackUsage counts event after the wrapped handler answered 200. With cond, the
// request is counted only when cond returns true.
func TrackUsage(tracker UsageTracker, event string, cond ...func(c *fiber.Ctx) bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}
		if c.Response().StatusCode() != fiber.StatusOK {
			return nil
		}
		for _, ok := range cond {
			if !ok(c) {
				return nil
			}
		}
		tracker.Track(event, Client(c))
		return nil
	}
}

// PositiveDelta reports a timer adjustment that adds time (delta_minutes > 0).
func PositiveDelta(c *fiber.Ctx) bool {
	var body struct {
		DeltaMinutes int `json:"delta_minutes"`
	}
	return json.Unmarshal(c.Body(), &body) == nil && body.DeltaMinutes > 0
}
