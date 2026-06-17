package mongo

import (
	"fmt"
	"time"
	"tracker-server/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
)

// GetTaskRecordToday retrieves task records for the current day from MongoDB
func (s *Storage) GetTaskRecordToday() ([]entity.TaskRecord, error) {

	today := time.Now().Format("2 January 2006")

	// Get the database and collection
	coll := s.Client.Database(dbName).Collection(tasksList)

	// Create the filter based on the provided options
	filter := bson.M{"date": today}

	// Find all records matching the filter
	cursor, err := coll.Find(s.Context, filter)
	if err != nil {
		return nil, fmt.Errorf("GetTaskRecordToday: failed to find records: %w", err)
	}
	defer cursor.Close(s.Context)

	// Decode the results into a slice of TaskRecord
	var result []entity.TaskRecord
	if err := cursor.All(s.Context, &result); err != nil {
		return nil, fmt.Errorf("GetTaskRecordToday: failed to decode records: %w", err)
	}

	return result, nil
}

// GetRecordsForDates retrieves task records for a slice of dates from MongoDB
func (s *Storage) GetRecordsForDates(dates []string) ([]entity.TaskRecord, error) {
	// Get the database and collection
	coll := s.Client.Database(dbName).Collection(tasksList)

	// Create the filter using $in to match any of the provided dates
	filter := bson.M{"date": bson.M{"$in": dates}}

	// Find all records matching the filter
	cursor, err := coll.Find(s.Context, filter)
	if err != nil {
		return nil, fmt.Errorf("GetRecordsForDates: failed to find records: %w", err)
	}
	defer cursor.Close(s.Context)

	// Decode the results into a slice of TaskRecord
	var result []entity.TaskRecord
	if err := cursor.All(s.Context, &result); err != nil {
		return nil, fmt.Errorf("GetRecordsForDates: failed to decode records: %w", err)
	}

	return result, nil
}
