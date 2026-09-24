package mongo

import (
	"context"
	"fmt"
	"time"
	"tracker-server/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetRampInfo retrieves the ramp configuration and state from MongoDB.
// If the document does not exist, it returns default RampInfo.
func (s *Storage) GetRampInfo(ctx context.Context) (entity.RampInfo, error) {
	if ctx == nil {
		ctx = s.Context
	}
	coll := s.Client.Database(dbName).Collection(taskInfo)

	var ramp entity.RampInfo
	err := coll.FindOne(ctx, bson.M{"title": rampDocName}).Decode(&ramp)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.DefaultRampInfo(), nil
		}
		return entity.RampInfo{}, fmt.Errorf("failed to get ramp info: %w", err)
	}

	if ramp.CapMinutes <= 0 {
		ramp.CapMinutes = 25
	}
	if ramp.CurrentStep <= 0 {
		ramp.CurrentStep = 1
	}
	if ramp.DefaultRestFallback <= 0 {
		ramp.DefaultRestFallback = 15
	}
	if ramp.EnabledRoles == nil {
		ramp.EnabledRoles = []string{"work", "learn"}
	}
	if ramp.EnabledTasks == nil {
		ramp.EnabledTasks = []string{"home_task"}
	}
	if ramp.ExcludedTasks == nil {
		ramp.ExcludedTasks = []string{"video", "movies", "games", "telegram"}
	}
	if ramp.Date == "" {
		ramp.Date = time.Now().Format("2 January 2006")
	}

	return ramp, nil
}

// SaveRampInfo persists the entire RampInfo document.
func (s *Storage) SaveRampInfo(ctx context.Context, ramp entity.RampInfo) error {
	if ctx == nil {
		ctx = s.Context
	}
	coll := s.Client.Database(dbName).Collection(taskInfo)

	ramp.Title = rampDocName
	ramp.UpdatedAt = time.Now().UTC()

	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(
		ctx,
		bson.M{"title": rampDocName},
		bson.M{"$set": ramp},
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to save ramp info: %w", err)
	}
	return nil
}

// ResetRampStep resets current_step to 1 and updates the date.
func (s *Storage) ResetRampStep(ctx context.Context, date string) error {
	if ctx == nil {
		ctx = s.Context
	}
	coll := s.Client.Database(dbName).Collection(taskInfo)

	now := time.Now().UTC()
	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(
		ctx,
		bson.M{"title": rampDocName},
		bson.M{
			"$set": bson.M{
				"current_step": 1,
				"date":         date,
				"updated_at":   now,
			},
			"$setOnInsert": bson.M{
				"title":                 rampDocName,
				"cap_minutes":          25,
				"enabled_roles":        []string{"work", "learn"},
				"enabled_tasks":        []string{"home_task"},
				"excluded_tasks":       []string{"video", "movies", "games", "telegram"},
				"default_rest_fallback": 15,
			},
		},
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to reset ramp step: %w", err)
	}
	return nil
}

// UpdateRampStep updates the current step and date.
func (s *Storage) UpdateRampStep(ctx context.Context, step int, date string) error {
	if ctx == nil {
		ctx = s.Context
	}
	coll := s.Client.Database(dbName).Collection(taskInfo)

	now := time.Now().UTC()
	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(
		ctx,
		bson.M{"title": rampDocName},
		bson.M{
			"$set": bson.M{
				"current_step": step,
				"date":         date,
				"updated_at":   now,
			},
			"$setOnInsert": bson.M{
				"title":                 rampDocName,
				"cap_minutes":          25,
				"enabled_roles":        []string{"work", "learn"},
				"enabled_tasks":        []string{"home_task"},
				"excluded_tasks":       []string{"video", "movies", "games", "telegram"},
				"default_rest_fallback": 15,
			},
		},
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to update ramp step: %w", err)
	}
	return nil
}
