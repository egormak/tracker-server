package entity

import "time"

type RunningTask struct {
	ID          string    `bson:"_id,omitempty"`
	TaskName    string    `bson:"task_name"`
	Role        string    `bson:"role"`
	StartTime   time.Time `bson:"start_time"`
	Accumulated int       `bson:"accumulated"` // accumulated seconds before last start
	IsRunning   bool      `bson:"is_running"`
}
