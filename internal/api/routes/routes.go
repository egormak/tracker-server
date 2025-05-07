package routes

import (
	"tracker-server/internal/api/handler"
	"tracker-server/internal/handler/manage"
	"tracker-server/internal/handler/record"
	"tracker-server/internal/handler/role"
	"tracker-server/internal/handler/welcome"
	"tracker-server/internal/notify"
	"tracker-server/internal/services"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, mongoconn storage.Storage, notify notify.Notify) {

	// Services
	taskService := services.NewTaskService(mongoconn, notify)
	taskRecordService := services.NewTaskRecordService(mongoconn)
	restService := services.NewRestService(mongoconn)
	statsService := services.NewStatisticService(mongoconn)
	manageService := services.NewManageService(mongoconn)

	// Handlers
	// Task
	taskHandler := handler.NewTaskHandler(taskService)
	// TaskRecords
	taskRecordHandler := handler.NewTaskRecordHandler(taskRecordService)
	// Rest
	restHandler := handler.NewRestHandler(restService)
	// Statistics
	statsHandler := handler.NewStatisticHandler(statsService)
	// Manage
	manageHandler := handler.NewManageHandler(manageService)

	// OLD Logic
	recordHandler := record.New(mongoconn, notify)
	roleHandler := role.New(mongoconn, notify)
	manageHandlerOld := manage.New(mongoconn, notify)
	//

	// Routes
	api := app.Group("/api")
	// Task
	api.Get("/v1/task/params", taskHandler.TaskParams)        // TODO
	api.Get("/v1/record/params", recordHandler.GetTaskParams) // Remove in future Task Params
	// api.Post("/v1/task/create", taskHandler.CreateTask)
	// TaskRecords
	api.Post("/v1/taskrecord", taskRecordHandler.AddRecord)
	// api.Get("/v1/task/next", taskRecordHandler.TasksNext)
	api.Get("/v1/task/plan/percent", taskRecordHandler.GetTaskPlanPercent)
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
	api.Get("/v1/stats/done/today", statsHandler.StatCompletionTimeDone)

	// General

	api.Post("/v1/timer/set", manageHandlerOld.TimerSet)
	api.Get("/v1/timer/get", manageHandlerOld.TimerGet)
	api.Post("/v1/timer/del", recordHandler.TimerDel)

	// Records
	api.Post("/v1/record", recordHandler.AddRecord)
	api.Get("/v1/record/task-day", recordHandler.GetDayTaskRecord)
	api.Post("/v1/record/params", recordHandler.SetTaskParams)

	api.Get("/v1/records", recordHandler.ShowRecords)
	api.Get("/v1/records/clean", recordHandler.CleanRecords)
	api.Get("/v1/tasklist", recordHandler.ShowTaskList)
	api.Get("/v1/task/plan-percent", recordHandler.GetTaskPlanPercent)
	api.Get("/v1/task/percent", recordHandler.GetTaskDayByPercent)
	// api.Post("/v1/task/schedule", recordHandler.SetTaskSchedule)
	api.Get("/v1/task/plan-percent/change", recordHandler.ChangeGroupPlanPercent)

	// Roles
	api.Get("/v1/roles/records", roleHandler.ShowRolesRecords)
	api.Get("/v1/roles/records/today", roleHandler.StatCompletionTimeDone) // Change function in future
	api.Get("/v1/role/recheck", roleHandler.RecheckRole)
	api.Get("/v1/role/get", roleHandler.TaskRoleGet)

	//Manage
	api.Post("/v1/manage/procents", manageHandlerOld.ProcentsSet)
	api.Get("/v1/manage/timer/recheck", manageHandlerOld.TimerRecheck)
	api.Post("/v1/manage/timer/global", manageHandlerOld.TimerGlobalSet)
	api.Get("/v1/manage/timer/global", manageHandlerOld.TimerGlobalGet)
	api.Post("/v1/manage/telegram/start", manageHandlerOld.TelegramSendStart)
	api.Post("/v1/manage/telegram/stop", manageHandlerOld.TelegramSendStop)

	app.Get("/", welcome.Welcome)
}
