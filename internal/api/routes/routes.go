package routes

import (
	"tracker-server/config"
	"tracker-server/internal/api/handler"
	"tracker-server/internal/api/middleware"
	"tracker-server/internal/handler/manage"
	"tracker-server/internal/handler/role"
	"tracker-server/internal/handler/welcome"
	"tracker-server/internal/notify"
	"tracker-server/internal/services"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, mongoconn storage.Storage, notify notify.Notify, cfg config.Config) {

	// Services
	taskService := services.NewTaskService(mongoconn, notify)
	taskRecordService := services.NewTaskRecordService(mongoconn)
	restService := services.NewRestService(mongoconn)
	statsService := services.NewStatisticService(mongoconn)
	manageService := services.NewManageService(mongoconn)
	scheduleService := services.NewScheduleService(mongoconn)
	runningTaskService := services.NewRunningTaskService(mongoconn, notify)
	eveningService := services.NewEveningService(statsService)

	// Handlers
	// Task
	taskHandler := handler.NewTaskHandler(taskService)
	// TaskRecords
	taskRecordHandler := handler.NewTaskRecordHandler(taskRecordService, scheduleService)
	// Rest
	restHandler := handler.NewRestHandler(restService)
	// Statistics
	statsHandler := handler.NewStatisticHandler(statsService)
	// Manage
	manageHandler := handler.NewManageHandler(manageService)
	// Schedule
	scheduleHandler := handler.NewScheduleHandler(scheduleService)
	// Running Task
	runningTaskHandler := handler.NewRunningTaskHandler(runningTaskService)
	// Evening Mode
	eveningHandler := handler.NewEveningHandler(eveningService)

	// OLD Logic (manage and role handlers still needed for legacy routes)
	roleHandler := role.New(mongoconn, notify)
	manageHandlerOld := manage.New(mongoconn, notify)
	//

	// Routes
	tgAuth := middleware.TelegramAuth(cfg)
	api := app.Group("/api", tgAuth)
	// Task
	api.Get("/v1/task/params", taskHandler.TaskParams)
	api.Get("/v1/record/params", taskHandler.TaskParams)         // Legacy endpoint - same handler
	api.Get("/v1/record/task-day", taskHandler.GetDayTaskRecord) // Legacy endpoint for CLI
	// api.Post("/v1/task/create", taskHandler.CreateTask)
	// TaskRecords
	api.Post("/v1/taskrecord", taskRecordHandler.AddRecord)
	api.Post("/v1/record", taskRecordHandler.AddRecord) // Legacy endpoint - same handler
	// api.Get("/v1/task/next", taskRecordHandler.TasksNext)
	api.Get("/v1/task/plan/percent", taskRecordHandler.GetTaskPlanPercent)
	api.Get("/v1/task/plan-percent", taskRecordHandler.GetTaskPlanPercent) // Legacy alias (hyphen instead of slash)
	api.Get("/v1/task/plan/percent/schedule", taskRecordHandler.GetTaskPlanPercentWithSchedule)
	api.Get("/v1/task/plan-percent/change", taskRecordHandler.ChangeGroupPlanPercent) // Legacy rotation
	// Rest
	api.Post("/v1/rest/add", restHandler.RestAdd)
	api.Post("/v1/rest-spend", restHandler.RestSpend) // Remove in future
	api.Post("/v1/rest/spend", restHandler.RestSpend)
	api.Get("/v1/rest-get", restHandler.RestGet) // Remove in future
	api.Get("/v1/rest/get", restHandler.RestGet)
	// Manage
	api.Post("/v1/manage/task/create", manageHandler.CreateTask)

	// Review

	// Routes

	// Statistics
	api.Get("/v1/stats/done/today", statsHandler.StatCompletionTimeDone) // TODO Remove in future
	// Alias for dashboard tasks list (today planned vs done)
	api.Get("/v1/stats/tasks/today", statsHandler.StatCompletionTimeDone)
	api.Get("/v1/stats/weekly", statsHandler.GetWeeklyStats)
	api.Get("/v1/tasklist", statsHandler.ShowTaskList) // Legacy endpoint for CLI and web UI

	// Plan Percents
	api.Get("/v1/manage/plan-percents", manageHandler.GetPlanPercents)                    // New route for plan percents
	api.Delete("/v1/manage/plan-percents/:group/:value", manageHandler.DeletePlanPercent) // Remove specific plan percent

	// General

	api.Post("/v1/timer/set", manageHandlerOld.TimerSet)
	api.Get("/v1/timer/get", manageHandlerOld.TimerGet)
	api.Post("/v1/timer/del", manageHandlerOld.TimerDel)

	// Legacy record routes for web UI and CLI
	api.Get("/v1/records", taskRecordHandler.ShowRecords)
	api.Get("/v1/records/clean", taskRecordHandler.CleanRecords)

	// Roles
	api.Get("/v1/roles/records", roleHandler.ShowRolesRecords)
	api.Get("/v1/roles/records/today", roleHandler.StatCompletionTimeDone) // Change function in future
	api.Get("/v1/role/recheck", roleHandler.RecheckRole)
	api.Get("/v1/role/get", roleHandler.TaskRoleGet)

	//Manage
	api.Post("/v1/manage/procents", manageHandlerOld.ProcentsSet)
	api.Get("/v1/manage/procents", manageHandlerOld.GetPlanProcents)
	api.Get("/v1/manage/timer/recheck", manageHandlerOld.TimerRecheck)
	api.Post("/v1/manage/timer/global", manageHandlerOld.TimerGlobalSet)
	api.Get("/v1/manage/timer/global", manageHandlerOld.TimerGlobalGet)
	api.Post("/v1/manage/telegram/start", manageHandlerOld.TelegramSendStart)
	api.Post("/v1/manage/telegram/stop", manageHandlerOld.TelegramSendStop)
	api.Post("/v1/manage/telegram/message", manageHandlerOld.TelegramSendCustom)

	// Schedule
	api.Post("/v1/schedule", scheduleHandler.CreateSchedule)
	api.Get("/v1/schedule/active", scheduleHandler.GetActiveSchedule)
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
	api.Get("/v1/timer/run/status", runningTaskHandler.Status)
	api.Get("/v1/timer/run/list", runningTaskHandler.List)

	// Evening Mode
	api.Get("/v1/mode/evening-focus", eveningHandler.GetEveningFocus)
	api.Post("/v1/mode/evening-focus/skip", eveningHandler.SkipTask)

	app.Get("/", welcome.Welcome)
}
