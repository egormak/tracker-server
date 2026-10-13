package services

import (
	"fmt"
	"sort"
	"strings"
	"tracker-server/internal/domain/entity"
)

const (
	WeekCurrent  = "current"
	WeekPrevious = "previous"
)

// ErrInvalidWeek is returned for a week selector other than "current" or "previous".
var ErrInvalidWeek = fmt.Errorf("week must be %q or %q", WeekCurrent, WeekPrevious)

func percentOf(part, total int) int {
	if total <= 0 {
		return 0
	}
	return part * 100 / total
}

// roleRank keeps the standard roles first and in a fixed order.
func roleRank(role string) int {
	switch role {
	case "work":
		return 0
	case "learn":
		return 1
	case "rest":
		return 2
	}
	return 3
}

// GetWeeklyDigest builds the end-of-week debrief. "previous" exists because the weekly
// wipe happens on Monday, when the finished week is no longer the current one.
func (s *StatisticService) GetWeeklyDigest(week string) (entity.WeeklyDigest, error) {
	offset := 0
	switch week {
	case "", WeekCurrent:
		week = WeekCurrent
	case WeekPrevious:
		offset = -1
	default:
		return entity.WeeklyDigest{}, ErrInvalidWeek
	}

	stats, err := s.weeklyStats(offset)
	if err != nil {
		return entity.WeeklyDigest{}, err
	}

	digest := entity.WeeklyDigest{
		Week:          week,
		WeekStartDate: stats.WeekStartDate,
		WeekEndDate:   stats.WeekEndDate,
		Roles:         []entity.WeeklyDigestRole{},
		Strict:        []entity.WeeklyDigestTask{},
		Free:          []entity.WeeklyDigestTask{},
		OverPlan:      []entity.WeeklyDigestDelta{},
		Untouched:     []entity.WeeklyDigestDelta{},
		Unscheduled:   []entity.WeeklyDigestDelta{},
		Days:          make([]entity.WeeklyDigestDay, 0, len(stats.Days)),
	}

	for _, rt := range stats.WeeklyTargets {
		digest.TotalPlanned += rt.TimeDuration
		digest.TotalDone += rt.TimeDone
	}
	for _, rt := range stats.WeeklyTargets {
		if rt.TimeDuration == 0 && rt.TimeDone == 0 {
			continue
		}
		digest.Roles = append(digest.Roles, entity.WeeklyDigestRole{
			Role:         rt.Role,
			Planned:      rt.TimeDuration,
			Done:         rt.TimeDone,
			SharePercent: percentOf(rt.TimeDone, digest.TotalDone),
		})
	}
	sort.Slice(digest.Roles, func(i, j int) bool {
		ri, rj := roleRank(digest.Roles[i].Role), roleRank(digest.Roles[j].Role)
		if ri != rj {
			return ri < rj
		}
		return digest.Roles[i].Role < digest.Roles[j].Role
	})

	strictDone := 0
	for _, tt := range stats.WeeklyTasks {
		task := entity.WeeklyDigestTask{
			Name:        tt.Name,
			Role:        tt.Role,
			Planned:     tt.TimeDuration,
			Done:        tt.TimeDone,
			PlanPercent: percentOf(tt.TimeDone, tt.TimeDuration),
		}
		// Same strict set as the evening and break selectors.
		if lower := strings.ToLower(tt.Name); lower == "work" || lower == "english" {
			digest.Strict = append(digest.Strict, task)
			strictDone += tt.TimeDone
		} else {
			digest.Free = append(digest.Free, task)
			digest.FreeDone += tt.TimeDone
		}

		delta := entity.WeeklyDigestDelta{
			Name:    tt.Name,
			Role:    tt.Role,
			Planned: tt.TimeDuration,
			Done:    tt.TimeDone,
			Delta:   tt.TimeDone - tt.TimeDuration,
		}
		switch {
		case tt.TimeDuration == 0 && tt.TimeDone > 0:
			digest.Unscheduled = append(digest.Unscheduled, delta)
		case tt.TimeDuration > 0 && tt.TimeDone == 0:
			digest.Untouched = append(digest.Untouched, delta)
		case tt.TimeDuration > 0 && tt.TimeDone > tt.TimeDuration:
			digest.OverPlan = append(digest.OverPlan, delta)
		}
	}
	for i := range digest.Strict {
		digest.Strict[i].SharePercent = percentOf(digest.Strict[i].Done, strictDone)
	}
	for i := range digest.Free {
		digest.Free[i].SharePercent = percentOf(digest.Free[i].Done, digest.FreeDone)
	}

	sort.Slice(digest.Strict, func(i, j int) bool {
		ri, rj := roleRank(digest.Strict[i].Role), roleRank(digest.Strict[j].Role)
		if ri != rj {
			return ri < rj
		}
		return digest.Strict[i].Name < digest.Strict[j].Name
	})
	sort.Slice(digest.Free, func(i, j int) bool {
		if digest.Free[i].Done != digest.Free[j].Done {
			return digest.Free[i].Done > digest.Free[j].Done
		}
		return digest.Free[i].Name < digest.Free[j].Name
	})
	sortDeltas := func(list []entity.WeeklyDigestDelta, key func(entity.WeeklyDigestDelta) int) {
		sort.Slice(list, func(i, j int) bool {
			if key(list[i]) != key(list[j]) {
				return key(list[i]) > key(list[j])
			}
			return list[i].Name < list[j].Name
		})
	}
	sortDeltas(digest.OverPlan, func(d entity.WeeklyDigestDelta) int { return d.Delta })
	sortDeltas(digest.Untouched, func(d entity.WeeklyDigestDelta) int { return d.Planned })
	sortDeltas(digest.Unscheduled, func(d entity.WeeklyDigestDelta) int { return d.Done })

	for _, day := range stats.Days {
		d := entity.WeeklyDigestDay{Day: day.Day, Date: day.Date, TotalDone: day.TotalDone}
		digest.Days = append(digest.Days, d)
		if d.TotalDone > 0 && (digest.BusiestDay == nil || d.TotalDone > digest.BusiestDay.TotalDone) {
			busiest := d
			digest.BusiestDay = &busiest
		}
	}

	return digest, nil
}
