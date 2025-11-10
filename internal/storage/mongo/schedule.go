package mongo

import (
	"fmt"
	"time"
	"tracker-server/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	weeklySchedulesCollection = "weekly_schedules"
)

// CreateSchedule creates a new weekly schedule and optionally sets it as active
func (s *Storage) CreateSchedule(schedule entity.WeeklySchedule) (string, error) {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	// Set timestamps
	now := time.Now().Format("2 January 2006")
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	// If this is set as active, deactivate all others first
	if schedule.IsActive {
		if err := s.deactivateAllSchedules(); err != nil {
			return "", fmt.Errorf("failed to deactivate existing schedules: %w", err)
		}
	}

	// Insert the schedule
	result, err := coll.InsertOne(s.Context, schedule)
	if err != nil {
		return "", fmt.Errorf("failed to create schedule: %w", err)
	}

	scheduleID := result.InsertedID.(primitive.ObjectID).Hex()
	return scheduleID, nil
}

// GetSchedule retrieves a schedule by ID
func (s *Storage) GetSchedule(id string) (entity.WeeklySchedule, error) {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return entity.WeeklySchedule{}, fmt.Errorf("invalid schedule ID: %w", err)
	}

	var schedule entity.WeeklySchedule
	err = coll.FindOne(s.Context, bson.M{"_id": objectID}).Decode(&schedule)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.WeeklySchedule{}, fmt.Errorf("schedule not found")
		}
		return entity.WeeklySchedule{}, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Set ID field from MongoDB _id
	schedule.ID = objectID.Hex()
	return schedule, nil
}

// GetActiveSchedule retrieves the currently active schedule
func (s *Storage) GetActiveSchedule() (entity.WeeklySchedule, error) {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	result := coll.FindOne(s.Context, bson.M{"is_active": true})
	if err := result.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.WeeklySchedule{}, fmt.Errorf("no active schedule found")
		}
		return entity.WeeklySchedule{}, fmt.Errorf("failed to get active schedule: %w", err)
	}

	var doc struct {
		entity.WeeklySchedule `bson:",inline"`
		ObjectID              primitive.ObjectID `bson:"_id"`
	}

	if err := result.Decode(&doc); err != nil {
		return entity.WeeklySchedule{}, fmt.Errorf("failed to decode active schedule: %w", err)
	}

	schedule := doc.WeeklySchedule
	schedule.ID = doc.ObjectID.Hex()
	return schedule, nil
}

// GetAllSchedules retrieves all schedules
func (s *Storage) GetAllSchedules() ([]entity.WeeklySchedule, error) {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	cursor, err := coll.Find(s.Context, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	defer cursor.Close(s.Context)

	var schedules []entity.WeeklySchedule
	for cursor.Next(s.Context) {
		var schedule entity.WeeklySchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, fmt.Errorf("failed to decode schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// UpdateSchedule updates an existing schedule
func (s *Storage) UpdateSchedule(id string, schedule entity.WeeklySchedule) error {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid schedule ID: %w", err)
	}

	// Update timestamp
	schedule.UpdatedAt = time.Now().Format("2 January 2006")

	// If setting this as active, deactivate others first
	if schedule.IsActive {
		if err := s.deactivateAllSchedules(); err != nil {
			return fmt.Errorf("failed to deactivate existing schedules: %w", err)
		}
	}

	filter := bson.M{"_id": objectID}
	// Build update document without _id field
	update := bson.M{
		"$set": bson.M{
			"title":      schedule.Title,
			"is_active":  schedule.IsActive,
			"monday":     schedule.Monday,
			"tuesday":    schedule.Tuesday,
			"wednesday":  schedule.Wednesday,
			"thursday":   schedule.Thursday,
			"friday":     schedule.Friday,
			"saturday":   schedule.Saturday,
			"sunday":     schedule.Sunday,
			"updated_at": schedule.UpdatedAt,
		},
	}

	result, err := coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// DeleteSchedule deletes a schedule by ID
func (s *Storage) DeleteSchedule(id string) error {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid schedule ID: %w", err)
	}

	result, err := coll.DeleteOne(s.Context, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("schedule not found")
	}

	return nil
}

// SetActiveSchedule sets a specific schedule as active and deactivates all others
func (s *Storage) SetActiveSchedule(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid schedule ID: %w", err)
	}

	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	// Check if schedule exists
	count, err := coll.CountDocuments(s.Context, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to check schedule existence: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("schedule not found")
	}

	// Deactivate all schedules
	if err := s.deactivateAllSchedules(); err != nil {
		return fmt.Errorf("failed to deactivate existing schedules: %w", err)
	}

	// Activate the specified schedule
	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"is_active":  true,
			"updated_at": time.Now().Format("2 January 2006"),
		},
	}

	_, err = coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return fmt.Errorf("failed to activate schedule: %w", err)
	}

	return nil
}

// deactivateAllSchedules sets is_active=false for all schedules
func (s *Storage) deactivateAllSchedules() error {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	filter := bson.M{"is_active": true}
	update := bson.M{"$set": bson.M{"is_active": false}}

	_, err := coll.UpdateMany(s.Context, filter, update)
	if err != nil {
		return fmt.Errorf("failed to deactivate schedules: %w", err)
	}

	return nil
}

// GetDaySchedule retrieves the schedule for a specific day from the active schedule
func (s *Storage) GetDaySchedule(day string) (entity.DaySchedule, error) {
	schedule, err := s.GetActiveSchedule()
	if err != nil {
		return entity.DaySchedule{}, err
	}

	switch day {
	case "monday":
		return schedule.Monday, nil
	case "tuesday":
		return schedule.Tuesday, nil
	case "wednesday":
		return schedule.Wednesday, nil
	case "thursday":
		return schedule.Thursday, nil
	case "friday":
		return schedule.Friday, nil
	case "saturday":
		return schedule.Saturday, nil
	case "sunday":
		return schedule.Sunday, nil
	default:
		return entity.DaySchedule{}, fmt.Errorf("invalid day: %s", day)
	}
}

// GetSchedulesWithFilter retrieves schedules based on filter criteria
func (s *Storage) GetSchedulesWithFilter(filter bson.M) ([]entity.WeeklySchedule, error) {
	coll := s.Client.Database(dbName).Collection(weeklySchedulesCollection)

	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := coll.Find(s.Context, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	defer cursor.Close(s.Context)

	var schedules []entity.WeeklySchedule
	for cursor.Next(s.Context) {
		var schedule entity.WeeklySchedule
		if err := cursor.Decode(&schedule); err != nil {
			return nil, fmt.Errorf("failed to decode schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}
