# Repository Guidelines

## Project Structure & Module Organization
- **Entry point**: `cmd/server/main.go`
  - Loads `./config.yaml`, initializes structured logging (`slog` with `tint` handler).
  - Connects to MongoDB, calls `EnsureIndexes` on startup.
  - Initializes `telegram.Telegram` notifier and `realtime.Hub` (starts WebSocket broadcast loop).
  - Calls `routes.RegisterRoutes` to configure middleware, handlers, and endpoints.
  - Starts two background workers returned by route registration:
    - `TimerWatchdog` (`services/timer_watchdog.go`)
    - `DayRollService` (`services/day_roll_service.go`)
  - Starts Fiber HTTP server on `:3000`.
- **Core packages under `internal/`**:
  - `api/handler/` – The only HTTP handler layer. Old legacy handlers from `internal/handler/` have been completely folded in here. Handlers parse/validate requests, invoke services, and return JSON responses.
  - `api/routes/` – Central route definition and manual dependency injection hub. Handlers and services are instantiated and wired here.
  - `api/middleware/` – `TelegramAuth` middleware. When `telegram.enable_webapp_auth` is true, validates requests against `X-Bot-Token` or HMAC-signed `X-Telegram-Init-Data` where the Telegram user ID equals `telegram.room_id`.
  - `realtime/` – WebSocket hub (`realtime.Hub`) handling client connections and broadcasting timer events (`GET /api/v1/timer/ws`).
  - `services/` – Business logic domain services:
    - `running_task_service.go`: Live timer state machine (start, pause, resume, stop, adjust, heartbeat), single active running task enforced by mutex lock (`sync.Mutex`), WebSocket event emission, interactive Telegram notifications, and forward overtime credit.
    - `schedule_service.go`: Weekly schedule CRUD, active schedule tracking, deficit/rollover calculation, `ApplyScheduleToday`, and thread-safe idempotent `EnsureTodaySchedule`.
    - `statistic_service.go`: Today's completion stats, weekly stats, tasklist. Injects `TodayScheduleEnsurer` (`ScheduleService`) to auto-apply today's schedule if the task list is empty.
    - `taskrecord_service.go`: 3-step storage persistence (`AddTaskRecord` → `AddRoleMinutes` → `AddRest`) followed by best-effort `RampService.AutoRecalculateOnRecord`, backfill logic (`ManageByService` / `source_day`), and strict task overtime credit to tomorrow.
    - `ramp_service.go`: Focus warm-up step calculation $k = \min(\text{cap}, \lfloor(1+\sqrt{1+8S})/2\rfloor)$ from day's focus minutes, persisted in `"Ramp Info"` singleton, lazy reset on new day.
    - `evening_service.go`: Catch-up task ranking by weekly deficit (excluding `work` and `english`), with in-memory thread-safe `snoozedTonight` set reset daily.
    - `day_roll_service.go`: Background worker (runs every 30s) calling `EnsureTodaySchedule` on server start and automatically when the calendar day changes.
    - `timer_watchdog.go`: Background worker (runs every 10s) enforcing target duration completion auto-stop, 20-minute safety cap for untargeted timers, and 3-minute heartbeat lease auto-pause.
    - `rest_service.go`: Rest balance tracking in `"Rest Info"` singleton document.
    - `plan_percent.go`: Percent-weighted task group rotations.
    - `days.go`: Date manipulation helpers (`CalculateDateForDay`, `DayIndex`, `IsWeekendNow`, `FormatDate`).
  - `storage/` + `storage/mongo/` – `Storage` interface contract (`internal/storage/storage.go`) and MongoDB implementation (`mongo/`). Collections: `task_info`, `tasks`, `task_list`, `role_info`, `weekly_schedules`. Special singletons: `"Rest Info"`, `"Procent Info"`, `"Ramp Info"`, `"Scheduler"`, `"Day List"`.
  - `domain/entity/`, `models/`, `options/` – Typed domain models.
  - `notify/` + `notify/telegram/` – Notification interface (`SendMessageStart`, `SendMessageStop`, `SendMessageCompletion`, `SendCustomMessage`) with Telegram Bot API implementation supporting HTML formatting and inline action keyboards.
- **Configuration**: `config/config.go` reads `./config.yaml` relative to CWD (see `config_example.yaml`).
- **Documentation**: `openapi.yml` (authoritative API contract) + `docs/SCHEDULE_*.md`.
- **Frontend**: **No web frontend lives in this repository** (it was extracted to a separate `tracker-web` repo). Do not create or expect a `web/` directory here.
- **Docker Assets**: `Dockerfile` (multi-stage build using Go 1.25.1-alpine builder and Alpine 3.21 runtime) and `docker-compose.yml` (orchestrating `mongo:6` and `api`).

## Build, Test, and Development Commands
- **Primary (Makefile)**:
  - `make help` – List available targets.
  - `make run` – Run server locally on `:3000` (requires local `./config.yaml` and reachable MongoDB).
  - `make build` – Build binary to `bin/server`.
  - `make test` – Run all unit tests (`go test ./...`).
  - `make fmt` / `make vet` / `make tidy` – Format code (`go fmt`), analyze (`go vet`), and tidy modules (`go mod tidy`).
  - `make docker-build TAG=...` – Build Docker image `ghcr.io/egormak/tracker-server:TAG`.
  - `make docker-run TAG=...` – Run backend image mapping `3000` and mounting `./config.yaml`.
  - `make docker-prod` – Run as named container `tracker` in background (mapping `8080->3000`).
  - `make docker-stop` – Stop and remove `tracker` container.
  - `make compose-up` – Start API + MongoDB stack via docker-compose (API exposed on `:3000`, MongoDB on `:27017`).
  - `make compose-down` – Stop docker-compose stack and remove volumes (`-v`).
  - `make compose-logs` – Tail docker-compose logs (`--tail=200`).
  - `make all` – Run fmt, vet, and build.
  - `make clean` – Remove `bin/` artifacts.
- **Direct**:
  - `go run cmd/server/main.go`
  - `go build -o bin/server ./cmd/server`
  - `go test ./internal/services/ -run TestName -v`

## Testing Guidelines
- Unit tests are located across `internal/services/`, `internal/api/handler/`, `internal/notify/telegram/`, `internal/domain/entity/`, and `internal/storage/mongo/`.
- **No MongoDB required for unit tests**: Services and handlers are tested using pure hand-written mock structs satisfying narrow storage interfaces (e.g. `mockRampStorage`, `mockRestStorage`).
- Date-sensitive tests: Tests involving backfill or rollover calculation call `t.Skip` on Mondays since there are no earlier weekdays in the current week to backfill.
- Run all tests before committing: `make test` (or `go test ./...`).
- `test/main.go` is an ad-hoc manual script that requires MongoDB; it is not run by `go test`.

## Coding Style & Naming Conventions
- Go 1.25.1. Always run `make fmt` and `make vet` before committing.
- Tabs for indentation, idiomatic Go.
- Naming:
  - Package names: `lower_snakecase`.
  - File names: `feature_action.go` (e.g. `taskrecord_service.go`, `running_task_handler.go`).
  - Exported identifiers: `PascalCase`; unexported: `lowerCamelCase`.
- Clean Architecture boundaries: Handlers (`internal/api/handler`) → Services (`internal/services`) → Storage interfaces (`internal/storage/storage.go` and narrow service interfaces) → MongoDB adapter (`internal/storage/mongo`).
- Date representation: Format strings use `"2 January 2006"` for day-level records; do not switch to RFC3339 for database record dates without updating all queries and indexes.
- Logging: Use standard library `log/slog` (with `tint` handler) for all new code. `logrus` is legacy and only retained in `cmd/server/main.go`.
- Error handling:
  - Wrap errors with descriptive context: `fmt.Errorf("action context: %w", err)`.
  - HTTP Handlers return JSON `{status: "error", message: "..."}` with HTTP 500 status code for internal failures (standard across the codebase).

## Security & Configuration Tips
- Never commit secrets. Provide `./config.yaml` based on `config_example.yaml`:
  - `mongodb`: `host`, `port`, `name` (use `host: mongo` inside docker-compose).
  - `telegram`: `api_key`, `room_id` (numeric `int64`), `enable_webapp_auth` (boolean).
- If `telegram.enable_webapp_auth: true`, incoming requests to `/api/*` must pass Telegram authentication via `X-Bot-Token` or `X-Telegram-Init-Data`.

## Key Business Logic Rules
- **Plan-Percent Auto-Rotation**: `GetTaskPlanPercent` automatically loops through and rotates to the next group when current percentage allocations are exhausted.
- **Record Multi-Step Persistence**: Recording time via `TaskRecordService.AddRecord` runs `AddTaskRecord` → `AddRoleMinutes` → `AddRest` sequentially, followed by a best-effort `RampService.AutoRecalculateOnRecord`.
- **Source Day & Backfill**: When `ManageByService` is active and `source_day` is specified or implied, the service walks Monday through yesterday against the active schedule to backfill any deficit before applying remaining time to today.
- **Dynamic Forward Overtime Credit**: For tasks marked `TimeStrictly: true`, time completed exceeding the scheduled daily target is credited to tomorrow as a separate task record.
- **Running Task Mutex & Single Active Timer**: `RunningTaskService` serializes all timer modifications with `sync.Mutex`. Starting or resuming a task automatically pauses any currently running task.
- **Timer Watchdog**: Runs every 10 seconds:
  1. Auto-stops running tasks when `TargetDuration` is reached.
  2. Auto-stops free stopwatch timers (`TargetDuration == 0`) when reaching the 20-minute safety cap.
  3. Auto-pauses running tasks if clients stop sending heartbeats for longer than 3 minutes (`LastHeartbeatAt`).
- **Day-Roll Worker & Schedule Auto-Apply**: Runs every 30 seconds. On application startup or when crossing the midnight date boundary, `EnsureTodaySchedule()` idempotently applies today's schedule from the active weekly schedule if tasks have not yet been populated.
- **Ramp Warm-up Progression**: Daily focus warm-up steps are calculated as $k = \min(\text{cap}, \lfloor(1+\sqrt{1+8S})/2\rfloor)$ from cumulative focus minutes $S$, resetting lazily on the first read of a new calendar day.
- **Evening Catch-up Mode**: Suggests tasks ordered by weekly deficit, excluding `work` and `english`, with in-memory snoozing cleared daily.
- **Realtime WebSocket Hub**: `RunningTaskService` broadcasts timer lifecycle events (`TASK_STARTED`, `TASK_PAUSED`, `TASK_STOPPED`, `STATE_SYNC`, etc.) to connected clients via `GET /api/v1/timer/ws`.

## API Highlights
Refer to `openapi.yml` for the complete OpenAPI 3.0 specification.
- **Dashboard**: `GET /api/v1/dashboard/state`
- **Tasks & Records**:
  - `GET /api/v1/task/params`, `POST /api/v1/task/params`
  - `POST /api/v1/taskrecord` – Add record `{ task_name, time_done, source_day? }`
  - `GET /api/v1/task/plan/percent` – Plan-based next task
  - `GET /api/v1/task/plan/percent/schedule` – Schedule-aware next task
  - `POST /api/v1/task/plan/rotate` – Rotate current plan group
- **Running Task Timer & WebSocket**:
  - `POST /api/v1/timer/run/start`, `POST /api/v1/timer/run/stop`, `POST /api/v1/timer/run/pause`, `POST /api/v1/timer/run/resume`
  - `POST /api/v1/timer/run/adjust` – Adjust elapsed minutes
  - `POST /api/v1/timer/run/heartbeat` – Renew timer lease (required every < 3 min)
  - `GET /api/v1/timer/run/status`, `GET /api/v1/timer/run/list`
  - `GET /api/v1/timer/ws` – WebSocket real-time event subscription
- **Ramp Warm-Up**:
  - `GET /api/v1/ramp/status`, `POST /api/v1/ramp/reset`, `POST /api/v1/ramp/advance`
  - `GET /api/v1/ramp/config`, `PUT /api/v1/ramp/config`
- **Schedule**:
  - `POST /api/v1/schedule` – Create schedule (optional `set_active`)
  - `GET /api/v1/schedule/active`, `GET /api/v1/schedule/active/today`, `GET /api/v1/schedule/active/rollover`
  - `PATCH /api/v1/schedule/active/task-time` – Modify scheduled minutes for a task
  - `POST /api/v1/schedule/apply` – Manually apply today's schedule
  - `GET/PUT/DELETE /api/v1/schedule/:id`, `PUT /api/v1/schedule/:id/activate`
- **Evening Focus Mode**:
  - `GET /api/v1/mode/evening-focus` – Get prioritized catch-up task
  - `POST /api/v1/mode/evening-focus/skip` – Temporarily snooze task for tonight
- **Rest & Plan Percents**:
  - `GET /api/v1/rest/get`, `POST /api/v1/rest/add`, `POST /api/v1/rest/spend`, `POST /api/v1/rest/reset`
  - `GET /api/v1/manage/plan-percents`, `DELETE /api/v1/manage/plan-percents/:group/:value`
- **Statistics**:
  - `GET /api/v1/stats/done/today`, `GET /api/v1/stats/tasks/today`, `GET /api/v1/stats/weekly`
  - `GET /api/v1/tasklist`
