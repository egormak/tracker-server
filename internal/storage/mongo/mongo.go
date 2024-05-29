package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName        = "tasker"
	taskInfo      = "task_info"
	tasksList     = "tasks"
	taskNamesList = "task_list"
	roleInfo      = "role_info"

	restDocName    = "Rest Info"
	procentDocName = "Procent Info"
	restCount      = 30
)

var roleTypes = [3]string{"work", "learn", "rest"}
var PlanTypesWeekDays = []string{"plan", "work", "work", "work", "work", "learn", "learn", "learn", "rest"}
var PlanTypesWeekEndsDays = []string{"plan", "work", "learn", "rest"}
var PlanTypes = []string{"plan", "work", "learn", "rest"}

type Storage struct {
	Client  *mongo.Client
	Context context.Context
}

// New returns a new mongo.Storage with a connection to the given uri
func New(ctx context.Context, uri string) (*Storage, error) {
	// Check that the required arguments are not null
	if ctx == nil {
		return nil, fmt.Errorf("null pointer: context is required")
	}

	if uri == "" {
		return nil, fmt.Errorf("null pointer: uri is required")
	}

	// Connect to the mongo database specified by the uri
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("error connecting to mongo: %w", err)
	}

	// Return the new storage with the mongo client and context
	return &Storage{
		Client:  client,
		Context: ctx,
	}, nil
}
