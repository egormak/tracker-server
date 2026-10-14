package entity

import "time"

// UsageCounter counts one kind of user choice (Event) made from one client within one
// calendar week. Week is the Monday date in the "2 January 2006" layout. Counters are
// wiped together with the task records.
type UsageCounter struct {
	Week   string `bson:"week" json:"week"`
	Event  string `bson:"event" json:"event"`
	Client string `bson:"client" json:"client"`
	Count  int    `bson:"count" json:"count"`
}

// DigestSnapshot is the single stored weekly digest, taken right before the weekly wipe.
// Payload is the JSON of a WeeklyDigest.
type DigestSnapshot struct {
	WeekStartDate string    `bson:"week_start_date"`
	TakenAt       time.Time `bson:"taken_at"`
	Payload       string    `bson:"payload"`
}
