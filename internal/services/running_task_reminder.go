package services

import (
	"log/slog"
	"time"
)

// RunningReminderInterval is how often a run nobody watches gets a "still running" card.
const RunningReminderInterval = 5 * time.Minute

// heartbeatFreeClients start or resume a task without keeping a timer open (a Telegram button,
// an MCP call, a Mini App closed right after the tap), so nothing renews the heartbeat lease.
var heartbeatFreeClients = map[string]bool{"bot": true, "mcp": true, "miniapp": true}

// HeartbeatExempt reports whether a run started or resumed by client is kept off the heartbeat
// lease. Such a run still stops at its target or at the free-timer safety cap, and the watchdog
// reminds about it in Telegram instead (RemindIfDue).
func HeartbeatExempt(client string) bool {
	return heartbeatFreeClients[client]
}

// RemindIfDue sends the "still running" card for a heartbeat-exempt run once every
// RunningReminderInterval. It stays quiet while some client is still sending heartbeats,
// since an open timer already shows the run. It reports whether a card was sent.
func (s *RunningTaskService) RemindIfDue(taskName string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.nt == nil {
		return false
	}
	// Re-read under the lock: the run may have been stopped or paused since the watchdog listed it.
	task, err := s.st.GetRunningTask(taskName)
	if err != nil || task.TaskName == "" || !task.IsRunning || !HeartbeatExempt(task.RunClient) {
		return false
	}
	if now.Sub(task.LastHeartbeatAt) <= DefaultHeartbeatLease {
		return false
	}
	last := task.StartTime
	if task.ReminderAt.After(last) {
		last = task.ReminderAt
	}
	if now.Sub(last) < RunningReminderInterval {
		return false
	}

	elapsed := task.Accumulated + int(now.Sub(task.StartTime).Minutes())
	msgID, err := s.nt.SendMessageReminder(task.TaskName, elapsed, task.TargetDuration, task.ReminderMessageID)
	// A failed send is retried at the next interval, not on every watchdog tick.
	task.ReminderAt = now
	sent := err == nil
	if sent {
		task.ReminderMessageID = msgID
	} else {
		slog.Warn("RunningTaskService: failed to send running reminder", "task", task.TaskName, "error", err)
	}
	if err := s.st.UpsertRunningTask(task); err != nil {
		slog.Error("RunningTaskService: failed to save reminder state", "task", task.TaskName, "error", err)
	}
	return sent
}
