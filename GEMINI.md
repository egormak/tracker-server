# tracker-server

The central REST API hub for the tracker system. Built with Go and Fiber, using MongoDB for persistence.

## Project Overview

- **Core Technologies**: Go 1.25+, Fiber framework, MongoDB 5.0+, `slog` (structured logging with `tint` handler).
- **Role**: Source of truth and sole persistent storage component. No web frontend code lives in this repo (the UI resides in `tracker-web`).
- **Architecture**:
  - `cmd/server/main.go`: Entry point. Loads config, connects to MongoDB (`EnsureIndexes`), initializes Telegram notifier and `realtime.Hub`, registers routes, and starts background workers (`TimerWatchdog` and `DayRollService`).
  - `internal/api/routes/routes.go`: Central route mapping and dependency injection hub. Handlers and services are instantiated and wired here.
  - `internal/api/middleware/telegram.go`: Authentication middleware (`TelegramAuth`) supporting `X-Bot-Token` or HMAC-signed `X-Telegram-Init-Data` validation against `telegram.room_id`.
  - `internal/api/handler/`: Consolidated HTTP handler layer (the legacy `internal/handler/` package has been completely eliminated and folded here). Handlers parse/validate requests, invoke services, and return JSON responses.
  - `internal/realtime/hub.go`: WebSocket broadcast hub (`GET /api/v1/timer/ws`) broadcasting timer state changes to connected clients.
  - `internal/services/`: Business logic domain services (tasks, taskrecords, plan, rest, schedule, running_task, evening, ramp, day_roll, timer_watchdog).
  - `internal/storage/`: Shared storage contract (`internal/storage/storage.go`) and MongoDB adapter (`internal/storage/mongo/`).
  - `internal/domain/entity/`: Typed domain entities.
  - `internal/notify/`: Telegram notification service with interactive inline action keyboards.

## API Endpoints

The API is versioned under `/api/v1/`. Key route groups:
- **Dashboard**: `GET /api/v1/dashboard/state`
- **Tasks & Records**: `/api/v1/task/...`, `/api/v1/taskrecord`, `/api/v1/records/...`
- **Planning**: `/api/v1/task/plan/percent`, `/api/v1/task/plan/percent/schedule`, `/api/v1/task/plan/rotate`, `/api/v1/manage/plan-percents/...`, `/api/v1/manage/procents`
- **Rest**: `/api/v1/rest/get`, `/api/v1/rest/add`, `/api/v1/rest/spend`, `/api/v1/rest/reset`
- **Ramp Warm-up**: `/api/v1/ramp/status`, `/api/v1/ramp/reset`, `/api/v1/ramp/advance`, `/api/v1/ramp/config`
- **Schedule**: `/api/v1/schedule/...` (CRUD, active, rollover, today, apply, task-time adjustment)
- **Statistics**: `/api/v1/stats/done/today`, `/api/v1/stats/tasks/today`, `/api/v1/stats/weekly`, `/api/v1/tasklist`
- **Running Task Timer & WebSocket**: `/api/v1/timer/run/...` (start, stop, pause, resume, adjust, status, list, heartbeat) and `/api/v1/timer/ws`
- **Evening Focus Mode**: `/api/v1/mode/evening-focus`, `/api/v1/mode/evening-focus/skip`
- **Roles & Legacy Timers**: `/api/v1/roles/...`, `/api/v1/timer/get`, `/api/v1/timer/set`, `/api/v1/timer/del`, `/api/v1/manage/timer/global`

Refer to `openapi.yml` for the definitive OpenAPI 3.0 specification.

## Core Business Logic Rules

- **Plan-Percent Auto-Rotation**: `GetTaskPlanPercent` automatically loops through and rotates to the next task group when current group percentages are depleted.
- **Record Multi-Step Persistence**: Recording time via `TaskRecordService.AddRecord` executes `AddTaskRecord` → `AddRoleMinutes` → `AddRest` sequentially, followed by a best-effort `RampService.AutoRecalculateOnRecord`.
- **Ramp Warm-Up**: Daily focus warm-up steps are calculated as $k = \min(\text{cap}, \lfloor(1+\sqrt{1+8S})/2\rfloor)$ from cumulative focus minutes $S$. State lives in `"Ramp Info"` and resets lazily on the first read of a new calendar day.
- **Source Day & Backfill**: Records can target past weekdays (`source_day`). When `ManageByService` is active, backfill checks Monday→yesterday against the active schedule to fill shortfalls before applying remaining time to today.
- **Dynamic Forward Overtime Credit**: For strict tasks (`TimeStrictly: true`), completed minutes past the scheduled daily target are automatically credited to tomorrow (`source_day = tomorrow`).
- **Running Task Mutex**: `RunningTaskService` uses a `sync.Mutex` lock to serialize concurrent timer actions across clients. Only one task can be `IsRunning` at any time; starting/resuming a task auto-pauses any active task.
- **Timer Watchdog Worker**: Runs every 10 seconds:
  1. Auto-stops tasks when `TargetDuration` is reached.
  2. Auto-stops untargeted stopwatch timers (`TargetDuration == 0`) at a 20-minute safety cap.
  3. Auto-pauses tasks if client heartbeats stop for longer than the 3-minute lease (`LastHeartbeatAt`).
- **Day-Roll Worker & Schedule Auto-Apply**: Runs every 30 seconds. On startup and midnight date boundaries, `EnsureTodaySchedule()` idempotently checks and applies today's schedule from the active weekly schedule.
- **Schedule Fallback in Statistics**: `StatisticService.ShowTaskList` automatically triggers `EnsureTodaySchedule()` if today's task list is empty.
- **Evening Catch-Up Mode**: Micro-sprint queue targeting weekly deficit tasks while excluding `work` and `english`. In-memory snoozed tasks reset once per calendar day.
- **Realtime WebSocket Hub**: `RunningTaskService` broadcasts timer lifecycle events (`TASK_STARTED`, `TASK_PAUSED`, `TASK_STOPPED`, `STATE_SYNC`) to clients on `GET /api/v1/timer/ws`.
- **Date Format**: Standard day-level record date format across MongoDB collections is `time.Now().Format("2 January 2006")`.

## Building and Running

### Commands (Makefile)
- `make run`: Run server locally on `:3000` (requires `./config.yaml`).
- `make build`: Build binary to `bin/server`.
- `make test`: Run Go unit tests (`go test ./...`).
- `make fmt` / `make vet` / `make tidy`: Format, vet, and tidy Go module.
- `make compose-up`: Start API and MongoDB stack using Docker Compose (API on `:3000`, Mongo on `:27017`).
- `make compose-down`: Stop compose stack and remove volumes (`-v`).
- `make docker-build TAG=...`: Build Docker image `ghcr.io/egormak/tracker-server:TAG`.
- `make all`: Format, vet, and build binary.

### Configuration
Loaded from `./config.yaml` relative to CWD (see `config_example.yaml` for structure):
- `mongodb`: `host`, `port`, `name` (`tasker`).
- `telegram`: `api_key`, `room_id` (`int64`), `enable_webapp_auth` (`bool`).

## Testing Guidelines
- Unit tests live across `internal/services/`, `internal/api/handler/`, `internal/notify/telegram/`, `internal/domain/entity/`, and `internal/storage/mongo/`.
- Hand-written mock structs implement narrow service storage interfaces; unit tests do not require MongoDB.
- Date-sensitive tests skip (`t.Skip`) on Mondays if evaluating backfill or rollover from earlier weekdays.
- Run tests with `make test` or `go test ./...`.

## Development Conventions

- **Clean Architecture**: Handlers (`internal/api/handler/`) → Services (`internal/services/`) → Narrow storage interfaces → MongoDB adapter (`internal/storage/mongo/`).
- **API First**: Always update `openapi.yml` when modifying endpoint parameters or responses.
- **Logging**: Use `log/slog` (with `tint` handler) for all new code.
- **Error Responses**: Return JSON `{status: "error", message: "..."}` with standard HTTP 500 for backend errors.
