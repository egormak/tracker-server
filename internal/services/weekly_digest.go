package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
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

// usageEventOrder is the order usage events appear in the digest.
var usageEventOrder = []string{
	UsageStartManual,
	UsageStartBreakVariety,
	UsageStartBreakMood,
	UsageStartEvening,
	UsageStartCombo,
	UsageEveningSkip,
	UsageRampReset,
	UsageTimerExtend,
}

// digestClients groups the week's records by the client that logged them.
func digestClients(records []entity.TaskRecord) []entity.WeeklyDigestClient {
	byClient := make(map[string]*entity.WeeklyDigestClient)
	total := 0
	for _, rec := range records {
		name := rec.SourceDevice
		if name == "" {
			name = ClientUnknown
		}
		c, ok := byClient[name]
		if !ok {
			c = &entity.WeeklyDigestClient{Client: name}
			byClient[name] = c
		}
		c.Records++
		c.Minutes += rec.TimeDuration
		total += rec.TimeDuration
	}

	clients := make([]entity.WeeklyDigestClient, 0, len(byClient))
	for _, c := range byClient {
		c.SharePercent = percentOf(c.Minutes, total)
		clients = append(clients, *c)
	}
	sort.Slice(clients, func(i, j int) bool {
		if clients[i].Minutes != clients[j].Minutes {
			return clients[i].Minutes > clients[j].Minutes
		}
		return clients[i].Client < clients[j].Client
	})
	return clients
}

// digestUsage folds the per-client counters into one entry per event.
func digestUsage(counters []entity.UsageCounter) []entity.WeeklyDigestUsage {
	byEvent := make(map[string]*entity.WeeklyDigestUsage)
	for _, c := range counters {
		if c.Count <= 0 {
			continue
		}
		u, ok := byEvent[c.Event]
		if !ok {
			u = &entity.WeeklyDigestUsage{Event: c.Event, Clients: []entity.WeeklyDigestUsageClient{}}
			byEvent[c.Event] = u
		}
		u.Count += c.Count
		u.Clients = append(u.Clients, entity.WeeklyDigestUsageClient{Client: c.Client, Count: c.Count})
	}

	usage := make([]entity.WeeklyDigestUsage, 0, len(byEvent))
	appendEvent := func(event string) {
		u, ok := byEvent[event]
		if !ok {
			return
		}
		sort.Slice(u.Clients, func(i, j int) bool {
			if u.Clients[i].Count != u.Clients[j].Count {
				return u.Clients[i].Count > u.Clients[j].Count
			}
			return u.Clients[i].Client < u.Clients[j].Client
		})
		usage = append(usage, *u)
		delete(byEvent, event)
	}
	for _, event := range usageEventOrder {
		appendEvent(event)
	}
	// Events this build does not know (counted by a newer one) go last, by name.
	rest := make([]string, 0, len(byEvent))
	for event := range byEvent {
		rest = append(rest, event)
	}
	sort.Strings(rest)
	for _, event := range rest {
		appendEvent(event)
	}
	return usage
}

// digestIsEmpty reports a week with no logged time and no counted choices.
func digestIsEmpty(d entity.WeeklyDigest) bool {
	return d.TotalDone == 0 && len(d.Usage) == 0
}

// GetWeeklyDigest builds the end-of-week debrief. "previous" exists because the weekly
// wipe happens on Monday, when the finished week is no longer the current one. Once that
// week's records are wiped, "previous" answers from the snapshot taken before the wipe.
func (s *StatisticService) GetWeeklyDigest(week string) (entity.WeeklyDigest, error) {
	digest, err := s.liveWeeklyDigest(week)
	if err != nil {
		return entity.WeeklyDigest{}, err
	}
	if digest.Week != WeekPrevious || !digestIsEmpty(digest) || s.digestStore == nil {
		return digest, nil
	}

	snapshot, found, err := s.digestStore.GetDigestSnapshot()
	if err != nil {
		slog.Warn("GetWeeklyDigest: failed to read digest snapshot", "err", err)
		return digest, nil
	}
	if !found || snapshot.WeekStartDate != digest.WeekStartDate {
		return digest, nil
	}
	var stored entity.WeeklyDigest
	if err := json.Unmarshal([]byte(snapshot.Payload), &stored); err != nil {
		slog.Warn("GetWeeklyDigest: digest snapshot is not readable", "err", err)
		return digest, nil
	}
	stored.Week = WeekPrevious
	stored.Source = entity.DigestSourceSnapshot
	stored.SnapshotTakenAt = snapshot.TakenAt.UTC().Format(time.RFC3339)
	return stored, nil
}

// SnapshotBeforeClean stores the digest of the week the wipe is about to erase: the
// previous week when it still has records (the usual Monday wipe), otherwise the current
// one. An empty digest is not stored, so a repeated wipe keeps the earlier snapshot.
func (s *StatisticService) SnapshotBeforeClean() {
	if s.digestStore == nil {
		return
	}
	digest, err := s.liveWeeklyDigest(WeekPrevious)
	if err == nil && digestIsEmpty(digest) {
		digest, err = s.liveWeeklyDigest(WeekCurrent)
	}
	if err != nil {
		slog.Warn("SnapshotBeforeClean: failed to build digest", "err", err)
		return
	}
	if digestIsEmpty(digest) {
		slog.Info("SnapshotBeforeClean: nothing logged, keeping the existing snapshot")
		return
	}
	payload, err := json.Marshal(digest)
	if err != nil {
		slog.Warn("SnapshotBeforeClean: failed to encode digest", "err", err)
		return
	}
	snapshot := entity.DigestSnapshot{
		WeekStartDate: digest.WeekStartDate,
		TakenAt:       s.now().UTC(),
		Payload:       string(payload),
	}
	if err := s.digestStore.SaveDigestSnapshot(snapshot); err != nil {
		slog.Warn("SnapshotBeforeClean: failed to save digest snapshot", "err", err)
		return
	}
	slog.Info("SnapshotBeforeClean: digest snapshot saved", "week_start", digest.WeekStartDate)
}

// liveWeeklyDigest computes the digest from the records and counters currently stored.
func (s *StatisticService) liveWeeklyDigest(week string) (entity.WeeklyDigest, error) {
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
		Source:        entity.DigestSourceLive,
		Clients:       []entity.WeeklyDigestClient{},
		Usage:         []entity.WeeklyDigestUsage{},
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

	dates := make([]string, 0, len(stats.Days))
	for _, day := range stats.Days {
		dates = append(dates, day.Date)
	}
	records, err := s.st.GetRecordsForDates(dates)
	if err != nil {
		return entity.WeeklyDigest{}, fmt.Errorf("failed to get records for week: %w", err)
	}
	digest.Clients = digestClients(records)

	if s.digestStore != nil {
		counters, err := s.digestStore.GetUsageCounters(stats.WeekStartDate)
		if err != nil {
			slog.Warn("GetWeeklyDigest: failed to read usage counters", "err", err)
		} else {
			digest.Usage = digestUsage(counters)
		}
	}

	return digest, nil
}
