package main

import (
	"context"
	"fmt"
	"log"
	"tracker-server/config"
	"tracker-server/internal/storage/mongo"
)

func main() {
	cfg := config.LoadConfig()
	ctx := context.Background()

	// Init DataBase
	mongoUri := fmt.Sprintf("mongodb://%s:%s/", cfg.MongoDB.Host, cfg.MongoDB.Port)
	mongoconn, err := mongo.New(ctx, mongoUri)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(mongoconn.GetTasksbyPriority("plan"))
}
