package services

import (
	"fmt"
	"log/slog"
)

type RestStorage interface {
	AddRest(restTime int) error
	RestSpend(restTime int) error
	GetRest() (int, error)
}

type RestService struct {
	st RestStorage
}

func NewRestService(st RestStorage) *RestService {
	return &RestService{st: st}
}

func (r *RestService) RestSpend(restTime int) error {

	var errMsg error

	if err := r.st.RestSpend(restTime); err != nil {
		errMsg := fmt.Errorf("can't rest spend: %s", err)
		slog.Error("rest_service, rest_spend", "err", errMsg)
	}

	return errMsg
}

func (r *RestService) AddRest(restTime int) error {
	var errMsg error

	if err := r.st.AddRest(restTime); err != nil {
		errMsg := fmt.Errorf("can't add rest: %s", err)
		slog.Error("rest_service, add_rest", "err", errMsg)
	}

	return errMsg
}

func (r *RestService) RestGet() (int, error) {
	var errMsg error

	restTime, err := r.st.GetRest()
	if err != nil {
		errMsg := fmt.Errorf("can't get rest: %s", err)
		slog.Error("rest_service, get_rest", "err", errMsg)
	}

	return restTime, errMsg

}
