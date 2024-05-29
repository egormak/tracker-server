package record

import (
	"fmt"
	"time"
	"tracker-server/internal/notify"
	"tracker-server/internal/service"
	"tracker-server/internal/storage"

	log "github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
)

type Record struct {
	st  storage.Storage
	ntf notify.Notify
}

func New(st storage.Storage, ntf notify.Notify) *Record {
	return &Record{st: st, ntf: ntf}
}

// ShowRecords returns a JSON response with task records for today, yesterday, and all time.
func (r *Record) ShowRecords(c *fiber.Ctx) error {
	// Get today's and yesterday's dates in the required format
	today := time.Now().Format("2 January 2006")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2 January 2006")

	// Retrieve task records from the storage
	records, err := r.st.GetRecords()
	if err != nil {
		// Return an error response if there was an issue retrieving records
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Initialize maps to store task records for today, yesterday, and all time
	taskRecordsToday := make(map[string]int)
	taskRecordsYesterday := make(map[string]int)
	taskRecordsAll := make(map[string]int)

	// Iterate over the records and update the corresponding maps based on the date
	for _, v := range records {
		if v.Date == today {
			// Add the time duration to the task's record for today
			taskRecordsToday[v.Name] += v.TimeDuration
		} else if v.Date == yesterday {
			// Add the time duration to the task's record for yesterday
			taskRecordsYesterday[v.Name] += v.TimeDuration
		}

		// Add the time duration to the task's record for all time
		taskRecordsAll[v.Name] += v.TimeDuration
	}

	// Create a response map with task records for today, yesterday, and all time
	answer := map[string]map[string]int{
		"today":     taskRecordsToday,
		"yesterday": taskRecordsYesterday,
		"all":       taskRecordsAll,
	}

	// Return the response as JSON
	return c.JSON(answer)
}

func (r *Record) AddRecord(c *fiber.Ctx) error {
	log.Info("Get request addRecord")

	var body struct {
		TaskName string `json:"task_name"`
		TimeDone int    `json:"time_done"`
	}

	if err := c.BodyParser(&body); err != nil {
		errMsg := fmt.Errorf("can't parse body: %s", err)
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if body.TaskName == "" {
		errMsg := "Task name is not Set"
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg,
		})
	}

	if body.TimeDone == 0 {
		errMsg := "Task duration is not Set"
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg,
		})
	}

	taskRole, err := r.st.GetRole(body.TaskName)
	if err != nil {
		errMsg := fmt.Errorf("task role can't get: %s", err)
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	record := storage.TaskRecord{
		Name:         body.TaskName,
		Role:         taskRole,
		TimeDuration: body.TimeDone,
		Date:         time.Now().Format("2 January 2006"),
	}

	if err := r.st.AddTaskRecord(record); err != nil {
		errMsg := fmt.Errorf("can't add record: %s", err)
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if err := r.st.AddRoleMinutes(record); err != nil {
		errMsg := fmt.Errorf("can't add role minutes: %s", err)
		log.Error(errMsg)
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": errMsg.Error(),
		})
	}

	if service.IsWeekendNow() || taskRole != "rest" {
		if err := r.st.AddRest(body.TimeDone); err != nil {
			errMsg := fmt.Errorf("can't add rest: %s", err)
			log.Error(errMsg)
			return c.Status(500).JSON(&fiber.Map{
				"status":  "error",
				"message": errMsg.Error(),
			})
		}
	}

	log.Info("Record was added")
	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Record was added",
	})
}

func (r *Record) CleanRecords(c *fiber.Ctx) error {

	log.Infof("Start Clean Data")
	r.st.CleanRecords()
	log.Info("Finish Data Clean")

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Data was erased",
	})
}

func (r *Record) ShowTaskList(c *fiber.Ctx) error {

	result, err := r.st.ShowTaskList()
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(result)
}

func (r *Record) SetTaskParams(c *fiber.Ctx) error {

	var body struct {
		Name         string `json:"name"`
		TimeDuration int    `json:"time_duration"`
		Priority     int    `json:"priority"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	taskParams := storage.TaskParams{
		Name:     body.Name,
		Time:     body.TimeDuration,
		Priority: body.Priority,
	}

	if err := r.st.SetTaskParams(taskParams); err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":  "accept",
		"message": "Params was set",
	})

}

func (r *Record) GetTaskParams(c *fiber.Ctx) error {

	result, err := r.st.GetTaskParams(c.Query("task_name"))
	if err != nil {
		if err == storage.ErrTaskNotFound {
			return c.Status(404).JSON(&fiber.Map{
				"status":  "Task Not Found",
				"message": err.Error(),
			})
		}
		if err == storage.ErrParamsOld {
			return c.Status(404).JSON(&fiber.Map{
				"status":  "params old",
				"message": err.Error(),
			})
		}
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(result)
}

func (r *Record) GetDayTaskRecord(c *fiber.Ctx) error {

	result, err := r.st.GetDayTaskRecord(c.Query("task_name"))
	if err != nil {
		return c.Status(500).JSON(&fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(200).JSON(&fiber.Map{
		"status":        "Done",
		"task_duration": result,
	})
}

func (r *Record) TimerDel(c *fiber.Ctx) error {
	// Define a struct to hold the request body
	var body struct {
		Count int `json:"count"`
	}

	// Parse the request body and store it in the 'body' variable
	if err := c.BodyParser(&body); err != nil {
		log.Errorf("Error parsing request body: %v", err)
		return err
	}

	// Log the start of the timer set process
	log.Infof("Start Timer Del")

	// Set the timer value in the database based on the count
	r.st.TimeListDelDB(body.Count)

	// Log the completion of the timer set process
	log.Info("Finish Timer Del")

	// Create a response map with the status and message
	response := fiber.Map{
		"status":  "accept",
		"message": "Timer was Del",
	}

	// Log the response being returned
	log.Infof("Returning response: %v", response)

	// Return the response as a JSON with status 200
	return c.Status(200).JSON(&response)
}
