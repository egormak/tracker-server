package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"tracker-server/internal/api/routes"
	"tracker-server/internal/notify/telegram"
	"tracker-server/internal/storage/mongo"

	"tracker-server/config"

	"github.com/lmittmann/tint"
	log "github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// main is the entry point of the application
func main() {

	logger_slog := slog.New(tint.NewHandler(os.Stderr, nil))
	slog.SetDefault(logger_slog)
	slog.Info("Start Application")

	cfg := config.LoadConfig()
	ctx := context.Background()

	// Init DataBase
	mongoUri := fmt.Sprintf("mongodb://%s:%s/", cfg.MongoDB.Host, cfg.MongoDB.Port)

	mongoconn, err := mongo.New(ctx, mongoUri)
	if err != nil {
		slog.Error("Can't connect to mongo", "error", err)
		os.Exit(1)

	}
	notify := telegram.TelegramNew(cfg.Telegram.APIKey, cfg.Telegram.RoomID)

	app := fiber.New()
	// Use the logger middleware
	app.Use(logger.New(logger.Config{}))

	routes.RegisterRoutes(app, mongoconn, notify)

	// Start the server
	log.Fatal(app.Listen(":3000"))

}
