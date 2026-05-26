package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"tracker-server/config"

	"github.com/gofiber/fiber/v2"
	"log/slog"
)

type tgUser struct {
	ID int64 `json:"id"`
}

// TelegramAuth validates incoming Telegram Mini App initialization data
func TelegramAuth(cfg config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If webapp auth is disabled in configuration, bypass validation
		if !cfg.Telegram.EnableWebappAuth {
			return c.Next()
		}

		// Allow bypassing if the request contains the bot's API Key (e.g. from the bot container)
		botToken := c.Get("X-Bot-Token")
		if botToken != "" && botToken == cfg.Telegram.APIKey {
			return c.Next()
		}

		initData := c.Get("X-Telegram-Init-Data")
		if initData == "" {
			slog.Warn("TelegramAuth: missing auth credentials")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Missing Telegram WebApp initialization data",
			})
		}

		// 1. Verify cryptographic signature of initData
		if !verifyTelegramSignature(initData, cfg.Telegram.APIKey) {
			slog.Warn("TelegramAuth: signature verification failed")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Forbidden: Invalid Telegram WebApp signature",
			})
		}

		// 2. Parse values to check user ID
		params, err := url.ParseQuery(initData)
		if err != nil {
			slog.Error("TelegramAuth: failed to parse query parameters", "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Bad Request: Malformed initialization data",
			})
		}

		userJson := params.Get("user")
		if userJson == "" {
			slog.Warn("TelegramAuth: user parameter not found in initData")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Forbidden: User data not found in Telegram WebApp data",
			})
		}

		var user tgUser
		if err := json.Unmarshal([]byte(userJson), &user); err != nil {
			slog.Error("TelegramAuth: failed to unmarshal user JSON", "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Bad Request: Invalid user JSON data",
			})
		}

		// 3. Verify user ID matches room_id (admin ID)
		if user.ID != cfg.Telegram.RoomID {
			slog.Warn("TelegramAuth: access denied for user", "userID", user.ID, "expectedID", cfg.Telegram.RoomID)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Forbidden: Access denied",
			})
		}

		return c.Next()
	}
}

func verifyTelegramSignature(initData string, botToken string) bool {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	hash := params.Get("hash")
	if hash == "" {
		return false
	}

	// Remove hash from validation data
	params.Del("hash")

	// Sort parameters alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Construct data-check-string
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params.Get(k))
	}
	dataCheckString := strings.Join(parts, "\n")

	// Secret key is HMAC-SHA256 of botToken with salt "WebApps"
	mac := hmac.New(sha256.New, []byte("WebApps"))
	mac.Write([]byte(botToken))
	secretKey := mac.Sum(nil)

	// Calculate check hash
	mac2 := hmac.New(sha256.New, secretKey)
	mac2.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac2.Sum(nil))

	return hmac.Equal([]byte(hash), []byte(expectedHash))
}
