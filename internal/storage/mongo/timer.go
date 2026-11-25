package mongo

import (
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Storage) TimeListDelDB(timeDuretion int) error {

	// Set Value
	var result storage.DayList

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	//Check if Exists Document
	err := coll.FindOne(s.Context, bson.D{{Key: "title", Value: "Day List"}}).Decode(&result)
	if err != nil {
		return err
	}
	// Find Value in Array
	for i, v := range result.ListTime {
		if v == timeDuretion {
			result.ListTime = append(result.ListTime[:i], result.ListTime[i+1:]...)
			break
		}
	}

	// Update Record
	filter := bson.D{{Key: "title", Value: "Day List"}}
	update := bson.D{{Key: "$set", Value: result}}
	_, err = coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return err
	}
	return nil
}

// Get the total time that was spent
func (s *Storage) TimeTasks() (int, error) {

	var timeTasks int

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(tasksList)

	//Find All tasks
	cursor, err := coll.Find(s.Context, bson.D{})
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
		timeTasks += result.TimeDuration
	}

	return timeTasks, nil
}

func (s *Storage) TimerGlobalSet(timeScheduler int, date ...string) error {

	// Set Value for DB

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	targetDate := time.Now().Format("2 January 2006")
	if len(date) > 0 && date[0] != "" {
		targetDate = date[0]
	}

	SchedulerRecord := storage.SchedulerInfo{
		Name:        "Scheduler",
		Date:        targetDate,
		ScheduleAll: timeScheduler,
	}

	filter := bson.D{{Key: "name", Value: "Scheduler"}}

	update := bson.D{{Key: "$set", Value: SchedulerRecord}}

	opts := options.Update().SetUpsert(true)

	_, err := coll.UpdateOne(s.Context, filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}
