package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RestData struct {
	RestCount int
	Date      string
}

// AddRest adds the specified rest time to the RestData collection in the database.
// It returns an error if there was a problem adding the rest time.
func (s *Storage) AddRest(restTime int) error {
	// Multiply rest time by the rest count
	restTime *= restCount

	// Connect to the RestData collection
	coll := s.Client.Database(dbName).Collection(taskInfo)

	// Find the RestData document with the given title
	filter := bson.D{{"title", restDocName}}
	var result RestData
	err := coll.FindOne(s.Context, filter).Decode(&result)
	if err != nil {
		// If no document is found, set rest count to 0
		if err == mongo.ErrNoDocuments {
			result.RestCount = 0
		} else {
			// Return an error if there was a decoding error
			return fmt.Errorf("error in AddRest: %s", err)
		}
	}

	// If the date is not today, reset the rest count to 0
	if result.Date != time.Now().Format("2 January 2006") {
		result.RestCount = 0
	}

	// Add rest time to the rest count and update the date
	result.RestCount += restTime
	result.Date = time.Now().Format("2 January 2006")

	// Update the RestData document
	update := bson.D{{"$set", result}}
	options := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(s.Context, filter, update, options)
	if err != nil {
		// Return an error if there was a problem updating the document
		return fmt.Errorf("error in AddRest: %s", err)
	}

	return nil
}

func (s *Storage) RestSpend(restTime int) error {

	var result RestData

	// Connect to Collection
	coll := s.Client.Database(dbName).Collection(taskInfo)

	// Find Collection
	filter := bson.D{{"title", restDocName}}
	err := coll.FindOne(s.Context, filter).Decode(&result)
	if err != nil {
		// If no matching document is found, set restTime to 0
		if err == mongo.ErrNoDocuments {
			restTime = 0
		} else {
			return fmt.Errorf("error occurred in restspend: %s", err)
		}
	}

	// If the date in the result is not today, reset the rest count to 0
	if result.Date != time.Now().Format("2 January 2006") {
		result.RestCount = 0
	}

	// Subtract restTime from the rest count
	result.RestCount -= restTime * 100
	result.Date = time.Now().Format("2 January 2006")

	// Update Record
	update := bson.D{{"$set", result}}
	options := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(s.Context, filter, update, options)
	if err != nil {
		return fmt.Errorf("error occurred in restspend: %s", err)
	}

	return nil

}

func (s *Storage) GetRest() (int, error) {

	var result RestData

	// Connect to Collection
	coll := s.Client.Database(dbName).Collection(taskInfo)

	// Find Collection
	filter := bson.D{{"title", restDocName}}
	err := coll.FindOne(s.Context, filter).Decode(&result)
	if err != nil {
		// If no matching document is found, set restTime to 0
		if err == mongo.ErrNoDocuments {
			return 0, nil
		} else {
			return 0, fmt.Errorf("error occurred in restspend: %s", err)
		}
	}

	// If the date in the result is not today, reset the rest count to 0
	if result.Date != time.Now().Format("2 January 2006") {
		result.RestCount = 0
	}

	return result.RestCount, nil

}
