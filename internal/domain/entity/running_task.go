package entity

import "time"

type RunningTask struct {
	ID                string    `bson:"_id,omitempty" json:"id"`
	TaskName          string    `bson:"task_name" json:"task_name"`
	Role              string    `bson:"role" json:"role"`
	StartTime         time.Time `bson:"start_time" json:"start_time"`
	Accumulated       int       `bson:"accumulated" json:"accumulated"` // accumulated minutes before last start
	IsRunning         bool      `bson:"is_running" json:"is_running"`
	TargetDuration    int       `bson:"target_duration" json:"target_duration"` // planned duration (minutes)
	SourceDay         string    `bson:"source_day,omitempty" json:"source_day,omitempty"`           // rollover day origin
	LastHeartbeatAt   time.Time `bson:"last_heartbeat_at,omitempty" json:"last_heartbeat_at,omitempty"`
	DeadlineAt        time.Time `bson:"deadline_at,omitempty" json:"deadline_at,omitempty"`
	TelegramMessageID int       `bson:"telegram_message_id,omitempty" json:"telegram_message_id,omitempty"`
}
