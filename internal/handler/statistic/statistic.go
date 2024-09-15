package statistic

import (
	"tracker-server/internal/storage"
)

type Statistic struct {
	st storage.Storage
}

type StatisticCompletion struct {
	CompletionTime    int `json:"completion_time"`
	CompletionPercent int `json:"completion_percent"`
}

func New(st storage.Storage) *Statistic {
	return &Statistic{st: st}
}

// TODO: Finish this function
// func (s *Statistic) StatCompletionTimeDone(c *fiber.Ctx) error {
// 	slog.Info("Get request StatCompletionTimeDone")

// 	// var answer StatisticCompletion

// 	statsData, err := s.st.GetTaskRecordToday(options.WithCheckBusinessDay(true))
// 	if err != nil {
// 		return c.Status(500).JSON(&fiber.Map{
// 			"status":  "error",
// 			"message": err.Error(),
// 		})
// 	}

// 	return c.Status(200).JSON(statsData)

// 	return c.SendString("Hello, World!")
// }
