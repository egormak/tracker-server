package services

import (
	"log/slog"
	"math"
	"strings"
	"time"
	"tracker-server/internal/domain/entity"
)

func clampShare(share int) int {
	if share < 0 {
		return 0
	}
	if share > 100 {
		return 100
	}
	return share
}

// splitSession decides where a regular (non-backfill) session's minutes go:
//  1. the task's DebtShare % of them pays its own previous-days debt, oldest day first,
//     each day at most its shortfall; what the debt cannot take returns;
//  2. the rest goes to today up to today's target;
//  3. the excess beyond the target pays the rest of the task's debt, oldest day first;
//  4. what is left goes to tomorrow for a strict task, to today for the others.
//
// Without a target (no schedule entry, no params time) everything that is not the share
// stays today, as before.
func (s *RunningTaskService) splitSession(task entity.RunningTask, duration int) []entity.SplitPart {
	var debt []entity.DebtItem
	if s.debt != nil {
		items, err := s.debt.TaskDebt(task.TaskName)
		if err != nil {
			slog.Warn("RunningTaskService: failed to read task debt", "task", task.TaskName, "error", err)
		} else {
			debt = items
		}
	}

	paid := make([]int, len(debt))
	payDebt := func(budget int) int {
		used := 0
		for i, item := range debt {
			if used >= budget {
				break
			}
			take := min(item.Remaining-paid[i], budget-used)
			if take > 0 {
				paid[i] += take
				used += take
			}
		}
		return used
	}

	shareMin := int(math.Round(float64(duration*clampShare(task.DebtShare)) / 100))
	todayPart := duration - payDebt(shareMin)

	toToday, excess := todayPart, 0
	if target := GetScheduledTargetTime(s.st, task.TaskName); target > 0 {
		todayDone, _ := s.st.GetTodayTaskDuration(task.TaskName)
		allowed := max(target-todayDone, 0)
		if todayPart > allowed {
			toToday, excess = allowed, todayPart-allowed
		}
	}
	excess -= payDebt(excess)

	tomorrow := 0
	if excess > 0 {
		if strict, _ := s.st.IsTaskStrict(task.TaskName); strict {
			tomorrow = excess
		} else {
			toToday += excess
		}
	}

	var parts []entity.SplitPart
	if toToday > 0 {
		parts = append(parts, entity.SplitPart{
			Kind:    entity.SplitToday,
			Day:     strings.ToLower(time.Now().Weekday().String()),
			Date:    time.Now().Format("2 January 2006"),
			Minutes: toToday,
		})
	}
	for i, item := range debt {
		if paid[i] > 0 {
			parts = append(parts, entity.SplitPart{Kind: entity.SplitDebt, Day: item.Day, Date: item.Date, Minutes: paid[i]})
		}
	}
	if tomorrow > 0 {
		parts = append(parts, entity.SplitPart{
			Kind:    entity.SplitTomorrow,
			Day:     GetTomorrowDayName(),
			Date:    CalculateDateForTomorrow(),
			Minutes: tomorrow,
		})
	}
	return parts
}

var (
	dayShortRuNames = map[string]string{"monday": "Пн", "tuesday": "Вт", "wednesday": "Ср", "thursday": "Чт", "friday": "Пт", "saturday": "Сб", "sunday": "Вс"}
	dayFullRuNames  = map[string]string{"monday": "Понедельник", "tuesday": "Вторник", "wednesday": "Среда", "thursday": "Четверг", "friday": "Пятница", "saturday": "Суббота", "sunday": "Воскресенье"}
)

func dayShortRu(day string) string {
	if n, ok := dayShortRuNames[strings.ToLower(day)]; ok {
		return n
	}
	return day
}

func dayFullRu(day string) string {
	if n, ok := dayFullRuNames[strings.ToLower(day)]; ok {
		return n
	}
	return day
}
