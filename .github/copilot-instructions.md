# Copilot Instructions for Tracker Server

## Architecture Overview

**Backend REST API for time tracking built with Go, Fiber, and MongoDB**

- **Entry**: `cmd/server/main.go` initializes structured logging (`slog` + `tint`), MongoDB connection (`EnsureIndexes`), Telegram notifier, WebSocket `realtime.Hub`, routes, and background workers (`TimerWatchdog` and `DayRollService`).
- **Layers**: HTTP handlers (`internal/api/handler/`) → Services (`internal/services/`) → Storage interfaces (`internal/storage/storage.go` and narrow service interfaces) → MongoDB adapter (`internal/storage/mongo/`).
- **No frontend code in repo**: The web frontend lives in a separate repository (`tracker-web`). Ignore any older references to a `web/` directory.
- **Key abstraction**: Services define their own narrow storage interfaces (e.g. `TaskRecordStorage`, `RunningTaskStorage`), satisfied by `*mongo.Storage`.
- **Notification**: `internal/notify/notify.go` interface with Telegram implementation supporting interactive inline buttons for timer actions.

### Core Domain Concepts
- **Three roles**: `work`, `learn`, `rest` (defined in `internal/storage/mongo/mongo.go`).
- **Task tracking**: Tasks have daily scheduled targets (`time_duration`), daily completion (`time_done`), and priority.
- **Plan percent system**: Tasks organized into groups (`plan`, `work`, `learn`, `rest`) with percentage allocations determining next task selection.
- **Rest balance**: Completing tasks automatically adds rest minutes (`AddRest`); rest activities deduct from the balance (`RestSpend`).
- **Ramp warm-up**: Daily warm-up counter calculating steps $k = \min(\text{cap}, \lfloor(1+\sqrt{1+8S})/2\rfloor)$ from cumulative focus minutes $S$.

## Critical Data Flow Patterns

### Adding a Task Record (POST `/api/v1/taskrecord`)
1. Handler (`taskrecord_handler.go`) validates request → Service (`taskrecord_service.go`).
2. Service fetches task role → creates `entity.TaskRecord`.
3. **Three-step storage sequence**: `AddTaskRecord()`, `AddRoleMinutes()`, `AddRest()`.
4. **Post-record hook**: `RampService.AutoRecalculateOnRecord` runs best-effort (logs errors without failing the request).
5. **Backfill & Strict tasks**: If `ManageByService` is set, backfills shortfalls from Monday to yesterday; if task is `TimeStrictly: true`, overtime beyond daily quota credits to tomorrow.

### Running Task Timer & WebSocket (POST `/api/v1/timer/run/...`)
1. Serialized with `sync.Mutex` inside `RunningTaskService`.
2. Starting or resuming a task pauses any currently running task.
3. Broadcasts events (`TASK_STARTED`, `TASK_PAUSED`, `TASK_STOPPED`, `STATE_SYNC`) via WebSocket hub (`GET /api/v1/timer/ws`).
4. Sends interactive Telegram messages with inline buttons (pause, stop, switch, extend, rest).

### Background Workers
- **`TimerWatchdog`** (`services/timer_watchdog.go`, runs every 10s):
  - Auto-stops running tasks that reach `TargetDuration`.
  - Auto-stops free stopwatch timers at 20-minute safety cap.
  - Auto-pauses running tasks if clients do not send `POST /v1/timer/run/heartbeat` within 3 minutes.
- **`DayRollService`** (`services/day_roll_service.go`, runs every 30s):
  - Calls `EnsureTodaySchedule()` on startup and on calendar day changes.
  - Automatically creates today's task definitions from the active weekly schedule if not yet populated.

## Coding Style & Best Practices

- **Go version**: 1.25.1 (specified in `go.mod` and matched in `Dockerfile`).
- **Formatting**: Use `gofmt`/`go fmt` and `go vet` before committing.
- **Naming conventions**: 
  - Package names: `lower_snakecase`.
  - Files: `feature_action.go` pattern.
  - Exported identifiers: `PascalCase`; unexported: `lowerCamelCase`.
- **Layer separation**: Maintain clean boundaries: handlers → services → storage. Handlers live solely in `internal/api/handler/`.
- **Logging**: Use `log/slog` (with `tint` handler) for all new code; `logrus` remains in `main.go` only for legacy startup logging.
- **Error handling**: Wrap errors with descriptive context using `fmt.Errorf("context: %w", err)`. Return HTTP 500 for handler errors.

## Layer-Specific Patterns

### Handlers (`internal/api/handler/`)
- Parse request body, validate required fields, call service method.
- Return JSON with `status` and `message`/`data` fields.
- Use `log/slog` for structured logging.
- Error status code 500 for handler error responses.

### Services (`internal/services/`)
- Each service declares a storage interface containing only the subset of methods it requires.
- Services compose storage operations and house all business logic.
- Date format: `time.Now().Format("2 January 2006")` used consistently across records.

### Storage (`internal/storage/mongo/`)
- Database: `tasker`. Collections: `task_info`, `tasks`, `task_list`, `role_info`, `weekly_schedules`.
- Special singleton documents in `task_info`: `"Rest Info"`, `"Procent Info"`, `"Ramp Info"`, `"Scheduler"`, `"Day List"`.
- Indexes are initialized on startup via `EnsureIndexes(ctx)`.

## Configuration & Environment

**Required `./config.yaml` in repo root** (relative to CWD):
```yaml
mongodb:
  host: 127.0.0.1  # Use 'mongo' for Docker Compose
  port: "27017"
  name: tasker
telegram:
  api_key: ""
  room_id: 0
  enable_webapp_auth: false
```

- When `telegram.enable_webapp_auth` is true, all `/api/*` requests require valid Telegram credentials (`X-Bot-Token` or HMAC-validated `X-Telegram-Init-Data`).

## Development Workflows

### Running Locally
```bash
make run                    # Runs backend locally on :3000 (needs ./config.yaml)
```

### Full Stack with Docker Compose
```bash
make compose-up            # Starts API (:3000) and MongoDB (:27017)
make compose-logs          # Tails logs (last 200 lines)
make compose-down          # Stops and cleans up (removes volumes)
```

### Code Quality & Testing
```bash
make fmt vet tidy          # Format, vet, and tidy Go modules
make build                 # Compiles binary to bin/server
make test                  # Runs all unit tests (go test ./...)
```

## Testing Guidelines
- Unit tests exist across `internal/services/`, `internal/api/handler/`, `internal/notify/telegram/`, `internal/domain/entity/`, and `internal/storage/mongo/`.
- Hand-written mock structs implement narrow storage interfaces; tests do NOT require a running MongoDB database.
- Date-sensitive tests call `t.Skip` on Mondays when testing backfill/rollover due to lack of preceding weekdays in the week.
- `test/main.go` is an ad-hoc manual script that requires MongoDB and is excluded from `go test`.

## Key Endpoints
See `openapi.yml` for the complete API contract.
- `GET /api/v1/dashboard/state` - Aggregated dashboard metrics
- `POST /api/v1/taskrecord` - Add task time record
- `GET /api/v1/task/plan/percent` - Get next task by plan percentage
- `GET /api/v1/task/plan/percent/schedule` - Schedule-aware next task
- `GET /api/v1/stats/done/today` - Today's task completion stats
- `GET /api/v1/stats/weekly` - Current week task and role statistics
- `POST /api/v1/timer/run/start` - Start running task timer
- `POST /api/v1/timer/run/stop` - Stop running task and record elapsed time
- `POST /api/v1/timer/run/heartbeat` - Refresh timer lease
- `GET /api/v1/timer/ws` - WebSocket connection for timer updates
- `GET /api/v1/ramp/status` - Current ramp warm-up status
- `GET /api/v1/mode/evening-focus` - Prioritized catch-up task for evening focus
- `POST /api/v1/schedule/apply` - Apply today's schedule tasks
