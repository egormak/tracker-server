package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	DefaultDayRollInterval = 30 * time.Second
)

// DayRollService provides background day-roll monitoring to silently apply
// the active schedule when calendar midnight transitions occur.
type DayRollService struct {
	scheduleService *ScheduleService
	interval        time.Duration
	lastDate        string
	mu              sync.Mutex
	stopCh          chan struct{}
	nowFunc         func() time.Time
}

// NewDayRollService creates a new DayRollService instance.
func NewDayRollService(scheduleService *ScheduleService) *DayRollService {
	return &DayRollService{
		scheduleService: scheduleService,
		interval:        DefaultDayRollInterval,
		stopCh:          make(chan struct{}),
		nowFunc:         time.Now,
	}
}

// SetInterval overrides the check interval (useful for testing).
func (d *DayRollService) SetInterval(interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.interval = interval
}

// SetNowFunc overrides the time provider (useful for testing).
func (d *DayRollService) SetNowFunc(f func() time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nowFunc = f
}

func (d *DayRollService) now() time.Time {
	if d.nowFunc != nil {
		return d.nowFunc()
	}
	return time.Now()
}

// Start begins background day-roll checking and ensures today's schedule on startup.
func (d *DayRollService) Start(ctx context.Context) {
	d.mu.Lock()
	currentDate := d.now().Format("2 January 2006")
	d.lastDate = currentDate
	interval := d.interval
	d.mu.Unlock()

	slog.Info("DayRollService: starting background worker", "initial_date", currentDate, "interval", interval)

	// Startup: ensure today's schedule is applied if database was empty or after tracker clean
	if err := d.scheduleService.EnsureTodaySchedule(); err != nil {
		slog.Warn("DayRollService: failed to ensure today schedule on startup", "error", err)
	} else {
		slog.Info("DayRollService: today schedule ensured on startup", "date", currentDate)
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("DayRollService: context cancelled, stopping day-roll worker")
				return
			case <-d.stopCh:
				slog.Info("DayRollService: received stop signal, stopping day-roll worker")
				return
			case <-ticker.C:
				if err := d.CheckDayRoll(); err != nil {
					slog.Error("DayRollService: error during day-roll check", "error", err)
				}
			}
		}
	}()
}

// Stop cleanly terminates the background worker loop.
func (d *DayRollService) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	select {
	case <-d.stopCh:
	default:
		close(d.stopCh)
	}
}

// CheckDayRoll checks if the calendar day has rolled over. If so, it calls ApplyScheduleToday() and updates the tracked date upon success.
func (d *DayRollService) CheckDayRoll() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	currentDate := d.now().Format("2 January 2006")
	if currentDate == d.lastDate {
		return nil
	}

	slog.Info("DayRollService: calendar day rollover detected", "previous_date", d.lastDate, "new_date", currentDate)

	if err := d.scheduleService.ApplyScheduleToday(); err != nil {
		slog.Error("DayRollService: failed to apply schedule on day rollover", "date", currentDate, "error", err)
		return err
	}

	d.lastDate = currentDate
	slog.Info("DayRollService: successfully applied schedule for new day", "date", currentDate)
	return nil
}

// GetLastDate returns the currently tracked date (thread-safe).
func (d *DayRollService) GetLastDate() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastDate
}
