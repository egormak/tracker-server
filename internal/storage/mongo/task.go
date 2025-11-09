package mongo

import (
	"fmt"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Storage) GetRecords() ([]storage.TaskRecord, error) {

	var taskRecords []storage.TaskRecord

	coll := s.Client.Database(dbName).Collection(tasksList)
	cursor, err := coll.Find(s.Context, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		// Declare a result BSON object
		var taskRecord storage.TaskRecord
		err := cursor.Decode(&taskRecord)
		if err != nil {
			return nil, err
		}
		taskRecords = append(taskRecords, taskRecord)
	}

	return taskRecords, nil
}

func (s *Storage) CleanRecords() {

	// Set Value for DB
	database := s.Client.Database(dbName)

	collRoleInfo := database.Collection(roleInfo)
	collTaskInfo := database.Collection(taskInfo)
	collTasks := database.Collection(tasksList)

	collRoleInfo.Drop(s.Context)
	collTaskInfo.Drop(s.Context)
	collTasks.Drop(s.Context)

}

func (s *Storage) ShowTaskList() ([]storage.TaskResult, error) {

	var taskDefinitions []entity.TaskDefinition
	var taskResults []storage.TaskResult
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	cursor, err := coll.Find(s.Context, bson.M{"date": time.Now().Format("2 January 2006")})
	if err != nil {
		return nil, fmt.Errorf("show-task-list: %w", err)
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		var taskDef entity.TaskDefinition
		err := cursor.Decode(&taskDef)
		if err != nil {
			return nil, fmt.Errorf("show-task-list: %w", err)
		}
		taskDefinitions = append(taskDefinitions, taskDef)
	}

	for _, taskData := range taskDefinitions {
		timeDuration, err := s.GetTodayTaskDuration(taskData.Name)
		if err != nil {
			return nil, fmt.Errorf("show-task-list: %w", err)
		}
		taskResults = append(taskResults, storage.TaskResult{
			Name:         taskData.Name,
			Role:         taskData.Role,
			TimeDuration: taskData.TimeSchedule,
			TimeDone:     timeDuration,
			Priority:     taskData.Priority,
		})
	}

	return taskResults, nil

}

func (s *Storage) GetTodayTaskDuration(taskName string) (int, error) {

	// Select Value
	var timeDuretion int

	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)

	// Get Information about tasks
	cursor_task, err := coll.Find(s.Context, bson.M{"name": taskName, "date": time.Now().Format("2 January 2006")})
	if err != nil {
		return 0, fmt.Errorf("get-today-task-duration: %w", err)
	}
	defer cursor_task.Close(s.Context)

	for cursor_task.Next(s.Context) {
		// Declare a result BSON object
		var result storage.TaskRecord
		err := cursor_task.Decode(&result)
		if err != nil {
			return 0, fmt.Errorf("get-today-task-duration: %w", err)
		}
		timeDuretion += result.TimeDuration
	}
	return timeDuretion, nil

}

// GetTaskDurationForDate gets the total duration for a task on a specific date
func (s *Storage) GetTaskDurationForDate(taskName string, date string) (int, error) {
	var timeDuration int

	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)

	// Get Information about tasks for the specific date
	cursor_task, err := coll.Find(s.Context, bson.M{"name": taskName, "date": date})
	if err != nil {
		return 0, fmt.Errorf("get-task-duration-for-date: %w", err)
	}
	defer cursor_task.Close(s.Context)

	for cursor_task.Next(s.Context) {
		var result storage.TaskRecord
		err := cursor_task.Decode(&result)
		if err != nil {
			return 0, fmt.Errorf("get-task-duration-for-date: %w", err)
		}
		timeDuration += result.TimeDuration
	}
	return timeDuration, nil
}

func (s *Storage) SetTaskParams(params entity.TaskParams) error {

	var result entity.TaskDefinition

	// Set Value for DB
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	// Find Collection
	err := coll.FindOne(s.Context, bson.D{{"name", params.Name}}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return fmt.Errorf("set-task-params: %w", mongo.ErrNoDocuments)
	}

	result.TimeSchedule = params.Time
	result.Priority = params.Priority
	result.Date = time.Now().Format("2 January 2006")

	filter := bson.D{{"name", params.Name}}
	update := bson.D{{"$set", result}}
	_, err = coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetTaskParams(taskName string) (entity.TaskParams, error) {

	// Set Value for DB
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	// Find Collection
	var result entity.TaskDefinition
	err := coll.FindOne(s.Context, bson.D{{"name", taskName}}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return entity.TaskParams{}, fmt.Errorf("get-task-params: %w", mongo.ErrNoDocuments)
	}

	if result.Date != time.Now().Format("2 January 2006") {
		return entity.TaskParams{}, storage.ErrParamsOld
	}

	return entity.TaskParams{
		Name:     result.Name,
		Time:     result.TimeSchedule,
		Priority: result.Priority,
	}, nil

}

func (s *Storage) GetDayTaskRecord(taskName string) (int, error) {

	// Set Value for DB
	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)
	var taskResult int

	cursor, err := coll.Find(s.Context, bson.M{"name": taskName, "date": time.Now().Format("2 January 2006")})
	if err != nil {
		return 0, fmt.Errorf("get-day-task-record: %w", err)
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		// Declare a result BSON object
		var result storage.TaskRecord
		err := cursor.Decode(&result)
		if err != nil {
			return 0, fmt.Errorf("get-day-task-record: %w", err)
		}
		taskResult += result.TimeDuration
	}

	return taskResult, nil

}

func (s *Storage) GetTasksbyPriority(groupName string) ([]entity.TaskDefinition, error) {
	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)
	var tasksDefinition []entity.TaskDefinition

	// Define filter variable outside if-else blocks
	var filter bson.M

	// Set filter based on groupName
	if groupName == "plan" {
		filter = bson.M{"date": time.Now().Format("2 January 2006")}
	} else {
		filter = bson.M{"role": groupName, "date": time.Now().Format("2 January 2006")}
	}

	opts := options.Find().SetSort(bson.M{"priority": -1})

	cursor, err := coll.Find(s.Context, filter, opts)
	if err != nil {
		// Return an error if there was a problem finding the document
		return nil, fmt.Errorf("error in GetTaskNamePlanPercent: %s", err)
	}

	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		// Declare a result BSON object
		var result entity.TaskDefinition
		err := cursor.Decode(&result)
		if err != nil {
			// Return an error if there was a problem decoding the document
			return nil, fmt.Errorf("error in GetTaskNamePlanPercent: %s", err)
		}
		tasksDefinition = append(tasksDefinition, result)
	}

	// Return the task config
	return tasksDefinition, nil
}

func (s *Storage) StatisticTaskGet(taskName string) (int, error) {
	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)
	var taskResult int

	// Search and Collect info from Tasks
	cursor, err := coll.Find(s.Context, bson.D{{"name", taskName}, {"date", time.Now().Format("2 January 2006")}})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		// Declare a result BSON object
		var result storage.TaskRecord
		err := cursor.Decode(&result)
		if err != nil {
			return 0, err
		}
		taskResult += result.TimeDuration
	}

	return taskResult, nil
}

func (s *Storage) CreateTask(taskDefinition entity.TaskDefinition) error {
	// Validate input
	if taskDefinition.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if taskDefinition.Role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	// Validate that the role is correct
	if err := CorrectRoleCheck(taskDefinition.Role); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	// Check if task already exists
	coll := s.Client.Database(dbName).Collection(taskNamesList)
	filter := bson.M{"name": taskDefinition.Name}

	// Check if task exists
	var existingTask entity.TaskDefinition
	err := coll.FindOne(s.Context, filter).Decode(&existingTask)

	if err == nil {
		// Task exists - update it with new values (allows updating params during the day)
		todayDate := time.Now().Format("2 January 2006")
		taskDefinition.Date = todayDate
		update := bson.M{
			"$set": bson.M{
				"role":         taskDefinition.Role,
				"timeschedule": taskDefinition.TimeSchedule,
				"priority":     taskDefinition.Priority,
				"date":         taskDefinition.Date,
			},
		}
		_, err = coll.UpdateOne(s.Context, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update existing task: %w", err)
		}
		return nil
	}

	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to check if task exists: %w", err)
	}

	// Task doesn't exist - create new one
	taskDefinition.Date = time.Now().Format("2 January 2006")

	// Insert into database
	_, err = coll.InsertOne(s.Context, taskDefinition)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}
