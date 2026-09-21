package mongo

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Storage) GetRecords() ([]entity.TaskRecord, error) {

	var taskRecords []entity.TaskRecord

	coll := s.Client.Database(dbName).Collection(tasksList)
	cursor, err := coll.Find(s.Context, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		// Declare a result BSON object
		var taskRecord entity.TaskRecord
		err := cursor.Decode(&taskRecord)
		if err != nil {
			return nil, err
		}
		taskRecords = append(taskRecords, taskRecord)
	}

	return taskRecords, nil
}

func (s *Storage) CleanRecords() {
	database := s.Client.Database(dbName)

	collRoleInfo := database.Collection(roleInfo)
	collTaskInfo := database.Collection(taskInfo)
	collTasks := database.Collection(tasksList)
	collRunning := database.Collection(runningTaskCollection)

	// Replace Drop with DeleteMany to preserve collection indexes!
	if _, err := collTasks.DeleteMany(s.Context, bson.M{}); err != nil {
		slog.Error("clean-records: failed to delete tasks", "error", err)
	}

	// Flush running_task collection to remove stale zombie timers across weeks!
	if _, err := collRunning.DeleteMany(s.Context, bson.M{}); err != nil {
		slog.Error("clean-records: failed to delete running tasks", "error", err)
	}

	// In task_info, reset Rest count to 0 while preserving Procent Info and Day List!
	today := time.Now().Format("2 January 2006")
	if _, err := collTaskInfo.UpdateOne(
		s.Context,
		bson.M{"title": restDocName},
		bson.M{"$set": bson.M{"restcount": 0, "date": today}},
		options.Update().SetUpsert(true),
	); err != nil {
		slog.Error("clean-records: failed to reset rest count in task_info", "error", err)
	}

	// In role_info: reset via DeleteMany
	if _, err := collRoleInfo.DeleteMany(s.Context, bson.M{}); err != nil {
		slog.Error("clean-records: failed to delete role_info", "error", err)
	}
}

type taskDurationAggResult struct {
	Name  string `bson:"_id"`
	Total int    `bson:"total"`
}

// GetTodayTaskDurationsMap returns a map of task name to total duration for the given date using an aggregation pipeline.
func (s *Storage) GetTodayTaskDurationsMap(date string) (map[string]int, error) {
	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"date": date}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$name",
			"total": bson.M{"$sum": "$time_duration"},
		}}},
	}

	cursor, err := coll.Aggregate(s.Context, pipeline)
	if err != nil {
		return nil, fmt.Errorf("get-today-task-durations-map: %w", err)
	}
	defer cursor.Close(s.Context)

	durations := make(map[string]int)
	for cursor.Next(s.Context) {
		var agg taskDurationAggResult
		if err := cursor.Decode(&agg); err != nil {
			return nil, fmt.Errorf("get-today-task-durations-map decode: %w", err)
		}
		durations[agg.Name] = agg.Total
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("get-today-task-durations-map cursor error: %w", err)
	}

	return durations, nil
}

func (s *Storage) ShowTaskList() ([]entity.TaskResult, error) {
	var taskDefinitions []entity.TaskDefinition
	taskResults := make([]entity.TaskResult, 0)
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	today := time.Now().Format("2 January 2006")
	cursor, err := coll.Find(s.Context, bson.M{"date": today})
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

	// Single aggregation pipeline query replacing N+1 per-task queries
	durationsMap, err := s.GetTodayTaskDurationsMap(today)
	if err != nil {
		return nil, fmt.Errorf("show-task-list durations: %w", err)
	}

	for _, taskData := range taskDefinitions {
		taskResults = append(taskResults, entity.TaskResult{
			Name:         taskData.Name,
			Role:         taskData.Role,
			TimeDuration: taskData.TimeSchedule,
			TimeDone:     durationsMap[taskData.Name],
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
		var result entity.TaskRecord
		err := cursor_task.Decode(&result)
		if err != nil {
			return 0, fmt.Errorf("get-today-task-duration: %w", err)
		}
		timeDuretion += result.TimeDuration
	}
	return timeDuretion, nil

}

// getWeekDates returns all 7 date strings (Monday to Sunday) for the calendar week containing dateStr.
// If dateStr cannot be parsed using "2 January 2006", it returns []string{dateStr}.
func getWeekDates(dateStr string) []string {
	parsed, err := time.Parse("2 January 2006", dateStr)
	if err != nil {
		return []string{dateStr}
	}
	weekday := parsed.Weekday()
	daysSinceMonday := int(weekday) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday is day 6 relative to Monday
	}
	monday := parsed.AddDate(0, 0, -daysSinceMonday)
	weekDates := make([]string, 7)
	for i := 0; i < 7; i++ {
		weekDates[i] = monday.AddDate(0, 0, i).Format("2 January 2006")
	}
	return weekDates
}

// GetTaskDurationForDate gets the total duration for a task on a specific date and source day.
// Rollover source_day matches are strictly bounded to the week of the queried date,
// preventing unbounded historical over-crediting from prior weeks.
func (s *Storage) GetTaskDurationForDate(taskName string, date string, sourceDay string) (int, error) {
	var timeDuration int

	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)

	var filter bson.M
	if sourceDay != "" {
		weekDates := getWeekDates(date)
		filter = bson.M{
			"name": taskName,
			"$or": []bson.M{
				{"date": date, "source_day": bson.M{"$in": []interface{}{"", nil}}},
				{"source_day": strings.ToLower(sourceDay), "date": bson.M{"$in": weekDates}},
			},
		}
	} else {
		filter = bson.M{
			"name":       taskName,
			"date":       date,
			"source_day": bson.M{"$in": []interface{}{"", nil}},
		}
	}

	// Get Information about tasks for the specific date/day
	cursor_task, err := coll.Find(s.Context, filter)
	if err != nil {
		return 0, fmt.Errorf("get-task-duration-for-date: %w", err)
	}
	defer cursor_task.Close(s.Context)

	for cursor_task.Next(s.Context) {
		var result entity.TaskRecord
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

func (s *Storage) IsTaskStrict(taskName string) (bool, error) {
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	var result entity.TaskDefinition
	err := coll.FindOne(s.Context, bson.D{{"name", taskName}}).Decode(&result)
	if err != nil {
		if strings.EqualFold(taskName, "work") || strings.EqualFold(taskName, "english") {
			return true, nil
		}
		return false, nil
	}

	return result.TimeStrictly, nil
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
		var result entity.TaskRecord
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
		var result entity.TaskRecord
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

// GetTaskNamesForDate returns all task names for a specific date
func (s *Storage) GetTaskNamesForDate(date string) ([]string, error) {
	coll := s.Client.Database(dbName).Collection(taskNamesList)

	cursor, err := coll.Find(s.Context, bson.M{"date": date})
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for date: %w", err)
	}
	defer cursor.Close(s.Context)

	var taskNames []string
	for cursor.Next(s.Context) {
		var task entity.TaskDefinition
		if err := cursor.Decode(&task); err != nil {
			return nil, fmt.Errorf("failed to decode task: %w", err)
		}
		taskNames = append(taskNames, task.Name)
	}

	return taskNames, nil
}

// MoveTaskToPreviousDate moves a task to the previous day by updating its date
func (s *Storage) MoveTaskToPreviousDate(taskName string, currentDate string) error {
	coll := s.Client.Database(dbName).Collection(taskNamesList)

	// Parse the current date
	parsedDate, err := time.Parse("2 January 2006", currentDate)
	if err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}

	// Set date to previous day
	previousDate := parsedDate.AddDate(0, 0, -1).Format("2 January 2006")

	filter := bson.M{"name": taskName, "date": currentDate}
	update := bson.M{
		"$set": bson.M{
			"date": previousDate,
		},
	}

	result, err := coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return fmt.Errorf("failed to move task to previous date: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("task not found: %s", taskName)
	}

	return nil
}
