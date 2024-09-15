package mongo

import "tracker-server/internal/domain/entity"

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
