package services

import (
	"log/slog"
	"time"
	"tracker-server/internal/domain/entity"
)

// Usage events. The start_* events partition every newly started running task by the
// way it was chosen.
const (
	UsageStartManual       = "start_manual"
	UsageStartBreakVariety = "start_break_variety"
	UsageStartBreakMood    = "start_break_mood"
	UsageStartEvening      = "start_evening"
	UsageStartCombo        = "start_combo"
	UsageStartBatch        = "start_batch"
	UsageEveningSkip       = "evening_skip"
	UsageRampReset         = "ramp_reset"
	UsageTimerExtend       = "timer_extend"
)

// Start origins a client may send with POST /timer/run/start.
const (
	OriginBreak   = "break"
	OriginEvening = "evening"
	OriginCombo   = "combo"
	OriginBatch   = "batch"
)

// ClientUnknown stands for requests and records with no client name.
const ClientUnknown = "unknown"

type UsageStorage interface {
	IncUsageCounter(week, event, client string) error
	GetUsageCounters(week string) ([]entity.UsageCounter, error)
}

// UsageService counts user choices per calendar week, for the weekly digest only.
type UsageService struct {
	st      UsageStorage
	nowFunc func() time.Time
}

func NewUsageService(st UsageStorage) *UsageService {
	return &UsageService{st: st, nowFunc: time.Now}
}

// weekStartDate returns the Monday of now's week shifted by weekOffset weeks.
func weekStartDate(now time.Time, weekOffset int) string {
	now = now.AddDate(0, 0, 7*weekOffset)
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6
	}
	return now.AddDate(0, 0, -daysSinceMonday).Format("2 January 2006")
}

// Track counts one event for the current week. It never fails the caller: a lost
// count is not worth a failed request.
func (s *UsageService) Track(event, client string) {
	if s == nil || event == "" {
		return
	}
	if client == "" {
		client = ClientUnknown
	}
	if err := s.st.IncUsageCounter(weekStartDate(s.nowFunc(), 0), event, client); err != nil {
		slog.Warn("UsageService: failed to count event", "event", event, "client", client, "err", err)
	}
}
