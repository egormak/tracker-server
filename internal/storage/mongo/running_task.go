package mongo

import (
	"context"
	"fmt"
	"time"

	"tracker-server/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const runningTaskCollection = "running_task"

func (s *Storage) GetRunningTask() (entity.RunningTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	var task entity.RunningTask
	// We assume only one running task at a time for single user app
	err := collection.FindOne(ctx, bson.M{}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.RunningTask{}, nil
		}
		return entity.RunningTask{}, fmt.Errorf("failed to get running task: %w", err)
	}

	return task, nil
}

func (s *Storage) UpsertRunningTask(task entity.RunningTask) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	// Keep only one document
	// We delete all others first to ensure singleton
	if err := s.DeleteRunningTask(); err != nil {
		return err
	}

	_, err := collection.InsertOne(ctx, task)
	return err
}

func (s *Storage) DeleteRunningTask() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	_, err := collection.DeleteMany(ctx, bson.M{})
	return err
}
