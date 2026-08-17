package services

import (
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"tracker-server/internal/domain/entity"
)

type EveningService struct {
	statSrv        *StatisticService
	snoozedTonight map[string]bool
	lastResetDate  string
	nowFunc        func() time.Time
	mu             sync.Mutex
}

func NewEveningService(statSrv *StatisticService) *EveningService {
	return &EveningService{
		statSrv:        statSrv,
		snoozedTonight: make(map[string]bool),
		nowFunc:        time.Now,
	}
}

func (s *EveningService) checkDailyResetLocked() {
	now := time.Now()
	if s.nowFunc != nil {
		now = s.nowFunc()
	}
	today := now.Format("2006-01-02")
	if s.lastResetDate != today {
		s.snoozedTonight = make(map[string]bool)
		s.lastResetDate = today
		slog.Info("Reset snoozed tasks for new day in evening focus", "date", today)
	}
}

func (s *EveningService) GetEveningFocus(category string, timeOverride int) (entity.EveningFocusResponse, error) {
	stats, err := s.statSrv.GetWeeklyStats()
	if err != nil {
		slog.Error("Failed to get weekly stats for evening focus", "err", err)
		return entity.EveningFocusResponse{}, err
	}

	s.mu.Lock()
	s.checkDailyResetLocked()
	snoozedSnapshot := make(map[string]bool, len(s.snoozedTonight))
	for k, v := range s.snoozedTonight {
		snoozedSnapshot[k] = v
	}
	s.mu.Unlock()

	type item struct {
		candidate entity.EveningFocusCandidate
		timeDone  int
	}

	var items []item

	for _, taskTarget := range stats.WeeklyTasks {
		lowerName := strings.ToLower(taskTarget.Name)
		// Strict exclusion of work and english
		if lowerName == "work" || lowerName == "english" {
			continue
		}

		// Filter category if specified (e.g. "learn")
		if category != "" && !strings.EqualFold(category, "all") && !strings.EqualFold(taskTarget.Role, category) {
			continue
		}

		// Skip snoozed for tonight
		if snoozedSnapshot[lowerName] {
			continue
		}

		gap := taskTarget.TimeDuration - taskTarget.TimeDone
		items = append(items, item{
			candidate: entity.EveningFocusCandidate{
				TaskName:  taskTarget.Name,
				Role:      taskTarget.Role,
				WeeklyGap: gap,
				Priority:  5,
				IsStrict:  false,
			},
			timeDone: taskTarget.TimeDone,
		})
	}

	// Sort candidates by least time done for the week (ascending)
	// Tie-break by highest remaining weekly deficit (WeeklyGap) descending
	sort.Slice(items, func(i, j int) bool {
		if items[i].timeDone != items[j].timeDone {
			return items[i].timeDone < items[j].timeDone
		}
		return items[i].candidate.WeeklyGap > items[j].candidate.WeeklyGap
	})

	candidates := make([]entity.EveningFocusCandidate, len(items))
	for i, it := range items {
		candidates[i] = it.candidate
	}

	sprintTime := 20
	if timeOverride > 0 {
		sprintTime = timeOverride
	}

	currentTask := entity.EveningFocusCandidate{}
	if len(candidates) > 0 {
		currentTask = candidates[0]
	}

	return entity.EveningFocusResponse{
		CurrentTask: currentTask,
		Candidates:  candidates,
		SprintTime:  sprintTime,
		RestPool:    10,
	}, nil
}

func (s *EveningService) SkipTask(taskName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkDailyResetLocked()

	if taskName != "" {
		lowerName := strings.ToLower(taskName)
		s.snoozedTonight[lowerName] = true
		slog.Info("Snoozed task for evening catch-up", "task", lowerName)
	}
	return nil
}
