package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
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
	rampDocName    = "Ramp Info"
	restCount      = 30
)

var roleTypes = [3]string{"work", "learn", "rest"}
var PlanTypesWeekDays = []string{"plan", "work", "learn", "rest"}
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

// EnsureIndexes creates required secondary and unique indexes on MongoDB collections.
func (s *Storage) EnsureIndexes(ctx context.Context) error {
	db := s.Client.Database(dbName)

	// tasks collection: date: 1, compound name: 1, date: 1, source_day: 1, created_at: -1
	tasksColl := db.Collection(tasksList)
	_, err := tasksColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "date", Value: 1}}},
		{Keys: bson.D{{Key: "name", Value: 1}, {Key: "date", Value: 1}}},
		{Keys: bson.D{{Key: "source_day", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes on %s: %w", tasksList, err)
	}

	// running_task collection: unique index on task_name: 1, index on is_running: 1
	runningColl := db.Collection(runningTaskCollection)
	_, err = runningColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "task_name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "is_running", Value: 1}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes on %s: %w", runningTaskCollection, err)
	}

	// weekly_schedules collection: index on is_active: 1
	schedColl := db.Collection(weeklySchedulesCollection)
	_, err = schedColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "is_active", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes on %s: %w", weeklySchedulesCollection, err)
	}

	// task_list collection: index on date: 1
	taskListColl := db.Collection(taskNamesList)
	_, err = taskListColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "date", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes on %s: %w", taskNamesList, err)
	}

	return nil
}
