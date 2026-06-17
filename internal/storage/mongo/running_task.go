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

const runningTaskCollection = "running_task"

func (s *Storage) GetRunningTask(taskName string) (entity.RunningTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	var task entity.RunningTask
	err := collection.FindOne(ctx, bson.M{"task_name": taskName}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.RunningTask{}, nil
		}
		return entity.RunningTask{}, fmt.Errorf("failed to get running task: %w", err)
	}

	return task, nil
}

func (s *Storage) GetActiveRunningTask() (entity.RunningTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	var task entity.RunningTask
	err := collection.FindOne(ctx, bson.M{"is_running": true}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.RunningTask{}, nil
		}
		return entity.RunningTask{}, fmt.Errorf("failed to get active running task: %w", err)
	}

	return task, nil
}

func (s *Storage) GetAllRunningTasks() ([]entity.RunningTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find running tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []entity.RunningTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("failed to decode running tasks: %w", err)
	}

	return tasks, nil
}

func (s *Storage) UpsertRunningTask(task entity.RunningTask) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	filter := bson.M{"task_name": task.TaskName}
	update := bson.M{"$set": task}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *Storage) DeleteRunningTask(taskName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.Client.Database(dbName).Collection(runningTaskCollection)

	_, err := collection.DeleteOne(ctx, bson.M{"task_name": taskName})
	return err
}
