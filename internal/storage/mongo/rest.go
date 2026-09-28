package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	maxDailyRestUnits = 6000
	MaxDailyRestUnits = maxDailyRestUnits
)

type RestData struct {
	Title     string `bson:"title,omitempty"`
	RestCount int    `bson:"restcount"`
	Date      string `bson:"date"`
}

// AddRestUnits adds the specified raw units directly to today's rest count using atomic $inc, clamped at maxDailyRestUnits.
func (s *Storage) AddRestUnits(units int) error {
	coll := s.Client.Database(dbName).Collection(taskInfo)
	today := time.Now().Format("2 January 2006")

	// Reset restcount to 0 if the existing document is from a previous date
	_, err := coll.UpdateOne(
		s.Context,
		bson.M{"title": restDocName, "date": bson.M{"$ne": today}},
		bson.M{"$set": bson.M{"restcount": 0, "date": today}},
	)
	if err != nil {
		return fmt.Errorf("error in AddRestUnits resetting previous date: %w", err)
	}

	// Atomically increment rest units and update date
	var updated RestData
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	err = coll.FindOneAndUpdate(
		s.Context,
		bson.M{"title": restDocName},
		bson.M{
			"$inc": bson.M{"restcount": units},
			"$set": bson.M{"date": today},
		},
		opts,
	).Decode(&updated)
	if err != nil {
		return fmt.Errorf("error in AddRestUnits: %w", err)
	}

	// Clamp between 0 and maxDailyRestUnits
	if updated.RestCount > maxDailyRestUnits {
		_, err = coll.UpdateOne(s.Context, bson.M{"title": restDocName}, bson.M{"$set": bson.M{"restcount": maxDailyRestUnits}})
		if err != nil {
			return fmt.Errorf("error in AddRestUnits clamping max: %w", err)
		}
	} else if updated.RestCount < 0 {
		_, err = coll.UpdateOne(s.Context, bson.M{"title": restDocName}, bson.M{"$set": bson.M{"restcount": 0}})
		if err != nil {
			return fmt.Errorf("error in AddRestUnits clamping min: %w", err)
		}
	}

	return nil
}

// AddRestMinutes adds minutes * 100 raw units (for direct rest time additions, e.g. API).
func (s *Storage) AddRestMinutes(minutes int) error {
	return s.AddRestUnits(minutes * 100)
}

// AddTaskEarnedRest adds earned rest for completed work tasks (30 units per work minute).
func (s *Storage) AddTaskEarnedRest(workMinutes int) error {
	return s.AddRestUnits(workMinutes * restCount)
}

// AddRest adds earned rest time for completed tasks (delegates to AddTaskEarnedRest).
func (s *Storage) AddRest(restTime int) error {
	return s.AddTaskEarnedRest(restTime)
}

// RestSpendUnits deducts raw rest units atomically from today's rest pool.
// Uses atomic $inc with filter requiring today's date and sufficient balance.
func (s *Storage) RestSpendUnits(units int) error {
	if units <= 0 {
		return fmt.Errorf("units to spend must be positive, got %d", units)
	}

	coll := s.Client.Database(dbName).Collection(taskInfo)
	today := time.Now().Format("2 January 2006")

	filter := bson.M{
		"title":     restDocName,
		"date":      today,
		"restcount": bson.M{"$gte": units},
	}
	update := bson.M{
		"$inc": bson.M{"restcount": -units},
	}

	result, err := coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return fmt.Errorf("error in RestSpendUnits: %w", err)
	}

	if result.MatchedCount == 0 {
		var doc RestData
		findErr := coll.FindOne(s.Context, bson.M{"title": restDocName}).Decode(&doc)
		if findErr != nil {
			if findErr == mongo.ErrNoDocuments {
				return fmt.Errorf("insufficient rest balance: no rest record found (have 0, need %d)", units)
			}
			return fmt.Errorf("error checking rest balance: %w", findErr)
		}
		if doc.Date != today {
			return fmt.Errorf("insufficient rest balance: record is from %s (have 0 for today, need %d)", doc.Date, units)
		}
		return fmt.Errorf("insufficient rest balance: have %d units, need %d units", doc.RestCount, units)
	}

	return nil
}

// RestSpend deducts minutes (scaled to raw units = minutes * 100) atomically from today's rest pool.
func (s *Storage) RestSpend(restMinutes int) error {
	return s.RestSpendUnits(restMinutes * 100)
}

func (s *Storage) ResetRest() error {
	coll := s.Client.Database(dbName).Collection(taskInfo)
	filter := bson.D{{Key: "title", Value: restDocName}}
	var result RestData
	err := coll.FindOne(s.Context, filter).Decode(&result)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return fmt.Errorf("error in ResetRest: %w", err)
		}
	}

	result.RestCount = 0
	result.Date = time.Now().Format("2 January 2006")

	update := bson.D{{Key: "$set", Value: result}}
	options := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(s.Context, filter, update, options)
	if err != nil {
		return fmt.Errorf("error in ResetRest: %w", err)
	}

	return nil
}

func (s *Storage) GetRest() (int, error) {

	var result RestData

	// Connect to Collection
	coll := s.Client.Database(dbName).Collection(taskInfo)

	// Find Collection
	filter := bson.D{{Key: "title", Value: restDocName}}
	err := coll.FindOne(s.Context, filter).Decode(&result)
	if err != nil {
		// If no matching document is found, set restTime to 0
		if err == mongo.ErrNoDocuments {
			return 0, nil
		} else {
			return 0, fmt.Errorf("error occurred in GetRest: %w", err)
		}
	}

	// If the date in the result is not today, reset the rest count to 0
	if result.Date != time.Now().Format("2 January 2006") {
		result.RestCount = 0
	}

	return result.RestCount, nil

}
