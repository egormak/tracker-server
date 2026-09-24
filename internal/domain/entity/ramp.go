package entity

import "time"

// RampInfo represents the warm-up ramp document stored in MongoDB task_info collection.
type RampInfo struct {
	Title               string    `bson:"title" json:"title"`
	CurrentStep         int       `bson:"current_step" json:"current_step"`
	CapMinutes          int       `bson:"cap_minutes" json:"cap_minutes"`
	Date                string    `bson:"date" json:"date"`
	EnabledRoles        []string  `bson:"enabled_roles" json:"enabled_roles"`
	EnabledTasks        []string  `bson:"enabled_tasks" json:"enabled_tasks"`
	ExcludedTasks       []string  `bson:"excluded_tasks" json:"excluded_tasks"`
	DefaultRestFallback int       `bson:"default_rest_fallback" json:"default_rest_fallback"`
	UpdatedAt           time.Time `bson:"updated_at" json:"updated_at"`
}

// RampConfig represents the configuration settings for the warm-up ramp.
type RampConfig struct {
	CapMinutes          int      `bson:"cap_minutes" json:"cap_minutes"`
	EnabledRoles        []string `bson:"enabled_roles" json:"enabled_roles"`
	EnabledTasks        []string `bson:"enabled_tasks" json:"enabled_tasks"`
	ExcludedTasks       []string `bson:"excluded_tasks" json:"excluded_tasks"`
	DefaultRestFallback int      `bson:"default_rest_fallback" json:"default_rest_fallback"`
}

// RampStatus represents the current status and settings of the warm-up ramp.
type RampStatus struct {
	CurrentStep       int        `json:"current_step"`
	CapMinutes        int        `json:"cap_minutes"`
	IsCapped          bool       `json:"is_capped"`
	TodayFocusMinutes int        `json:"today_focus_minutes"`
	Date              string     `json:"date"`
	Config            RampConfig `json:"config"`
}

// DefaultRampInfo returns a RampInfo with default settings.
func DefaultRampInfo() RampInfo {
	return RampInfo{
		Title:               "Ramp Info",
		CurrentStep:         1,
		CapMinutes:          25,
		Date:                time.Now().Format("2 January 2006"),
		EnabledRoles:        []string{"work", "learn"},
		EnabledTasks:        []string{"home_task"},
		ExcludedTasks:       []string{"video", "movies", "games", "telegram"},
		DefaultRestFallback: 15,
		UpdatedAt:           time.Now().UTC(),
	}
}

// ToConfig converts RampInfo to RampConfig.
func (r RampInfo) ToConfig() RampConfig {
	roles := r.EnabledRoles
	if roles == nil {
		roles = []string{}
	}
	tasks := r.EnabledTasks
	if tasks == nil {
		tasks = []string{}
	}
	excluded := r.ExcludedTasks
	if excluded == nil {
		excluded = []string{}
	}
	return RampConfig{
		CapMinutes:          r.CapMinutes,
		EnabledRoles:        roles,
		EnabledTasks:        tasks,
		ExcludedTasks:       excluded,
		DefaultRestFallback: r.DefaultRestFallback,
	}
}

// ToStatus converts RampInfo to RampStatus given today's focus minutes.
func (r RampInfo) ToStatus(todayFocusMinutes int) RampStatus {
	return RampStatus{
		CurrentStep:       r.CurrentStep,
		CapMinutes:        r.CapMinutes,
		IsCapped:          r.CurrentStep >= r.CapMinutes,
		TodayFocusMinutes: todayFocusMinutes,
		Date:              r.Date,
		Config:            r.ToConfig(),
	}
}
