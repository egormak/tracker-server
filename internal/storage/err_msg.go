package storage

import "errors"

var ErrListEmpty = errors.New("list is empty")
var ErrTaskNotFound = errors.New("this task is not found")
var ErrScheduleOld = errors.New("schedule is old")
var ErrParamsOld = errors.New("params is old")
