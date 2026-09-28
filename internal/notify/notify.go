package notify

type Notify interface {
	SendMessageStart(taskName string) (int, error)
	SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error
	SendMessageCompletion(taskName string, timeDone int, todayDone int, targetDuration int, remainingTasks []string, nextTask string, msgID int) error
	SendCustomMessage(message string) error
}
