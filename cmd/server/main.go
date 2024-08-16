package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"tracker-server/internal/api/routes"
	"tracker-server/internal/notify/telegram"
	"tracker-server/internal/repository/mongodb"
	"tracker-server/internal/services"
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
	// Old
	mongoconn, err := mongo.New(ctx, mongoUri)
	if err != nil {
		slog.Error("main error", "error", err)
		os.Exit(1)
	}
	// New
	mongodbconn, err := mongodb.New(ctx, mongoUri)
	notify := telegram.TelegramNew(cfg.Telegram.APIKey, cfg.Telegram.RoomID)
	service := services.New(mongodbconn, notify)

	app := fiber.New()
	// Use the logger middleware
	app.Use(logger.New(logger.Config{}))

	routes.RegisterRoutes(app, mongoconn, notify, service)

	// Start the server
	log.Fatal(app.Listen(":3000"))

}
