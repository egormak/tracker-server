package mongodb

import (
	"fmt"
	"time"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
)

type StatisticRepository struct {
	Storage
}

// GetTaskRecordToday retrieves task records for the current day from MongoDB
func (s *Storage) GetTaskRecordToday(opts ...storage.TaskRecordOption) ([]storage.TaskRecord, error) {
	// Initialize default options
	options := &storage.TaskRecordOptions{
		CheckBusinessDay: false,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get the database and collection
	coll := s.Client.Database(dbName).Collection(tasksList)

	// Create the filter based on the provided options
	filter := createFilter(options)

	// Find all records matching the filter
	cursor, err := coll.Find(s.Context, filter)
	if err != nil {
		return nil, fmt.Errorf("GetTaskRecordToday: failed to find records: %w", err)
	}
	defer cursor.Close(s.Context)

	// Decode the results into a slice of TaskRecord
	var result []storage.TaskRecord
	if err := cursor.All(s.Context, &result); err != nil {
		return nil, fmt.Errorf("GetTaskRecordToday: failed to decode records: %w", err)
	}

	return result, nil
}

// createFilter generates the appropriate BSON filter based on the provided options
func createFilter(opts *storage.TaskRecordOptions) bson.M {
	today := time.Now().Format("2 January 2006")
	filter := bson.M{"date": today}

	if opts.CheckBusinessDay {
		filter["role"] = bson.M{"$ne": "rest"}
	}

	return filter
}
