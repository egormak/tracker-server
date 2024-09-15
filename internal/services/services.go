package services

import (
	"tracker-server/internal/notify"
	"tracker-server/internal/storage"
)

type Service struct {
	st storage.Storage
	nt notify.Notify
}

func NewService(st storage.Storage, nt notify.Notify) *Service {
	return &Service{st: st, nt: nt}
}
