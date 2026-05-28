package storage

type DayList struct {
	Title    string
	Count    int
	ListTime []int
}

type SchedulerInfo struct {
	Name        string
	Date        string
	ScheduleAll int
}
