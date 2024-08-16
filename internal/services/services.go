package services

import (
	"tracker-server/internal/notify"
	"tracker-server/internal/storage"
)

type ServicesInt interface {
	StatCompletionTimeDone() ([]storage.TaskRecord, error)
}

type Service struct {
	st storage.Storage
	nt notify.Notify
}
