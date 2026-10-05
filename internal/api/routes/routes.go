package routes

import (
	"tracker-server/config"
	"tracker-server/internal/api/handler"
	"tracker-server/internal/api/middleware"
	"tracker-server/internal/notify"
	"tracker-server/internal/realtime"
	"tracker-server/internal/services"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, mongoconn storage.Storage, notify notify.Notify, cfg config.Config, hub *realtime.Hub) (*services.RunningTaskService, *services.DayRollService) {

	// Services
	taskService := services.NewTaskService(mongoconn, notify)
	rampService := services.NewRampService(mongoconn)
	taskRecordService := services.NewTaskRecordService(mongoconn, rampService)
	restService := services.NewRestService(mongoconn)
	scheduleService := services.NewScheduleService(mongoconn)
	statsService := services.NewStatisticService(mongoconn, scheduleService)
	manageService := services.NewManageService(mongoconn)
	runningTaskService := services.NewRunningTaskService(mongoconn, notify)
	runningTaskService.SetRampService(rampService)
	if hub != nil {
		runningTaskService.SetHub(hub)
	}
	eveningService := services.NewEveningService(statsService, restService)
	dayRollService := services.NewDayRollService(scheduleService)

	// Domain Handlers
	taskHandler := handler.NewTaskHandler(taskService)
	taskRecordHandler := handler.NewTaskRecordHandler(taskRecordService, scheduleService)
	rampHandler := handler.NewRampHandler(rampService)
	restHandler := handler.NewRestHandler(restService)
	statsHandler := handler.NewStatisticHandler(statsService, runningTaskService, restService, eveningService)
	manageHandler := handler.NewManageHandler(manageService)
	scheduleHandler := handler.NewScheduleHandler(scheduleService)
	runningTaskHandler := handler.NewRunningTaskHandler(runningTaskService, hub)
	eveningHandler := handler.NewEveningHandler(eveningService)
	roleHandler := handler.NewRoleHandler(mongoconn, notify)
	legacyTimerHandler := handler.NewLegacyTimerHandler(mongoconn)
	telegramHandler := handler.NewTelegramHandler(notify)

	// Routes
	tgAuth := middleware.TelegramAuth(cfg)
	api := app.Group("/api", tgAuth)

	// Dashboard
	api.Get("/v1/dashboard/state", statsHandler.GetDashboardState)

	// Task
	api.Get("/v1/task/params", taskHandler.TaskParams)
	api.Post("/v1/task/params", taskHandler.SetTaskParams)
	api.Get("/v1/record/task-day", taskHandler.GetDayTaskRecord) // Legacy endpoint for CLI

	// TaskRecords
	api.Post("/v1/taskrecord", taskRecordHandler.AddRecord)
	api.Get("/v1/task/plan/percent", taskRecordHandler.GetTaskPlanPercent)
	api.Get("/v1/task/plan/percent/schedule", taskRecordHandler.GetTaskPlanPercentWithSchedule)
	api.Post("/v1/task/plan/rotate", taskRecordHandler.ChangeGroupPlanPercent)

	// Rest
	api.Post("/v1/rest/add", restHandler.RestAdd)
	api.Post("/v1/rest/spend", restHandler.RestSpend)
	api.Get("/v1/rest/get", restHandler.RestGet)
	api.Post("/v1/rest/reset", restHandler.RestReset)

	// Ramp
	api.Get("/v1/ramp/status", rampHandler.GetStatus)
	api.Post("/v1/ramp/reset", rampHandler.Reset)
	api.Post("/v1/ramp/advance", rampHandler.Advance)
	api.Get("/v1/ramp/config", rampHandler.GetConfig)
	api.Put("/v1/ramp/config", rampHandler.UpdateConfig)

	// Manage
	api.Post("/v1/manage/task/create", manageHandler.CreateTask)

	// Statistics
	api.Get("/v1/stats/done/today", statsHandler.StatCompletionTimeDone)
	api.Get("/v1/stats/tasks/today", statsHandler.StatCompletionTimeDone)
	api.Get("/v1/stats/weekly", statsHandler.GetWeeklyStats)
	api.Get("/v1/tasklist", statsHandler.ShowTaskList) // Legacy endpoint for CLI and web UI

	// Plan Percents
	api.Get("/v1/manage/plan-percents", manageHandler.GetPlanPercents)                    // New route for plan percents
	api.Delete("/v1/manage/plan-percents/:group/:value", manageHandler.DeletePlanPercent) // Remove specific plan percent
	api.Post("/v1/manage/procents", manageHandler.ProcentsSet)
	api.Get("/v1/manage/procents", manageHandler.GetPlanProcentsLegacy)

	// Legacy Countdown & Global Timers
	api.Post("/v1/timer/set", legacyTimerHandler.TimerSet)
	api.Get("/v1/timer/get", legacyTimerHandler.TimerGet)
	api.Post("/v1/timer/del", legacyTimerHandler.TimerDel)
	api.Post("/v1/manage/timer/global", legacyTimerHandler.TimerGlobalSet)
	api.Get("/v1/manage/timer/global", legacyTimerHandler.TimerGlobalGet)

	// Records
	api.Get("/v1/records", taskRecordHandler.ShowRecords)
	api.Post("/v1/records/clean", taskRecordHandler.CleanRecords)

	// Roles
	api.Get("/v1/roles/records", roleHandler.ShowRolesRecords)
	api.Get("/v1/roles/records/today", roleHandler.StatCompletionTimeDone)
	api.Get("/v1/role/recheck", roleHandler.RecheckRole)
	api.Get("/v1/role/get", roleHandler.TaskRoleGet)

	// Telegram
	api.Post("/v1/manage/telegram/start", telegramHandler.TelegramSendStart)
	api.Post("/v1/manage/telegram/stop", telegramHandler.TelegramSendStop)
	api.Post("/v1/manage/telegram/message", telegramHandler.TelegramSendCustom)

	// Schedule
	api.Post("/v1/schedule", scheduleHandler.CreateSchedule)
	api.Get("/v1/schedule/active", scheduleHandler.GetActiveSchedule)
	api.Patch("/v1/schedule/active/task-time", scheduleHandler.UpdateTaskTime)
	api.Get("/v1/schedule/active/today", scheduleHandler.GetTodaySchedule)
	api.Get("/v1/schedule/active/rollover", scheduleHandler.GetRolloverTasks)
	api.Post("/v1/schedule/apply", scheduleHandler.ApplySchedule)
	api.Get("/v1/schedule/:id", scheduleHandler.GetSchedule)
	api.Put("/v1/schedule/:id", scheduleHandler.UpdateSchedule)
	api.Delete("/v1/schedule/:id", scheduleHandler.DeleteSchedule)
	api.Put("/v1/schedule/:id/activate", scheduleHandler.SetActiveSchedule)

	// Running Task
	api.Post("/v1/timer/run/start", runningTaskHandler.Start)
	api.Post("/v1/timer/run/stop", runningTaskHandler.Stop)
	api.Post("/v1/timer/run/pause", runningTaskHandler.Pause)
	api.Post("/v1/timer/run/resume", runningTaskHandler.Resume)
	api.Post("/v1/timer/run/adjust", runningTaskHandler.Adjust)
	api.Get("/v1/timer/run/status", runningTaskHandler.Status)
	api.Get("/v1/timer/run/list", runningTaskHandler.List)
	api.Post("/v1/timer/run/heartbeat", runningTaskHandler.Heartbeat)
	api.Get("/v1/timer/ws", runningTaskHandler.WebSocketUpgradeCheck, runningTaskHandler.WebSocketHandler())

	// Evening Mode
	api.Get("/v1/mode/evening-focus", eveningHandler.GetEveningFocus)
	api.Post("/v1/mode/evening-focus/skip", eveningHandler.SkipTask)

	app.Get("/", handler.Welcome)

	return runningTaskService, dayRollService
}
