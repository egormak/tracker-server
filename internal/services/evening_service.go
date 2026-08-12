package services

import (
	"log/slog"
	"sort"
	"strings"
	"sync"
	"tracker-server/internal/domain/entity"
)

type EveningService struct {
	statSrv        *StatisticService
	snoozedTonight map[string]bool
	mu             sync.RWMutex
}

func NewEveningService(statSrv *StatisticService) *EveningService {
	return &EveningService{
		statSrv:        statSrv,
		snoozedTonight: make(map[string]bool),
	}
}

func (s *EveningService) GetEveningFocus(category string, timeOverride int) (entity.EveningFocusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats, err := s.statSrv.GetWeeklyStats()
	if err != nil {
		slog.Error("Failed to get weekly stats for evening focus", "err", err)
		return entity.EveningFocusResponse{}, err
	}

	var candidates []entity.EveningFocusCandidate

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
		if s.snoozedTonight[lowerName] {
			continue
		}

		gap := taskTarget.TimeDuration - taskTarget.TimeDone
		candidates = append(candidates, entity.EveningFocusCandidate{
			TaskName:  taskTarget.Name,
			Role:      taskTarget.Role,
			WeeklyGap: gap,
			Priority:  5,
			IsStrict:  false,
		})
	}

	// Sort candidates by highest weekly deficit (gap) descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].WeeklyGap > candidates[j].WeeklyGap
	})

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

	if taskName != "" {
		lowerName := strings.ToLower(taskName)
		s.snoozedTonight[lowerName] = true
		slog.Info("Snoozed task for evening catch-up", "task", lowerName)
	}
	return nil
}
