package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"
	"tracker-server/internal/domain/entity"
)

// RampStorage defines the storage interface required by RampService.
type RampStorage interface {
	GetRampInfo(ctx context.Context) (entity.RampInfo, error)
	SaveRampInfo(ctx context.Context, ramp entity.RampInfo) error
	ResetRampStep(ctx context.Context, date string, baselineMinutes int) error
	UpdateRampBaseline(ctx context.Context, baselineMinutes int) error
	UpdateRampStep(ctx context.Context, step int, date string) error
	GetRole(taskName string) (string, error)
	GetTodayTaskDurationsMap(date string) (map[string]int, error)
}

// RampService manages linear warm-up ramp progression, configuration, and eligibility.
type RampService struct {
	st RampStorage
	// mu serializes the read-modify-write sequences on the Ramp Info document.
	mu sync.Mutex
}

// NewRampService creates a new RampService instance.
func NewRampService(st RampStorage) *RampService {
	return &RampService{st: st}
}

// CalculateStep computes the ramp step for a given amount of focus minutes S:
// k = min(cap, floor((1 + sqrt(1 + 8S)) / 2)).
func CalculateStep(focusMinutes int, capMinutes int) int {
	if focusMinutes <= 0 {
		return 1
	}
	k := int(math.Floor((1.0 + math.Sqrt(1.0+8.0*float64(focusMinutes))) / 2.0))
	if k < 1 {
		k = 1
	}
	if capMinutes > 0 && k > capMinutes {
		k = capMinutes
	}
	return k
}

// CalculateStep computes the ramp step for a given amount of focus minutes.
func (s *RampService) CalculateStep(focusMinutes int, capMinutes int) int {
	return CalculateStep(focusMinutes, capMinutes)
}

// getOrResetRamp loads RampInfo and applies lazy midnight auto-reset if needed.
func (s *RampService) getOrResetRamp(ctx context.Context) (entity.RampInfo, error) {
	today := time.Now().Format("2 January 2006")
	ramp, err := s.st.GetRampInfo(ctx)
	if err != nil {
		return entity.RampInfo{}, fmt.Errorf("failed to get ramp info: %w", err)
	}

	if ramp.Date != today {
		slog.Info("Ramp lazy midnight reset triggered",
			"stored_date", ramp.Date,
			"today", today,
			"prev_step", ramp.CurrentStep,
		)
		if err := s.st.ResetRampStep(ctx, today, 0); err != nil {
			return entity.RampInfo{}, fmt.Errorf("failed to reset ramp step on midnight: %w", err)
		}
		ramp.CurrentStep = 1
		ramp.Date = today
		ramp.ResetBaselineMinutes = 0
	}

	return ramp, nil
}

// IsTaskEligible checks whether a task or its role participates in the warm-up ramp.
func (s *RampService) IsTaskEligible(taskName string, ramp entity.RampInfo) bool {
	if taskName == "" {
		return false
	}

	// 1. Excluded tasks check (highest precedence)
	for _, excluded := range ramp.ExcludedTasks {
		if strings.EqualFold(taskName, excluded) {
			return false
		}
	}

	// 2. Direct enabled tasks check
	for _, enabled := range ramp.EnabledTasks {
		if strings.EqualFold(taskName, enabled) {
			return true
		}
	}

	// 3. Query role from storage
	role, err := s.st.GetRole(taskName)
	if err == nil && role != "" {
		for _, excluded := range ramp.ExcludedTasks {
			if strings.EqualFold(role, excluded) {
				return false
			}
		}
		for _, enabledRole := range ramp.EnabledRoles {
			if strings.EqualFold(role, enabledRole) {
				return true
			}
		}
	}

	// 4. Check if taskName itself matches an enabled role (e.g. task named "work" or "learn")
	for _, enabledRole := range ramp.EnabledRoles {
		if strings.EqualFold(taskName, enabledRole) {
			return true
		}
	}

	return false
}

// getTodayFocusMinutes calculates sum of today's focus minutes for eligible tasks.
func (s *RampService) getTodayFocusMinutes(ctx context.Context, ramp entity.RampInfo, today string) (int, error) {
	durationsMap, err := s.st.GetTodayTaskDurationsMap(today)
	if err != nil {
		return 0, fmt.Errorf("failed to get task durations map: %w", err)
	}

	total := 0
	for tName, dur := range durationsMap {
		if dur > 0 && s.IsTaskEligible(tName, ramp) {
			total += dur
		}
	}
	return total, nil
}

// effectiveFocus returns the focus minutes counted since the last manual reset.
// If today's focus dropped below the baseline (a record was removed or became ineligible),
// the baseline is lowered to match so later focus minutes keep advancing the ramp.
func (s *RampService) effectiveFocus(ctx context.Context, ramp *entity.RampInfo, focusMin int) int {
	if focusMin < ramp.ResetBaselineMinutes {
		if err := s.st.UpdateRampBaseline(ctx, focusMin); err != nil {
			slog.Error("failed to lower ramp reset baseline in storage", "err", err)
		}
		ramp.ResetBaselineMinutes = focusMin
	}
	return ramp.EffectiveFocusMinutes(focusMin)
}

// GetStatus returns the current RampStatus with lazy midnight auto-reset applied.
func (s *RampService) GetStatus(ctx context.Context) (entity.RampStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return entity.RampStatus{}, err
	}

	today := time.Now().Format("2 January 2006")
	focusMin, err := s.getTodayFocusMinutes(ctx, ramp, today)
	if err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to get focus minutes: %w", err)
	}

	effectiveFocus := s.effectiveFocus(ctx, &ramp, focusMin)

	expectedStep := s.CalculateStep(effectiveFocus, ramp.CapMinutes)
	if expectedStep > ramp.CurrentStep {
		if err := s.st.UpdateRampStep(ctx, expectedStep, today); err != nil {
			slog.Error("GetStatus: failed to reconcile ramp step in storage", "err", err)
		}
		ramp.CurrentStep = expectedStep
	}

	return ramp.ToStatus(focusMin), nil
}

// GetDurationForTask checks eligibility and returns current_step or default_fallback_minutes.
func (s *RampService) GetDurationForTask(ctx context.Context, taskName string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return 0, err
	}

	if s.IsTaskEligible(taskName, ramp) {
		step := ramp.CurrentStep
		if step < 1 {
			step = 1
		}
		if step > ramp.CapMinutes {
			step = ramp.CapMinutes
		}
		return step, nil
	}

	fallback := ramp.DefaultFallbackMinutes
	if fallback < 1 {
		fallback = 15
	}
	return fallback, nil
}

// Reset manually resets the current step to 1.
func (s *RampService) Reset(ctx context.Context) (entity.RampStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return entity.RampStatus{}, err
	}

	today := time.Now().Format("2 January 2006")
	focusMin, err := s.getTodayFocusMinutes(ctx, ramp, today)
	if err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to get focus minutes: %w", err)
	}

	if err := s.st.ResetRampStep(ctx, today, focusMin); err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to reset ramp step: %w", err)
	}
	ramp.CurrentStep = 1
	ramp.Date = today
	ramp.ResetBaselineMinutes = focusMin

	slog.Info("Ramp manual reset performed", "date", today, "current_step", ramp.CurrentStep, "reset_baseline_minutes", ramp.ResetBaselineMinutes)
	return ramp.ToStatus(focusMin), nil
}

// Advance manually increments the step by 1, clamped to CapMinutes.
func (s *RampService) Advance(ctx context.Context) (entity.RampStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return entity.RampStatus{}, err
	}

	today := time.Now().Format("2 January 2006")
	newStep := ramp.CurrentStep + 1
	if newStep > ramp.CapMinutes {
		newStep = ramp.CapMinutes
	}

	if err := s.st.UpdateRampStep(ctx, newStep, today); err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to update ramp step: %w", err)
	}

	ramp.CurrentStep = newStep
	focusMin, err := s.getTodayFocusMinutes(ctx, ramp, today)
	if err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to get focus minutes: %w", err)
	}

	slog.Info("Ramp manual advance performed", "new_step", newStep, "cap", ramp.CapMinutes)
	return ramp.ToStatus(focusMin), nil
}

// GetConfig returns the current ramp configuration.
func (s *RampService) GetConfig(ctx context.Context) (entity.RampConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return entity.RampConfig{}, err
	}
	return ramp.ToConfig(), nil
}

// UpdateConfig validates and saves the updated configuration settings.
func (s *RampService) UpdateConfig(ctx context.Context, cfg entity.RampConfig) (entity.RampStatus, error) {
	if cfg.CapMinutes < 5 {
		return entity.RampStatus{}, fmt.Errorf("cap_minutes must be at least 5")
	}
	if cfg.DefaultFallbackMinutes < 1 {
		return entity.RampStatus{}, fmt.Errorf("default_fallback_minutes must be at least 1")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return entity.RampStatus{}, err
	}

	today := time.Now().Format("2 January 2006")
	focusBefore := 0
	if ramp.ResetBaselineMinutes > 0 {
		focusBefore, err = s.getTodayFocusMinutes(ctx, ramp, today)
		if err != nil {
			return entity.RampStatus{}, fmt.Errorf("failed to get focus minutes: %w", err)
		}
	}
	hadBaseline := ramp.ResetBaselineMinutes > 0

	ramp.CapMinutes = cfg.CapMinutes
	if cfg.EnabledRoles != nil {
		ramp.EnabledRoles = cfg.EnabledRoles
	}
	if cfg.EnabledTasks != nil {
		ramp.EnabledTasks = cfg.EnabledTasks
	}
	if cfg.ExcludedTasks != nil {
		ramp.ExcludedTasks = cfg.ExcludedTasks
	}
	ramp.DefaultFallbackMinutes = cfg.DefaultFallbackMinutes

	if ramp.CurrentStep > ramp.CapMinutes {
		ramp.CurrentStep = ramp.CapMinutes
	}

	focusMin, err := s.getTodayFocusMinutes(ctx, ramp, today)
	if err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to get focus minutes: %w", err)
	}

	// Eligibility may have changed: shift the manual-reset baseline by the same amount
	// so focus counted since the reset stays the same.
	if hadBaseline {
		baseline := ramp.ResetBaselineMinutes + focusMin - focusBefore
		if baseline < 0 {
			baseline = 0
		}
		if baseline > focusMin {
			baseline = focusMin
		}
		ramp.ResetBaselineMinutes = baseline
	}

	if err := s.st.SaveRampInfo(ctx, ramp); err != nil {
		return entity.RampStatus{}, fmt.Errorf("failed to save ramp info: %w", err)
	}

	slog.Info("Ramp configuration updated",
		"cap_minutes", ramp.CapMinutes,
		"default_fallback_minutes", ramp.DefaultFallbackMinutes,
	)
	return ramp.ToStatus(focusMin), nil
}

// AutoRecalculateOnRecord calculates progression step k from today's focus minutes
// and advances current_step if k > current_step.
func (s *RampService) AutoRecalculateOnRecord(ctx context.Context, taskName string, duration int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ramp, err := s.getOrResetRamp(ctx)
	if err != nil {
		return fmt.Errorf("failed to get ramp info for auto-recalculate: %w", err)
	}

	if !s.IsTaskEligible(taskName, ramp) {
		slog.Debug("Task not eligible for ramp recalculation", "task", taskName)
		return nil
	}

	today := time.Now().Format("2 January 2006")
	focusMin, err := s.getTodayFocusMinutes(ctx, ramp, today)
	if err != nil {
		return fmt.Errorf("failed to get today focus minutes for recalculate: %w", err)
	}

	effectiveFocus := s.effectiveFocus(ctx, &ramp, focusMin)

	k := CalculateStep(effectiveFocus, ramp.CapMinutes)
	if k > ramp.CurrentStep {
		slog.Info("Auto-advancing ramp step on record",
			"prev_step", ramp.CurrentStep,
			"new_step", k,
			"today_focus_minutes", focusMin,
			"effective_focus_minutes", effectiveFocus,
			"task", taskName,
		)
		if err := s.st.UpdateRampStep(ctx, k, today); err != nil {
			return fmt.Errorf("failed to update ramp step on auto-recalculate: %w", err)
		}
	}

	return nil
}
