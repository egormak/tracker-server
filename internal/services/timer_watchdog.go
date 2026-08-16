package services

import (
	"context"
	"log/slog"
	"time"
)

const (
	DefaultSafetyCapMinutes   = 20
	DefaultHeartbeatLease     = 3 * time.Minute
	WatchdogCheckInterval     = 10 * time.Second
)

type TimerWatchdog struct {
	runningService *RunningTaskService
	safetyCapMin   int
	leaseTimeout   time.Duration
	stopCh         chan struct{}
}

func NewTimerWatchdog(runningService *RunningTaskService) *TimerWatchdog {
	return &TimerWatchdog{
		runningService: runningService,
		safetyCapMin:   DefaultSafetyCapMinutes,
		leaseTimeout:   DefaultHeartbeatLease,
		stopCh:         make(chan struct{}),
	}
}

func (w *TimerWatchdog) Start(ctx context.Context) {
	slog.Info("TimerWatchdog: started background watchdog service",
		"safety_cap_min", w.safetyCapMin,
		"heartbeat_lease", w.leaseTimeout,
		"check_interval", WatchdogCheckInterval,
	)

	ticker := time.NewTicker(WatchdogCheckInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("TimerWatchdog: context cancelled, stopping watchdog")
				return
			case <-w.stopCh:
				slog.Info("TimerWatchdog: received stop signal")
				return
			case <-ticker.C:
				w.checkRunningTasks()
			}
		}
	}()
}

func (w *TimerWatchdog) Stop() {
	close(w.stopCh)
}

func (w *TimerWatchdog) checkRunningTasks() {
	tasks, err := w.runningService.GetAllTasks()
	if err != nil {
		slog.Error("TimerWatchdog: failed to get running tasks", "error", err)
		return
	}

	now := time.Now()
	for _, task := range tasks {
		if !task.IsRunning {
			continue
		}

		elapsedMin := task.Accumulated + int(now.Sub(task.StartTime).Minutes())

		// Rule 1: Target Duration reached
		if task.TargetDuration > 0 && elapsedMin >= task.TargetDuration {
			slog.Info("TimerWatchdog: target duration reached, auto-stopping task",
				"task", task.TaskName,
				"target_min", task.TargetDuration,
				"elapsed_min", elapsedMin,
			)
			if _, err := w.runningService.Stop(task.TaskName, "target_reached"); err != nil {
				slog.Error("TimerWatchdog: error auto-stopping task on target reached", "task", task.TaskName, "error", err)
			}
			continue
		}

		// Rule 2: Absolute Safety Cap (20 minutes for free stopwatch)
		if task.TargetDuration == 0 && elapsedMin >= w.safetyCapMin {
			slog.Info("TimerWatchdog: absolute safety cap reached for free timer, auto-stopping task",
				"task", task.TaskName,
				"safety_cap_min", w.safetyCapMin,
				"elapsed_min", elapsedMin,
			)
			if _, err := w.runningService.Stop(task.TaskName, "safety_cap_reached"); err != nil {
				slog.Error("TimerWatchdog: error auto-stopping task on safety cap reached", "task", task.TaskName, "error", err)
			}
			continue
		}

		// Rule 3: Heartbeat Lease Timeout
		if !task.LastHeartbeatAt.IsZero() && now.Sub(task.LastHeartbeatAt) > w.leaseTimeout {
			slog.Warn("TimerWatchdog: heartbeat lease expired, pausing task",
				"task", task.TaskName,
				"last_heartbeat", task.LastHeartbeatAt,
				"elapsed_since_heartbeat", now.Sub(task.LastHeartbeatAt),
			)
			if _, err := w.runningService.Pause(task.TaskName, "heartbeat_timeout"); err != nil {
				slog.Error("TimerWatchdog: error auto-pausing task on heartbeat timeout", "task", task.TaskName, "error", err)
			}
			continue
		}
	}
}
