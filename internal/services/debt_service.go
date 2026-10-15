package services

import (
	"fmt"
	"strings"
	"time"
	"tracker-server/internal/domain/entity"
)

var weekDayNames = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

// DebtService reads the debt (the previous days' unfinished schedule of this week) from the
// rollover computation of ScheduleService, which already walks Monday up to today.
type DebtService struct {
	sched *ScheduleService
}

func NewDebtService(sched *ScheduleService) *DebtService {
	return &DebtService{sched: sched}
}

func todayName() string {
	return strings.ToLower(time.Now().Weekday().String())
}

// rows returns the rollover rows from Monday up to and including today.
func (s *DebtService) rows() ([]entity.RolloverTask, error) {
	return s.sched.GetRolloverTasks(todayName())
}

// Summary returns the debt items, today's remainder and the week strip.
func (s *DebtService) Summary() (entity.DebtSummary, error) {
	schedule, err := s.sched.st.GetActiveSchedule()
	if err != nil {
		return entity.DebtSummary{}, fmt.Errorf("no active schedule: %w", err)
	}
	rows, err := s.rows()
	if err != nil {
		return entity.DebtSummary{}, err
	}

	today := todayName()
	todayOrd := dayOrder[today]
	now := time.Now()

	summary := entity.DebtSummary{Items: []entity.DebtItem{}}
	remainingByDay := make(map[string]int)
	for _, r := range rows {
		day := strings.ToLower(r.SourceDay)
		remainingByDay[day] += r.RemainingTime
		if day == today {
			summary.TodayRemaining += r.RemainingTime
			continue
		}
		summary.Items = append(summary.Items, entity.DebtItem{
			TaskName:  r.TaskName,
			Role:      r.Role,
			Day:       day,
			Date:      s.sched.getDateForDay(day, now),
			Remaining: r.RemainingTime,
		})
		summary.TotalMinutes += r.RemainingTime
	}
	if len(summary.Items) > 0 {
		summary.OldestDay = summary.Items[0].Day
	}

	for _, day := range weekDayNames {
		planned := 0
		for _, t := range s.sched.getDayScheduleFromWeekly(schedule, day).Tasks {
			planned += t.Time
		}
		ord := dayOrder[day]
		date := now.AddDate(0, 0, ord-todayOrd).Format("2 January 2006")
		cell := entity.DebtDay{Day: day, Date: date, Planned: planned}
		switch {
		case planned == 0:
			cell.Status = entity.DebtDayNone
		case ord > todayOrd:
			cell.Status = entity.DebtDayFuture
		case ord == todayOrd:
			cell.Remaining = remainingByDay[day]
			cell.Status = entity.DebtDayToday
		default:
			cell.Remaining = remainingByDay[day]
			cell.Status = debtDayStatus(planned, cell.Remaining)
		}
		summary.Days = append(summary.Days, cell)
	}
	return summary, nil
}

func debtDayStatus(planned, remaining int) string {
	switch {
	case remaining <= 0:
		return entity.DebtDayClosed
	case remaining >= planned:
		return entity.DebtDayOpen
	default:
		return entity.DebtDayPartial
	}
}

// TaskDebt returns one task's debt on the previous days, oldest day first.
func (s *DebtService) TaskDebt(taskName string) ([]entity.DebtItem, error) {
	rows, err := s.rows()
	if err != nil {
		return nil, err
	}
	today := todayName()
	now := time.Now()
	var items []entity.DebtItem
	for _, r := range rows {
		day := strings.ToLower(r.SourceDay)
		if day == today || !strings.EqualFold(r.TaskName, taskName) {
			continue
		}
		items = append(items, entity.DebtItem{
			TaskName:  r.TaskName,
			Role:      r.Role,
			Day:       day,
			Date:      s.sched.getDateForDay(day, now),
			Remaining: r.RemainingTime,
		})
	}
	return items, nil // rollover rows are already ordered oldest day first
}

// DayClosed reports whether a previous day has a schedule and none of it is left.
func (s *DebtService) DayClosed(day string) (bool, error) {
	day = strings.ToLower(day)
	schedule, err := s.sched.st.GetActiveSchedule()
	if err != nil {
		return false, err
	}
	if len(s.sched.getDayScheduleFromWeekly(schedule, day).Tasks) == 0 {
		return false, nil
	}
	rows, err := s.rows()
	if err != nil {
		return false, err
	}
	for _, r := range rows {
		if strings.EqualFold(r.SourceDay, day) && r.RemainingTime > 0 {
			return false, nil
		}
	}
	return true, nil
}

// DebtToDate returns, per lowercased task name, the previous days' debt plus today's remainder.
func (s *DebtService) DebtToDate() (map[string]int, error) {
	rows, err := s.rows()
	if err != nil {
		return nil, err
	}
	out := make(map[string]int)
	for _, r := range rows {
		out[strings.ToLower(r.TaskName)] += r.RemainingTime
	}
	return out, nil
}
