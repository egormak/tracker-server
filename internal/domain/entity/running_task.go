package entity

import "time"

type RunningTask struct {
	ID             string    `bson:"_id,omitempty" json:"id"`
	TaskName       string    `bson:"task_name" json:"task_name"`
	Role           string    `bson:"role" json:"role"`
	StartTime      time.Time `bson:"start_time" json:"start_time"`
	Accumulated    int       `bson:"accumulated" json:"accumulated"` // accumulated seconds before last start
	IsRunning      bool      `bson:"is_running" json:"is_running"`
	TargetDuration int       `bson:"target_duration" json:"target_duration"` // planned duration (minutes)
	SourceDay      string    `bson:"source_day" json:"source_day"`           // rollover day origin
}
