# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tracker-server is a Go/Fiber REST API for personal time tracking, backed by MongoDB. It tracks tasks against three roles (`work`, `learn`, `rest`), supports a percentage-based task-rotation "plan", a weekly schedule with rollover/backfill logic, and live running-task timers, with optional Telegram notifications. There is **no web frontend in this repo** — it was removed (see commit `99aee1b`/`7514f8c`); the frontend, if any, lives in a separate `tracker-web` repo. Ignore any references to a `web/` directory in older docs (`AGENTS.md`, `GEMINI.md`, `Readme.md`, `.github/copilot-instructions.md`) — they predate that removal and are stale.

## Build, Test, and Run

- `make run` — run the server locally on `:3000` (requires `./config.yaml` in repo root; MongoDB must be reachable).
- `make build` — build binary to `bin/server`.
- `make test` — `go test ./...` (only `internal/services/taskrecord_backfill_test.go` exists today).
- `go test ./internal/services/... -run TestName -v` — run a single test.
- `make fmt` / `make vet` / `make tidy` — `go fmt`, `go vet`, `go mod tidy`.
- `make docker-build TAG=...` / `make docker-run TAG=...` / `make docker-prod` / `make docker-stop`.
- `make compose-up` / `make compose-down` / `make compose-logs` — run API + MongoDB via docker-compose (API on `:3000`).
- `make all` — fmt + vet + build.

## Architecture

Request flow: **Handler → Service → Storage interface → MongoDB**.

- `cmd/server/main.go` — entry point. Loads config, connects to MongoDB, constructs the Telegram notifier, creates the Fiber app (with request logger middleware), and calls `internal/api/routes.RegisterRoutes`.
- `internal/api/routes/routes.go` — the single place all routes are wired up. Services and handlers are constructed here and injected; there's no DI framework. When adding an endpoint, this is where handler/service instantiation and route registration both happen.
- `internal/api/middleware/telegram.go` — `TelegramAuth` middleware wraps the entire `/api` group. If `telegram.enable_webapp_auth` is true in config, every request must carry either `X-Bot-Token` (matching `telegram.api_key`) or a valid signed `X-Telegram-Init-Data` header (HMAC-verified, and the embedded Telegram user ID must equal `telegram.room_id`). When the flag is false (the default), auth is bypassed entirely.
- `internal/api/handler/` — **current** HTTP handler layer (task, taskrecord, rest, statistic, manage, schedule, running_task). Parse/validate request, call service, return JSON `{status, message/data}`. Prefer this layer for new endpoints.
- `internal/handler/` — **legacy** Fiber handlers (`manage/`, `role/`, `welcome/`) still powering some routes (e.g. `/roles/*`, `/manage/procents`, `/manage/timer/*`, `/manage/telegram/*`, the `/` welcome route). Only touch these for backwards-compat fixes, not new features.
- `internal/services/` — business logic. Each service declares its own narrow storage interface (e.g. `TaskRecordStorage`, `RunningTaskStorage`, `ScheduleStorage`) naming only the methods it needs, even though one concrete `*mongo.Storage` satisfies all of them. `services/days.go` holds shared date helpers (`CalculateDateForDay`, `DayIndex`, `IsWeekendNow`) used across taskrecord/schedule/running-task logic.
- `internal/storage/storage.go` — the full `Storage` interface that `internal/storage/mongo` implements; this is the contract new storage methods must be added to.
- `internal/storage/mongo/` — MongoDB adapter, one file per concern (`task.go`, `task_record.go`, `rest.go`, `role.go`, `procents.go`, `schedule.go`, `running_task.go`, `statistic.go`, `timer.go`). `mongo.go` defines shared constants: DB `tasker`; collections `task_info`, `tasks`, `task_list`, `role_info`; special singleton documents `"Rest Info"` and `"Procent Info"`.
- `internal/domain/entity/` — typed domain models (task, rest, role, manage, schedule, running_task).
- `internal/notify/` — `Notify` interface (`SendMessageStart`, `SendMessageStop`, `SendCustomMessage`) with a Telegram implementation in `notify/telegram/`. Passed into services that need to message the user (running-task start/stop).
- `config/config.go` — loads `./config.yaml` (relative to CWD) into `Config` at startup; process exits on missing file or bad YAML. See `config_example.yaml` for shape: `mongodb.{host,port,name}`, `telegram.{api_key,room_id,enable_webapp_auth}`.
- `openapi.yml` — authoritative API contract; update it when endpoints change.

### Key business logic to know before changing behavior

- **Plan-percent rotation** (`services/plan_percent.go`, `taskrecord_service.go`): tasks are organized into percent-weighted groups; `GetTaskPlanPercent` auto-rotates to the next group when the current group's percent list is exhausted (loop inside the method, not the caller).
- **Task records & rest** (`taskrecord_service.go`): recording time against a task is a 3-step storage sequence — `AddTaskRecord`, `AddRoleMinutes`, `AddRest` — done in that order; a partial failure leaves inconsistent state, so don't reorder without considering that.
- **Source-day / backfill** (`taskrecord_service.AddRecord`, `services/days.go`): a record can target a specific past weekday (`source_day`) rather than today via `CalculateDateForDay`. When `ManageByService` is set, the service walks Monday→yesterday against the active weekly schedule and backfills any shortfall for that task before applying remaining time to today.
- **Running tasks** (`services/running_task_service.go`): supports multiple concurrently-tracked tasks, but only one can be `IsRunning` at a time — starting/resuming one pauses whichever other task was active, accumulating its elapsed minutes. All mutating methods hold `RunningTaskService.mu` (a `sync.Mutex`) to serialize concurrent start/stop/pause/resume calls. `Stop` computes the record date from the task's `SourceDay` if set, otherwise today.
- **Weekly schedule / rollover** (`services/schedule_service.go`): `WeeklySchedule` has one `DaySchedule` per weekday; rollover/deficit calculations use a fixed Monday=0..Sunday=6 `dayOrder` map distinct from `days.go`'s `DayIndex` (same semantics, kept separately — check both if you change day-ordering logic).
- Date format used consistently for records: `time.Now().Format("2 January 2006")` — not RFC3339. Don't introduce a different format without updating all read/write sites.

## Coding Conventions

- Go 1.25.1 (per `go.mod`; note the Dockerfile may pin a different Go version for the build image — check it isn't silently mismatched before relying on toolchain-specific behavior).
- `gofmt`/`go vet` before committing. Tabs, idiomatic Go.
- Files: `feature_action.go` naming (e.g. `taskrecord_service.go`, `running_task_handler.go`). Packages lower_snakecase.
- Exported PascalCase, unexported lowerCamelCase.
- Logging: `slog` (with `tint` handler) for all new code. `logrus` only remains in `main.go` for legacy reasons — don't spread it further.
- Errors: wrap with context via `fmt.Errorf("context: %w", err)`.
- Handlers return HTTP 500 for all error cases (not fully RESTful, but consistent with existing handlers — match this unless deliberately changing the convention project-wide).
- When adding a service method, extend that service's own narrow storage interface (not the global `storage.Storage`) unless the method is genuinely storage-wide.
