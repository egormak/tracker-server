package services

import (
	"context"
	"errors"
	"testing"
	"time"
	"tracker-server/internal/domain/entity"
)

func TestDayRollService_Start_EnsuresScheduleOnStartup(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	// Wednesday, 23 September 2026
	fixedTime := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)
	scheduleService.SetNowFunc(func() time.Time { return fixedTime })

	sched := entity.WeeklySchedule{
		ID:       "sched_active",
		IsActive: true,
		Wednesday: entity.DaySchedule{
			Day:       "wednesday",
			TotalTime: 180,
			Tasks: []entity.ScheduleTask{
				{Name: "Morning Dev", Role: "work", Time: 180, Priority: 1},
			},
		},
	}
	storage.schedules["sched_active"] = sched
	storage.activeID = "sched_active"

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetNowFunc(func() time.Time { return fixedTime })
	dayRoll.SetInterval(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dayRoll.Start(ctx)
	defer dayRoll.Stop()

	dateStr := fixedTime.Format("2 January 2006")
	has, _ := storage.HasTasksForDate(dateStr)
	if !has {
		t.Fatalf("expected tasks to be created for today on startup")
	}

	tasks := storage.tasksByDate[dateStr]
	if len(tasks) != 1 || tasks[0].Name != "Morning Dev" {
		t.Errorf("unexpected tasks created on startup: %+v", tasks)
	}

	if dayRoll.GetLastDate() != dateStr {
		t.Errorf("expected lastDate=%s, got %s", dateStr, dayRoll.GetLastDate())
	}
}

func TestDayRollService_CheckDayRoll_MidnightTransition(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	currentTime := time.Date(2026, 9, 23, 23, 59, 0, 0, time.UTC) // Wednesday
	timeProvider := func() time.Time { return currentTime }
	scheduleService.SetNowFunc(timeProvider)

	sched := entity.WeeklySchedule{
		ID:       "sched_active",
		IsActive: true,
		Wednesday: entity.DaySchedule{
			Day:       "wednesday",
			TotalTime: 100,
			Tasks: []entity.ScheduleTask{
				{Name: "Wed Task", Role: "work", Time: 100, Priority: 1},
			},
		},
		Thursday: entity.DaySchedule{
			Day:       "thursday",
			TotalTime: 200,
			Tasks: []entity.ScheduleTask{
				{Name: "Thu Task", Role: "work", Time: 200, Priority: 1},
			},
		},
	}
	storage.schedules["sched_active"] = sched
	storage.activeID = "sched_active"

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetNowFunc(timeProvider)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dayRoll.Start(ctx)
	defer dayRoll.Stop()

	wedDate := currentTime.Format("2 January 2006")
	if dayRoll.GetLastDate() != wedDate {
		t.Fatalf("expected lastDate=%s, got %s", wedDate, dayRoll.GetLastDate())
	}

	// Advance clock past midnight to Thursday
	currentTime = time.Date(2026, 9, 24, 0, 1, 0, 0, time.UTC)
	thuDate := currentTime.Format("2 January 2006")

	// Trigger CheckDayRoll
	err := dayRoll.CheckDayRoll()
	if err != nil {
		t.Fatalf("unexpected error from CheckDayRoll: %v", err)
	}

	if dayRoll.GetLastDate() != thuDate {
		t.Errorf("expected lastDate updated to %s, got %s", thuDate, dayRoll.GetLastDate())
	}

	thuTasks := storage.tasksByDate[thuDate]
	if len(thuTasks) != 1 || thuTasks[0].Name != "Thu Task" {
		t.Errorf("expected Thursday task created after midnight rollover, got: %+v", thuTasks)
	}
}

func TestDayRollService_CheckDayRoll_SameDayNoop(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	fixedTime := time.Date(2026, 9, 23, 14, 0, 0, 0, time.UTC)
	scheduleService.SetNowFunc(func() time.Time { return fixedTime })

	sched := entity.WeeklySchedule{
		ID:       "sched_active",
		IsActive: true,
		Wednesday: entity.DaySchedule{
			Day:       "wednesday",
			TotalTime: 100,
			Tasks: []entity.ScheduleTask{
				{Name: "Task 1", Role: "work", Time: 100, Priority: 1},
			},
		},
	}
	storage.schedules["sched_active"] = sched
	storage.activeID = "sched_active"

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetNowFunc(func() time.Time { return fixedTime })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dayRoll.Start(ctx)
	defer dayRoll.Stop()

	// Calling CheckDayRoll on the same day should return nil without repeating apply
	err := dayRoll.CheckDayRoll()
	if err != nil {
		t.Fatalf("unexpected error from CheckDayRoll: %v", err)
	}

	dateStr := fixedTime.Format("2 January 2006")
	if len(storage.tasksByDate[dateStr]) != 1 {
		t.Errorf("expected task count to remain 1, got %d", len(storage.tasksByDate[dateStr]))
	}
}

func TestDayRollService_Start_ContextCancellation(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetInterval(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	dayRoll.Start(ctx)

	// Cancel context immediately
	cancel()
	// Allow goroutine to exit cleanly
	time.Sleep(20 * time.Millisecond)
}

func TestDayRollService_Stop(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetInterval(10 * time.Millisecond)

	ctx := context.Background()
	dayRoll.Start(ctx)
	dayRoll.Stop()
	// Double stop should be safe
	dayRoll.Stop()
}

func TestDayRollService_StartupWithoutActiveSchedule(t *testing.T) {
	storage := NewMockScheduleStorage()
	// No schedule configured in storage
	scheduleService := NewScheduleService(storage)

	dayRoll := NewDayRollService(scheduleService)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Must not panic
	dayRoll.Start(ctx)
	defer dayRoll.Stop()
}

func TestDayRollService_CheckDayRoll_DoesNotAdvanceDateOnFailure(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	currentTime := time.Date(2026, 9, 23, 23, 59, 0, 0, time.UTC) // Wednesday
	timeProvider := func() time.Time { return currentTime }
	scheduleService.SetNowFunc(timeProvider)

	sched := entity.WeeklySchedule{
		ID:       "sched_active",
		IsActive: true,
		Wednesday: entity.DaySchedule{
			Day:       "wednesday",
			TotalTime: 100,
			Tasks:     []entity.ScheduleTask{{Name: "Wed Task", Role: "work", Time: 100, Priority: 1}},
		},
		Thursday: entity.DaySchedule{
			Day:       "thursday",
			TotalTime: 200,
			Tasks:     []entity.ScheduleTask{{Name: "Thu Task", Role: "work", Time: 200, Priority: 1}},
		},
	}
	storage.schedules["sched_active"] = sched
	storage.activeID = "sched_active"

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetNowFunc(timeProvider)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dayRoll.Start(ctx)
	defer dayRoll.Stop()

	wedDate := currentTime.Format("2 January 2006")
	if dayRoll.GetLastDate() != wedDate {
		t.Fatalf("expected initial lastDate=%s, got %s", wedDate, dayRoll.GetLastDate())
	}

	// Advance clock to Thursday
	currentTime = time.Date(2026, 9, 24, 0, 1, 0, 0, time.UTC)
	// Simulate transient storage error on Thursday check
	storage.getDayScheduleErr = errors.New("transient mongo connection error")

	err := dayRoll.CheckDayRoll()
	if err == nil {
		t.Fatalf("expected error from CheckDayRoll during transient storage failure")
	}

	// Crucial: lastDate must NOT have advanced to Thursday, ensuring retry is possible
	if dayRoll.GetLastDate() != wedDate {
		t.Errorf("expected lastDate to remain %s after failure, but got %s", wedDate, dayRoll.GetLastDate())
	}

	// Now recover storage: next check should retry and succeed
	storage.getDayScheduleErr = nil
	err = dayRoll.CheckDayRoll()
	if err != nil {
		t.Fatalf("expected retry to succeed after storage recovery: %v", err)
	}

	thuDate := currentTime.Format("2 January 2006")
	if dayRoll.GetLastDate() != thuDate {
		t.Errorf("expected lastDate updated to %s upon successful retry, got %s", thuDate, dayRoll.GetLastDate())
	}
}

func TestDayRollService_CheckDayRoll_PreservesUserTasksAtMidnight(t *testing.T) {
	storage := NewMockScheduleStorage()
	scheduleService := NewScheduleService(storage)

	currentTime := time.Date(2026, 9, 23, 23, 59, 0, 0, time.UTC) // Wednesday
	timeProvider := func() time.Time { return currentTime }
	scheduleService.SetNowFunc(timeProvider)

	sched := entity.WeeklySchedule{
		ID:       "sched_active",
		IsActive: true,
		Wednesday: entity.DaySchedule{
			Day:       "wednesday",
			TotalTime: 100,
			Tasks:     []entity.ScheduleTask{{Name: "Wed Task", Role: "work", Time: 100, Priority: 1}},
		},
		Thursday: entity.DaySchedule{
			Day:       "thursday",
			TotalTime: 200,
			Tasks:     []entity.ScheduleTask{{Name: "Thu Task", Role: "work", Time: 200, Priority: 1}},
		},
	}
	storage.schedules["sched_active"] = sched
	storage.activeID = "sched_active"

	dayRoll := NewDayRollService(scheduleService)
	dayRoll.SetNowFunc(timeProvider)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dayRoll.Start(ctx)
	defer dayRoll.Stop()

	// Advance to Thursday 00:00:01
	currentTime = time.Date(2026, 9, 24, 0, 0, 1, 0, time.UTC)
	thuDate := currentTime.Format("2 January 2006")

	// User manually created a task for Thursday before the ticker fires
	storage.tasksByDate[thuDate] = []entity.TaskDefinition{
		{Name: "Ad-hoc Midnight Task", Role: "work", TimeSchedule: 45, Date: thuDate},
	}

	// Ticker runs CheckDayRoll
	err := dayRoll.CheckDayRoll()
	if err != nil {
		t.Fatalf("unexpected error from CheckDayRoll: %v", err)
	}

	// Tracked date should advance
	if dayRoll.GetLastDate() != thuDate {
		t.Errorf("expected lastDate updated to %s, got %s", thuDate, dayRoll.GetLastDate())
	}

	// Both user's ad-hoc task AND Thursday's scheduled task must be present!
	tasks := storage.tasksByDate[thuDate]
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks (1 ad-hoc + 1 scheduled) for Thursday, got %d: %+v", len(tasks), tasks)
	}

	hasAdHoc := false
	hasThuTask := false
	for _, task := range tasks {
		if task.Name == "Ad-hoc Midnight Task" {
			hasAdHoc = true
		}
		if task.Name == "Thu Task" {
			hasThuTask = true
		}
	}
	if !hasAdHoc {
		t.Errorf("expected user's ad-hoc task to be preserved")
	}
	if !hasThuTask {
		t.Errorf("expected Thursday's scheduled task to be applied on rollover")
	}
}
