package notify

type Notify interface {
	SendMessageStart(taskName string) (int, error)
	SendMessageStop(taskName string, timeDone int, msgID int, timeEnd string) error
}
