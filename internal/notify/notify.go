package notify

import "tracker-server/internal/domain/entity"

type Notify interface {
	SendMessageStart(taskName string) (int, error)
	SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error
	SendMessageCompletion(taskName string, timeDone int, todayDone int, targetDuration int, remainingTasks []string, nextTask string, msgID int, breakOptions []entity.BreakOption, notes []string) error
	SendCustomMessage(message string) error
	// DeleteMessage removes a card, e.g. the start card of a run too short to record.
	DeleteMessage(msgID int) error
	// SendMessageReminder posts a "still running" card with pause/stop buttons and removes the
	// previous one (prevMsgID, 0 = none). It returns the new card's ID.
	SendMessageReminder(taskName string, elapsedMin int, targetMin int, prevMsgID int) (int, error)
}
