package notify

import "tracker-server/internal/domain/entity"

type Notify interface {
	SendMessageStart(taskName string) (int, error)
	SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error
	SendMessageCompletion(taskName string, timeDone int, todayDone int, targetDuration int, remainingTasks []string, nextTask string, msgID int, breakOptions []entity.BreakOption) error
	SendCustomMessage(message string) error
}
