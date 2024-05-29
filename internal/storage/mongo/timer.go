package mongo

import (
	"time"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TimeListSetDB populates the "Day List" document in the database with a new DayList struct.
// The count parameter determines the size of the ListTime slice in the DayList struct.
// If the document doesn't exist, it will be created.
func (s *Storage) TimeListSetDB(count int) error {
	// Create a new DayList struct with the given count
	result := storage.DayList{
		Title:    "Day List",
		Count:    count,
		ListTime: make([]int, count),
	}

	// Populate the ListTime slice with values from 1 to count
	for i := 0; i < count; i++ {
		result.ListTime[i] = i + 1
	}

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	// Define the filter to find the document with title "Day List"
	filter := bson.D{{"title", "Day List"}}

	// Define the update with the new DayList struct
	update := bson.D{{"$set", result}}

	// Set the options to upsert the document if it doesn't exist
	opts := options.Update().SetUpsert(true)

	// Update the document with the new DayList struct
	_, err := coll.UpdateOne(s.Context, filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) TimeListGetDB() ([]int, error) {

	// Set Value
	var result storage.DayList

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	//Check if Exists Document
	err := coll.FindOne(s.Context, bson.D{{"title", "Day List"}}).Decode(&result)
	if err != nil {
		return nil, err
	}

	if len(result.ListTime) == 0 {
		return nil, storage.ErrListEmpty
	}

	return result.ListTime, nil
}

func (s *Storage) TimeListDelDB(timeDuretion int) error {

	// Set Value
	var result storage.DayList

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	//Check if Exists Document
	err := coll.FindOne(s.Context, bson.D{{"title", "Day List"}}).Decode(&result)
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
	filter := bson.D{{"title", "Day List"}}
	update := bson.D{{"$set", result}}
	_, err = coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) TimeDurationGet() (int, error) {

	var timeValue int

	// Get Time
	timeList, err := s.TimeListGetDB()
	if err != nil {
		// MongoDocument is not exist
		if err == mongo.ErrNoDocuments {

			s.TimeListSetDB(1)
			timeList, err = s.TimeListGetDB()
			if err != nil {
				return 0, err
			}
		} else if err == storage.ErrListEmpty {
			count := s.TimeListCountGetDB()
			s.TimeListSetDB(count + 1)
			timeList, err = s.TimeListGetDB()
			if err != nil {

				return 0, err
			}
		}
	}

	timeValue = timeList[0]

	return timeValue, nil
}

func (s *Storage) TimeListCountGetDB() int {

	// Set Value
	var result storage.DayList

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	//Check if Exists Document
	coll.FindOne(s.Context, bson.D{{"title", "Day List"}}).Decode(&result)

	return result.Count
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
		var result storage.TaskRecord
		err := cursor.Decode(&result)
		if err != nil {
			return 0, err
		}
		timeTasks += result.TimeDuration
	}

	return timeTasks, nil
}

func (s *Storage) TimerGlobalSet(timeScheduler int) error {

	// Set Value for DB

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	SchedulerRecord := storage.SchedulerInfo{
		Name:        "Scheduler",
		Date:        time.Now().Format("2 January 2006"),
		ScheduleAll: timeScheduler,
	}

	filter := bson.D{{"name", "Scheduler"}}

	update := bson.D{{"$set", SchedulerRecord}}

	opts := options.Update().SetUpsert(true)

	_, err := coll.UpdateOne(s.Context, filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) TimerGlobalGet() (int, error) {

	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	var result storage.SchedulerInfo

	err := coll.FindOne(s.Context, bson.D{{"name", "Scheduler"}}).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.ScheduleAll, nil
}
