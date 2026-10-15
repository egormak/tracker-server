package entity

import "time"

// RampInfo represents the warm-up ramp document stored in MongoDB task_info collection.
type RampInfo struct {
	Title                  string   `bson:"title" json:"title"`
	CurrentStep            int      `bson:"current_step" json:"current_step"`
	CapMinutes             int      `bson:"cap_minutes" json:"cap_minutes"`
	Date                   string   `bson:"date" json:"date"`
	EnabledRoles           []string `bson:"enabled_roles" json:"enabled_roles"`
	EnabledTasks           []string `bson:"enabled_tasks" json:"enabled_tasks"`
	ExcludedTasks          []string `bson:"excluded_tasks" json:"excluded_tasks"`
	DefaultFallbackMinutes int      `bson:"default_fallback_minutes" json:"default_fallback_minutes"`
	// LegacyRestFallback is the pre-2026-10 name of DefaultFallbackMinutes, read once and migrated.
	LegacyRestFallback   int       `bson:"default_rest_fallback,omitempty" json:"-"`
	ResetBaselineMinutes int       `bson:"reset_baseline_minutes" json:"reset_baseline_minutes"`
	UpdatedAt            time.Time `bson:"updated_at" json:"updated_at"`
}

// RampConfig represents the configuration settings for the warm-up ramp.
type RampConfig struct {
	CapMinutes             int      `bson:"cap_minutes" json:"cap_minutes"`
	EnabledRoles           []string `bson:"enabled_roles" json:"enabled_roles"`
	EnabledTasks           []string `bson:"enabled_tasks" json:"enabled_tasks"`
	ExcludedTasks          []string `bson:"excluded_tasks" json:"excluded_tasks"`
	DefaultFallbackMinutes int      `bson:"default_fallback_minutes" json:"default_fallback_minutes"`
	// LegacyRestFallback accepts the old JSON key from clients not yet rebuilt.
	LegacyRestFallback int `bson:"-" json:"default_rest_fallback,omitempty"`
}

// RampStatus represents the current status and settings of the warm-up ramp.
type RampStatus struct {
	CurrentStep           int        `json:"current_step"`
	CapMinutes            int        `json:"cap_minutes"`
	IsCapped              bool       `json:"is_capped"`
	TodayFocusMinutes     int        `json:"today_focus_minutes"`
	ResetBaselineMinutes  int        `json:"reset_baseline_minutes"`
	EffectiveFocusMinutes int        `json:"effective_focus_minutes"`
	Date                  string     `json:"date"`
	Config                RampConfig `json:"config"`
}

// DefaultRampInfo returns a RampInfo with default settings.
func DefaultRampInfo() RampInfo {
	return RampInfo{
		Title:                  "Ramp Info",
		CurrentStep:            1,
		CapMinutes:             25,
		Date:                   time.Now().Format("2 January 2006"),
		EnabledRoles:           []string{"work", "learn"},
		EnabledTasks:           []string{"home_task"},
		ExcludedTasks:          []string{"video", "movies", "games", "telegram"},
		DefaultFallbackMinutes: 15,
		ResetBaselineMinutes:   0,
		UpdatedAt:              time.Now().UTC(),
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
		CapMinutes:             r.CapMinutes,
		EnabledRoles:           roles,
		EnabledTasks:           tasks,
		ExcludedTasks:          excluded,
		DefaultFallbackMinutes: r.DefaultFallbackMinutes,
	}
}

// EffectiveFocusMinutes returns today's focus minutes counted since the last manual reset:
// todayFocusMinutes minus the reset baseline, never negative.
func (r RampInfo) EffectiveFocusMinutes(todayFocusMinutes int) int {
	effective := todayFocusMinutes - r.ResetBaselineMinutes
	if effective < 0 {
		return 0
	}
	return effective
}

// ToStatus converts RampInfo to RampStatus given today's focus minutes.
func (r RampInfo) ToStatus(todayFocusMinutes int) RampStatus {
	return RampStatus{
		CurrentStep:           r.CurrentStep,
		CapMinutes:            r.CapMinutes,
		IsCapped:              r.CurrentStep >= r.CapMinutes,
		TodayFocusMinutes:     todayFocusMinutes,
		ResetBaselineMinutes:  r.ResetBaselineMinutes,
		EffectiveFocusMinutes: r.EffectiveFocusMinutes(todayFocusMinutes),
		Date:                  r.Date,
		Config:                r.ToConfig(),
	}
}
