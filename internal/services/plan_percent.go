package services

import (
	"fmt"
	"log/slog"
	"tracker-server/internal/domain/entity"
	"tracker-server/internal/storage"
)

func (s *TaskRecordService) GetTaskPlanPercent() (entity.PlanPercentResponse, error) {

	var planPercent entity.PlanPercentResponse
	// var GroupPlanOrdinal int
	// var GroupPercent int

	for {
		GroupPlanOrdinal, err := s.st.GetGroupPlanPercent()
		if err != nil {
			errMsg := fmt.Errorf("can't get group percent: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_group_percent", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}
		GroupPercent, err := s.st.GetGroupPercent(GroupPlanOrdinal)
		if err != nil {
			if err == storage.ErrListEmpty {
				if err := s.st.CheckIfPlanPercentEmpty(); err != nil {
					if err == storage.ErrAllEmpty {
						return entity.PlanPercentResponse{}, storage.ErrAllEmpty
					} else {
						errMsg := fmt.Errorf("can't check if plan percent empty: %s", err)
						slog.Error("task_record_service, get_task_plan_percent:check_if_plan_percent_empty", "err", errMsg)
						return entity.PlanPercentResponse{}, errMsg
					}
				}
			}
		}
		s.st.ChangeGroupPlanPercent(GroupPlanOrdinal)
		groupName, err := s.st.GetGroupName(GroupPlanOrdinal)
		if err != nil {
			errMsg := fmt.Errorf("can't get group name: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_group_name", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}
		TaskNamePlanPercent, err := s.st.GetTaskNamePlanPercent(groupName, GroupPercent)
		if err != nil {
			errMsg := fmt.Errorf("can't get task name plan percent: %s", err)
			slog.Error("task_record_service, get_task_plan_percent:get_task_name_plan_percent", "err", errMsg)
			return entity.PlanPercentResponse{}, errMsg
		}

		if TaskNamePlanPercent != "" {
			timeLeft, _ := s.GetTodayTaskTimeLeft(TaskNamePlanPercent)
			planPercent = entity.PlanPercentResponse{
				TaskName: TaskNamePlanPercent,
				Percent:  GroupPercent,
				TimeLeft: timeLeft,
			}
			break
		} else if TaskNamePlanPercent == "" {
			if err := s.st.DelGroupPercent(groupName); err != nil {
				errMsg := fmt.Errorf("can't delete group percent: %s", err)
				slog.Error("task_record_service, get_task_plan_percent:del_group_percent", "err", errMsg)
				return entity.PlanPercentResponse{}, errMsg
			}
		}
	}

	return planPercent, nil

}
