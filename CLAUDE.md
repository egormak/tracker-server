# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tracker-server is a Go/Fiber REST API for personal time tracking, backed by MongoDB. It tracks tasks against three roles (`work`, `learn`, `rest`), supports a percentage-based task-rotation "plan", a weekly schedule with rollover/backfill logic, and live running-task timers, with optional Telegram notifications. There is **no web frontend in this repo** — it was removed (see commit `99aee1b`/`7514f8c`); the frontend lives in a separate `tracker-web` repo.

## Build, Test, and Run

- `make run` — run the server locally on `:3000` (requires `./config.yaml` in repo root; MongoDB must be reachable).
- `make build` — build binary to `bin/server`.
- `make test` — `go test ./...`. Tests live in `internal/services`, `internal/api/handler`, `internal/storage/mongo`, `internal/notify/telegram`, and `internal/domain/entity`. None need MongoDB: services/handlers use hand-written mock structs that implement the narrow storage interfaces (e.g. `mockRampStorage`, `mockRestStorage`), and the `mongo` package tests cover only pure helpers. Some date-dependent tests `t.Skip` on Mondays, since there are no earlier weekdays to backfill or roll over.
- `go test ./internal/services/ -run TestName -v` — run a single test.
- `test/main.go` is an ad-hoc manual script that hits real MongoDB via `config.yaml`. It is not part of the test suite.
- `make fmt` / `make vet` / `make tidy` — `go fmt`, `go vet`, `go mod tidy`.
- `make docker-build TAG=...` / `make docker-run TAG=...` / `make docker-prod` / `make docker-stop`.
- `make compose-up` / `make compose-down` / `make compose-logs` — run API + MongoDB via docker-compose (API on `:3000`).
- `make all` — fmt + vet + build.
- CI (`.github/workflows/github-actions-demo.yml`) builds a Docker image on every push or PR to `main` and pushes it to `ghcr.io/egormak/tracker-server` tagged `:YYYY-MM-DD` and `:latest`. CI does not run tests, so run `make test` locally. Helm charts are in `helm/`.

## Architecture

Request flow: **Handler → Service → Storage interface → MongoDB**.

- `cmd/server/main.go` — entry point. Loads config, connects to MongoDB (`EnsureIndexes` on startup), constructs the Telegram notifier and the `realtime.Hub`, calls `routes.RegisterRoutes`, then starts two background workers using the services that `RegisterRoutes` returns: `TimerWatchdog` and `DayRollService`.
- `internal/api/routes/routes.go` — the single place all routes are wired up. Services and handlers are constructed here and injected; there's no DI framework. When adding an endpoint, this is where handler/service instantiation and route registration both happen. If a new service needs a background loop, return it from `RegisterRoutes` and start it in `main.go`.
- `internal/api/middleware/telegram.go` — `TelegramAuth` middleware wraps the entire `/api` group. If `telegram.enable_webapp_auth` is true in config, every request must carry either `X-Bot-Token` (matching `telegram.api_key`) or a valid signed `X-Telegram-Init-Data` header (HMAC-verified, and the embedded Telegram user ID must equal `telegram.room_id`). When the flag is false (the default), auth is bypassed entirely.
- `internal/api/handler/` — the only HTTP handler layer. The old `internal/handler/` package has been folded in here. Handlers parse and validate the request, call a service, and return JSON `{status, message/data}`. Most handlers go through a service. A few legacy ones still take `storage.Storage` directly and skip the service layer: `role_handler.go`, `legacy_timer_handler.go` (the old countdown/global timers under `/v1/timer/{set,get,del}` and `/v1/manage/timer/global`, which are unrelated to running tasks), and `telegram_handler.go` (which takes `notify`). Route new endpoints through a service. Several routes are kept only for older clients; they're commented `// Legacy endpoint for CLI` in `routes.go`.
- `internal/realtime/hub.go` — WebSocket broadcast hub (`GET /api/v1/timer/ws`). `RunningTaskService` pushes typed events to it (`TASK_STARTED`, `TASK_PAUSED`, `TASK_STOPPED`, `STATE_SYNC`, …) through `broadcastEvent`, which does nothing when the hub is nil. That lets tests build the service without a hub.
- `internal/services/` — business logic. Each service declares its own narrow storage interface (e.g. `TaskRecordStorage`, `RunningTaskStorage`, `ScheduleStorage`) naming only the methods it needs, even though one concrete `*mongo.Storage` satisfies all of them. `services/days.go` holds shared date helpers (`CalculateDateForDay`, `DayIndex`, `IsWeekendNow`) used across taskrecord/schedule/running-task logic.
- `internal/storage/storage.go` — the full `Storage` interface that `internal/storage/mongo` implements; this is the contract new storage methods must be added to (e.g. `HasTasksForDate`, `TimerGlobalSet(timeScheduler int, date ...string)`).
- `internal/storage/mongo/` — MongoDB adapter, one file per concern (`task.go`, `task_record.go`, `rest.go`, `role.go`, `procents.go`, `schedule.go`, `running_task.go`, `statistic.go`, `timer.go`). `mongo.go` defines shared constants: DB `tasker`; collections `task_info`, `tasks`, `task_list`, `role_info`; special singleton documents `"Rest Info"`, `"Procent Info"`, and `"Ramp Info"`. It also defines `EnsureIndexes`, so add any new index there.
- `internal/domain/entity/` — typed domain models (task, rest, role, manage, schedule, running_task).
- `internal/notify/` — `Notify` interface (`SendMessageStart`, `SendMessageStop`, `SendMessageCompletion`, `SendCustomMessage`) with a Telegram implementation in `notify/telegram/` supporting rich HTML and inline action keyboards (pause, stop, switch, extend, rest). Passed into services that need to message the user (running-task start/stop/completion).
- `config/config.go` — loads `./config.yaml` (relative to CWD) into `Config` at startup; process exits on missing file or bad YAML. See `config_example.yaml` for shape: `mongodb.{host,port,name}`, `telegram.{api_key,room_id,enable_webapp_auth}`.
- `openapi.yml` — authoritative API contract; update it when endpoints change.

### Key business logic to know before changing behavior

- **Plan-percent rotation** (`services/plan_percent.go`, `taskrecord_service.go`): tasks are organized into percent-weighted groups; `GetTaskPlanPercent` auto-rotates to the next group when the current group's percent list is exhausted (loop inside the method, not the caller).
- **Task records & rest** (`taskrecord_service.go`): recording time against a task is a 3-step storage sequence — `AddTaskRecord`, `AddRoleMinutes`, `AddRest` — done in that order; a partial failure leaves inconsistent state, so don't reorder without considering that. After those succeed, `RampService.AutoRecalculateOnRecord` runs as a best-effort step: if it fails, the error is logged and not returned.
- **Ramp warm-up** (`services/ramp_service.go`, `/api/v1/ramp/*`): a daily "warm-up" step counter. `CalculateStep` computes k = min(cap, ⌊(1+√(1+8S))/2⌋) from S, the day's focus minutes. State lives in the `"Ramp Info"` singleton. It resets lazily: the first read on a new calendar day (`getOrResetRamp`) resets it, with no scheduled job.
- **Background workers** (started in `main.go`, both stop on ctx cancel or `Stop()`):
  - `TimerWatchdog` (`services/timer_watchdog.go`, checks every 10s) enforces three rules on the running task. It auto-stops the task when it reaches its `TargetDuration`. It auto-stops a free stopwatch (`TargetDuration == 0`) at a 20-minute safety cap. It pauses the task if clients stop sending `POST /v1/timer/run/heartbeat` for longer than the 3-minute lease (`LastHeartbeatAt`). Clients that run a timer must send heartbeats, or the task gets paused.
  - `DayRollService` (`services/day_roll_service.go`, checks every 30s) calls `ScheduleService.EnsureTodaySchedule` at startup. When the calendar date changes, it re-applies the active schedule. It advances `lastDate` only if applying the schedule succeeded. It exposes `SetNowFunc`/`SetInterval` for tests. In addition, `StatisticService.ShowTaskList` / `GetTaskRecordToday` has a lazy fallback that calls `EnsureTodaySchedule` if today's task list is empty.
- **Source-day / backfill** (`taskrecord_service.AddRecord`, `services/days.go`): a record can target a specific past weekday (`source_day`) rather than today via `CalculateDateForDay`. When `ManageByService` is set, the service walks Monday→yesterday against the active weekly schedule and backfills any shortfall for that task before applying remaining time to today.
- **Running tasks** (`services/running_task_service.go`): supports multiple concurrently-tracked tasks, but only one can be `IsRunning` at a time — starting/resuming one pauses whichever other task was active, accumulating its elapsed minutes. All mutating methods hold `RunningTaskService.mu` (a `sync.Mutex`) to serialize concurrent start/stop/pause/resume calls. `Stop` computes the record date from the task's `SourceDay` if set, otherwise today.
- **Weekly schedule / rollover** (`services/schedule_service.go`): `WeeklySchedule` has one `DaySchedule` per weekday; rollover/deficit calculations use a fixed Monday=0..Sunday=6 `dayOrder` map distinct from `days.go`'s `DayIndex` (same semantics, kept separately — check both if you change day-ordering logic).
- **Strict tasks & overtime credit** (`taskrecord_service.AddRecord`, `running_task_service.Stop`, `storage.IsTaskStrict`): a task with `TimeStrictly: true` is capped at its scheduled target time for today (`GetScheduledTargetTime`); any time recorded (or accumulated on a running-task stop) beyond that cap is *not* applied to today — it's split off into a separate `TaskRecord` dated tomorrow (`GetTomorrowDayName`/`CalculateDateForTomorrow`) via a second `AddTaskRecord`+`AddRoleMinutes` call. This only triggers when `SourceDay` is empty (i.e. not already a backfill record), avoiding double-shifting.
- **Evening focus mode** (`services/evening_service.go`, `GET/POST /api/v1/mode/evening-focus[/skip]`): ranks non-`work`/`english` tasks by weekly deficit (from `StatisticService.GetWeeklyStats`) to suggest a catch-up task. `EveningService` holds in-memory state (`snoozedTonight`, guarded by its own `mu`) that is *not* persisted to MongoDB and resets once per calendar day (`checkDailyResetLocked`, keyed off `nowFunc`) — a server restart also clears it. `SkipTask` snoozes a task (by lowercased name) until the next day-boundary reset.
- Date format used consistently for records: `time.Now().Format("2 January 2006")` — not RFC3339. Don't introduce a different format without updating all read/write sites.

## Coding Conventions

- Go 1.25.1 (per `go.mod`; `Dockerfile`'s builder image is currently pinned to match — re-check both if you bump either).
- `gofmt`/`go vet` before committing. Tabs, idiomatic Go.
- Files: `feature_action.go` naming (e.g. `taskrecord_service.go`, `running_task_handler.go`). Packages lower_snakecase.
- Exported PascalCase, unexported lowerCamelCase.
- Logging: `slog` (with `tint` handler) for all new code. `logrus` only remains in `main.go` for legacy reasons — don't spread it further.
- Errors: wrap with context via `fmt.Errorf("context: %w", err)`.
- Handlers return HTTP 500 for all error cases (not fully RESTful, but consistent with existing handlers — match this unless deliberately changing the convention project-wide).
- When adding a service method, extend that service's own narrow storage interface (not the global `storage.Storage`) unless the method is genuinely storage-wide.
