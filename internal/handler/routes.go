package handler

import (
	"tracker-server/internal/handler/manage"
	"tracker-server/internal/handler/record"
	"tracker-server/internal/handler/rest"
	"tracker-server/internal/handler/role"
	"tracker-server/internal/handler/statistic"
	"tracker-server/internal/handler/welcome"
	"tracker-server/internal/notify"
	"tracker-server/internal/storage"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, mongoconn storage.Storage, notify notify.Notify) {

	restHandler := rest.New(mongoconn, notify)
	recordHandler := record.New(mongoconn, notify)
	roleHandler := role.New(mongoconn, notify)
	manageHandler := manage.New(mongoconn, notify)
	statsHandler := statistic.New(mongoconn)

	api := app.Group("/api")

	// Rest
	api.Post("/v1/rest/add", restHandler.RestAdd)
	// Statistics
	api.Get("/v1/stats/done", statsHandler.StatCompletionTimeDone)

	// General

	api.Post("/v1/rest-spend", restHandler.RestSpend)

	api.Get("/v1/rest-get", restHandler.RestGet)
	api.Post("/v1/timer/set", manageHandler.TimerSet)
	api.Get("/v1/timer/get", manageHandler.TimerGet)
	api.Post("/v1/timer/del", recordHandler.TimerDel)

	// Records
	api.Post("/v1/record", recordHandler.AddRecord)
	api.Get("/v1/record/task-day", recordHandler.GetDayTaskRecord)
	api.Post("/v1/record/params", recordHandler.SetTaskParams)
	api.Get("/v1/record/params", recordHandler.GetTaskParams)
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
	api.Post("/v1/manage/procents", manageHandler.ProcentsSet)
	api.Get("/v1/manage/timer/recheck", manageHandler.TimerRecheck)
	api.Post("/v1/manage/timer/global", manageHandler.TimerGlobalSet)
	api.Get("/v1/manage/timer/global", manageHandler.TimerGlobalGet)
	api.Post("/v1/manage/telegram/start", manageHandler.TelegramSendStart)
	api.Post("/v1/manage/telegram/stop", manageHandler.TelegramSendStop)

	app.Get("/", welcome.Welcome)
}
