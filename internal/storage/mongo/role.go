package mongo

import (
	"fmt"
	"time"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// StatisticRolesGet retrieves the role statistics from the storage.
// It returns a slice of RoleRecord structs and an error if any.
func (s *Storage) StatisticRolesGet() ([]storage.RoleRecord, error) {
	// Get the database and collection
	database := s.Client.Database(dbName)
	coll_role_stat := database.Collection(roleInfo)

	// Find all roles in the collection
	cursor_roles, err := coll_role_stat.Find(s.Context, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor_roles.Close(s.Context)

	// Retrieve all role records in a single operation
	var resultData []storage.RoleRecord
	if err := cursor_roles.All(s.Context, &resultData); err != nil {
		return nil, err
	}

	return resultData, nil
}

func (s *Storage) StatisticRolesGetToday() ([]storage.RoleRecord, error) {
	// Get the database and collection
	database := s.Client.Database(dbName)
	coll := database.Collection(roleInfo)

	// Find all roles in the collection
	cursor, err := coll.Find(s.Context, bson.M{"recorddate": time.Now().Format("2 January 2006")})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.Context)

	var resultData []storage.RoleRecord
	if err := cursor.All(s.Context, &resultData); err != nil {
		return nil, err
	}

	return resultData, nil
}

// RecheckRole rechecks the roles in the tasks list and updates the role info collection in the database.
func (s *Storage) RecheckRole() error {

	database := s.Client.Database(dbName)
	collTasks := database.Collection(tasksList)
	collInfo := database.Collection(roleInfo)

	// Drop the collection to remove the existing role info
	if err := collInfo.Drop(s.Context); err != nil {
		return err
	}

	cursor, err := collTasks.Find(s.Context, bson.D{})
	if err != nil {
		return err
	}
	defer cursor.Close(s.Context)

	for cursor.Next(s.Context) {
		var result entity.TaskRecord
		if err := cursor.Decode(&result); err != nil {
			return err
		}
		if err := s.AddRoleMinutes(result); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) AddRoleMinutes(t entity.TaskRecord) error {
	// Set Value for DB
	// client := dbClient.Client
	// ctx := dbClient.Context
	today := time.Now().Format("2 January 2006")

	var result storage.RoleRecord

	// Check if the role is correct
	if err := CorrectRoleCheck(t.Role); err != nil {
		return err
	}

	// Connect to the database and select the collection
	database := s.Client.Database(dbName)
	coll := database.Collection(roleInfo)

	// Find the role in the collection
	err := coll.FindOne(s.Context, bson.D{{"name", t.Role}}).Decode(&result)
	if err != nil {
		// If the role doesn't exist, insert a new record for it
		if err == mongo.ErrNoDocuments {
			result = storage.RoleRecord{Name: t.Role, Duration: t.TimeDuration, RecordDate: t.Date, DurationToday: t.TimeDuration}
			coll.InsertOne(s.Context, result)
			return nil
		} else {
			return err
		}
	}

	// Update the role's duration with the new time duration
	result.Duration += t.TimeDuration

	// If the date is today, update the duration today
	if t.Date == today {
		if result.RecordDate == today {
			result.DurationToday += t.TimeDuration
		} else {
			result.DurationToday = t.TimeDuration
			result.RecordDate = today
		}
	}

	// Update the record in the collection
	filter := bson.D{{"name", t.Role}}
	update := bson.D{{"$set", result}}
	_, err = coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		return err
	}
	return nil
}

// CorrectRoleCheck checks if a given roleName is a valid role.
// It returns an error if the roleName does not exist in the roleTypes slice.
func CorrectRoleCheck(roleName string) error {
	// Set roleCorrect to false initially
	roleCorrect := false

	// Iterate over the roleTypes slice
	for _, role := range roleTypes {
		// Check if the roleName matches any role in the slice
		if roleName == role {
			// Set roleCorrect to true if a match is found
			roleCorrect = true
		}
	}

	// If roleCorrect is false, return an error
	if !roleCorrect {
		return fmt.Errorf("role: %s does not exist, please check", roleName)
	}

	// If roleCorrect is true, return nil
	return nil
}

// GetRole retrieves the role associated with a task name from the storage.
// It returns the role and any error encountered during the retrieval process.
func (s *Storage) GetRole(taskName string) (string, error) {
	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskNamesList)

	// Find the task configuration in the collection
	var result entity.TaskDefinition
	err := coll.FindOne(s.Context, bson.D{{"name", taskName}}).Decode(&result)
	if err != nil {
		// Return an error if the task configuration is not found
		if err == mongo.ErrNoDocuments {
			return "", err
		}
		return "", fmt.Errorf("error decoding result: %v", err)
	}

	// Return the role associated with the task name
	return result.Role, nil
}
