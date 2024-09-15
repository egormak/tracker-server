package mongo

import (
	"fmt"
	"time"
	"tracker-server/internal/service"
	"tracker-server/internal/storage"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Storage) ProcentsSet(procentM storage.Procents) error {

	planProcentDate, err := s.getDatePlanPercent()
	if err != nil {
		if err != storage.ErrListEmpty {
			return err
		}
	}

	if planProcentDate != time.Now().Format("2 January 2006") || planProcentDate == "" {
		if service.IsWeekendNow() {
			procentM.Plans = PlanTypesWeekEndsDays
			fmt.Println("Plans Weekends: ", procentM.Plans)
		} else {
			procentM.Plans = PlanTypesWeekDays
			fmt.Println("Plans Weekdays: ", procentM.Plans)
		}
		fmt.Println("Plans: ", procentM.Plans)
		procentM.Date = time.Now().Format("2 January 2006")
		procentM.CurrentChoice = 0
		procentM.Title = procentDocName
	}

	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	// Update the RestData document
	filter := bson.M{"title": procentDocName}
	update := bson.M{"$set": procentM}
	options := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(s.Context, filter, update, options)
	if err != nil {
		// Return an error if there was a problem updating the document
		return fmt.Errorf("procents-set error update-one: %s", err)
	}

	return nil
}

func (s *Storage) GetGroupPlanPercent() (int, error) {

	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	// Find the document
	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		// Return an error if there was a problem finding the document
		return 0, fmt.Errorf("error in GetTaskPlanPercent: %s", err)
	}

	return procentM.CurrentChoice, nil

}

func (s *Storage) GetPlanProcents() (storage.Procents, error) {
	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	// Find the document
	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return storage.Procents{}, storage.ErrListEmpty
		}
		// Return an error if there was a problem finding the document
		return storage.Procents{}, fmt.Errorf("get-plan-procents error in findone: %s", err)
	}

	return procentM, nil
}

func (s *Storage) ChangeGroupPlanPercent(groupPlane int) error {
	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		// Return an error if there was a problem finding the document
		return fmt.Errorf("change-group-plan-percent error in findone: %s", err)
	}

	if groupPlane == len(procentM.Plans)-1 {
		groupPlane = 0
	} else {
		groupPlane++
	}

	// Remove first element in array
	filter := bson.M{"title": procentDocName}
	update := bson.M{"$set": bson.M{"currentchoice": groupPlane}}

	_, err = coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		// Return an error if there was a problem updating the document
		return fmt.Errorf("change-group-plan-percent error in updateone: %s", err)
	}

	return nil
}

func (s *Storage) GetGroupPercent(groupPlan int) (int, error) {

	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	// Find the document
	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		// Return an error if there was a problem finding the document
		return 0, fmt.Errorf("error in GetTaskPercent: %s", err)
	}

	groupPlanName := procentM.Plans[groupPlan]

	switch groupPlanName {
	case PlanTypes[0]:
		if len(procentM.Plan) == 0 {
			return 0, storage.ErrListEmpty
		}
		return procentM.Plan[0], nil
	case PlanTypes[1]:
		if len(procentM.Work) == 0 {
			return 0, storage.ErrListEmpty
		}
		return procentM.Work[0], nil
	case PlanTypes[2]:
		if len(procentM.Learn) == 0 {
			return 0, storage.ErrListEmpty
		}
		return procentM.Learn[0], nil
	case PlanTypes[3]:
		if len(procentM.Rest) == 0 {
			return 0, storage.ErrListEmpty
		}
		return procentM.Rest[0], nil
	}

	return 0, fmt.Errorf("error in GetTaskPercent")
}

func (s *Storage) DelGroupPercent(groupPlanName string) error {

	// Connect to the database
	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	// Remove first element in array
	filter := bson.M{"title": procentDocName}
	update := bson.M{"$pop": bson.M{groupPlanName: -1}}

	_, err := coll.UpdateOne(s.Context, filter, update)
	if err != nil {
		// Return an error if there was a problem finding the document
		return fmt.Errorf("error in DelGroupPercent: %s", err)
	}

	return nil
}

func (s *Storage) GetTaskNamePlanPercent(groupName string, groupPercent int) (string, error) {

	PriorityTasks, err := s.GetTasksbyPriority(groupName)
	if err != nil {
		return "", fmt.Errorf("error in GetTaskNamePlanPercent: %s", err)
	}

	// Get Valid TaskName
	for _, v := range PriorityTasks {
		taskTimeDone, err := s.StatisticTaskGet(v.Name)
		if err != nil {
			return "", fmt.Errorf("error in GetTaskNamePlanPercent: %s", err)
		}
		taskTimeLeft := (v.TimeSchedule*groupPercent)/100 - taskTimeDone

		if taskTimeLeft > 0 {
			return v.Name, nil
		}
	}

	return "", nil

}

func (s *Storage) GetGroupName(groupNameOrdinal int) (string, error) {

	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		// Return an error if there was a problem finding the document
		return "", fmt.Errorf("get-group-name error in findone: %s", err)
	}

	return procentM.Plans[groupNameOrdinal], nil
}

func (s *Storage) getDatePlanPercent() (string, error) {

	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return an error if there was a problem finding the document
			return "", storage.ErrListEmpty
		}
		// Return an error if there was a problem finding the document
		return "", fmt.Errorf("get-date-plan-percent error in findone: %s", err)
	}

	return procentM.Date, nil
}

func (s *Storage) CheckIfPlanPercentEmpty() error {

	database := s.Client.Database(dbName)
	coll := database.Collection(taskInfo)

	var procentM storage.Procents
	err := coll.FindOne(s.Context, bson.M{"title": procentDocName}).Decode(&procentM)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return an error if there was a problem finding the document
			return storage.ErrListEmpty
		}
		// Return an error if there was a problem finding the document
		return fmt.Errorf("check-if-plan-percent-empty error in findone: %s", err)
	}

	if len(procentM.Rest) == 0 && len(procentM.Learn) == 0 && len(procentM.Work) == 0 && len(procentM.Plan) == 0 {
		return storage.ErrListEmpty

	} else {
		return nil
	}
}
