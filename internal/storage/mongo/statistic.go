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

// GetRecordsForDatesOrSourceDays retrieves task records matching either an exact
// record date (with no source_day) or a rollover source_day, mirroring
// GetTaskDurationForDate's per-task fallback semantics: a source_day match ignores
// date, while a date match only applies to records that aren't already a rollover.
func (s *Storage) GetRecordsForDatesOrSourceDays(dates []string, sourceDays []string) ([]entity.TaskRecord, error) {
	coll := s.Client.Database(dbName).Collection(tasksList)

	filter := bson.M{
		"$or": []bson.M{
			{"date": bson.M{"$in": dates}, "source_day": bson.M{"$in": []interface{}{"", nil}}},
			{"source_day": bson.M{"$in": sourceDays}},
		},
	}

	cursor, err := coll.Find(s.Context, filter)
	if err != nil {
		return nil, fmt.Errorf("GetRecordsForDatesOrSourceDays: failed to find records: %w", err)
	}
	defer cursor.Close(s.Context)

	var result []entity.TaskRecord
	if err := cursor.All(s.Context, &result); err != nil {
		return nil, fmt.Errorf("GetRecordsForDatesOrSourceDays: failed to decode records: %w", err)
	}

	return result, nil
}
