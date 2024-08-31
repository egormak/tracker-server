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

// type TaskRecord struct {
// 	Name         string
// 	Role         string
// 	TimeDuration int
// 	Date         string
// }

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

// AddTaskRecord adds a task record to the storage.
func (s *Storage) AddTaskRecord(task entity.TaskRecord) error {
	// Get the database
	database := s.Client.Database(dbName)

	// Get the collection
	coll := database.Collection(tasksList)

	// Insert the task record into the collection
	_, err := coll.InsertOne(s.Context, task)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) ShowTaskList() ([]storage.TaskResult, error) {

	var taskConfigs []storage.TaskConfig
	var taskResults []storage.TaskResult
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	cursor, err := coll.Find(s.Context, bson.M{"date": time.Now().Format("2 January 2006")})
	if err != nil {
		return nil, fmt.Errorf("show-task-list: %w", err)
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		var taskConf storage.TaskConfig
		err := cursor.Decode(&taskConf)
		if err != nil {
			return nil, fmt.Errorf("show-task-list: %w", err)
		}
		taskConfigs = append(taskConfigs, taskConf)
	}

	for _, taskData := range taskConfigs {
		timeDuration, err := s.TaskRecordTimeTodayGetDB(taskData.Name)
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

func (s *Storage) TaskRecordTimeTodayGetDB(taskName string) (int, error) {

	// Select Value
	var timeDuretion int

	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)

	// Get Information about tasks
	cursor_task, err := coll.Find(s.Context, bson.M{"name": taskName, "date": time.Now().Format("2 January 2006")})
	if err != nil {
		return 0, fmt.Errorf("task-record-time-today-get-db: %w", err)
	}
	defer cursor_task.Close(s.Context)

	for cursor_task.Next(s.Context) {
		// Declare a result BSON object
		var result storage.TaskRecord
		err := cursor_task.Decode(&result)
		if err != nil {
			return 0, fmt.Errorf("task-record-time-today-get-db: %w", err)
		}
		timeDuretion += result.TimeDuration
	}
	return timeDuretion, nil

}

func (s *Storage) SetTaskParams(params storage.TaskParams) error {

	var result storage.TaskConfig

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

func (s *Storage) GetTaskParams(taskName string) (storage.TaskParams, error) {

	// Set Value for DB
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	// Find Collection
	var result storage.TaskConfig
	err := coll.FindOne(s.Context, bson.D{{"name", taskName}}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return storage.TaskParams{}, fmt.Errorf("get-task-params: %w", mongo.ErrNoDocuments)
	}

	if result.Date != time.Now().Format("2 January 2006") {
		return storage.TaskParams{}, storage.ErrParamsOld
	}

	return storage.TaskParams{
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

func (s *Storage) GetTasksbyPriority(groupName string) ([]storage.TaskConfig, error) {
	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)
	var tasksConfig []storage.TaskConfig

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
		var result storage.TaskConfig
		err := cursor.Decode(&result)
		if err != nil {
			// Return an error if there was a problem decoding the document
			return nil, fmt.Errorf("error in GetTaskNamePlanPercent: %s", err)
		}
		tasksConfig = append(tasksConfig, result)
	}

	// Return the task config
	return tasksConfig, nil
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
