# tracker-server

The central REST API hub for the tracker system. Built with Go and Fiber, using MongoDB for persistence.

## Project Overview

- **Core Technologies**: Go 1.25+, Fiber framework, MongoDB 5.0+, `slog` (structured logging with `tint` handler).
- **Role**: Source of truth and sole persistent storage component. No web frontend code lives in this repo (the UI resides in `tracker-web`).
- **Architecture**:
  - `cmd/server/main.go`: Entry point, loads config, connects to MongoDB, sets up logging middleware.
  - `internal/api/routes/routes.go`: Central route mapping and dependency injection hub.
  - `internal/api/middleware/telegram.go`: Authentication middleware (`X-Bot-Token` / `X-Telegram-Init-Data` validation).
  - `internal/api/handler/`: Modern API HTTP handler layer (preferred for new endpoints).
  - `internal/handler/`: Legacy Fiber handlers (backwards compatibility for roles, legacy manage endpoints).
  - `internal/services/`: Business logic services (tasks, taskrecords, plan, rest, schedule, running_task, evening).
  - `internal/storage/`: Shared storage contract (`Storage`) and MongoDB adapter (`internal/storage/mongo/`).
  - `internal/domain/entity/`: Typed domain entities.
  - `internal/notify/`: Telegram notification service.

## API Endpoints

The API is versioned under `/api/v1/`. Key route groups:
- **Tasks & Records**: `/api/v1/task/...`, `/api/v1/taskrecord/...`
- **Planning**: `/api/v1/task/plan-percent/...`, `/api/v1/manage/procents/...`
- **Rest**: `/api/v1/rest/...`
- **Schedule**: `/api/v1/schedule/...`
- **Statistics**: `/api/v1/stats/...`
- **Running Task Timer**: `/api/v1/timer/run/...` (start, status, pause, resume, stop, list)
- **Evening Focus Mode**: `/api/v1/mode/evening-focus`, `/api/v1/mode/evening-focus/skip`
- **Roles**: `/api/v1/roles/...`

Refer to `openapi.yml` for the definitive OpenAPI 3.0 specification.

## Core Business Logic Rules

- **Plan-Percent Auto-Rotation**: `GetTaskPlanPercent` automatically rotates to the next task group when current group percentages are depleted.
- **Record Multi-Step Persistence**: Recording time executes `AddTaskRecord` → `AddRoleMinutes` → `AddRest` in sequence.
- **Source Day & Backfill**: Records can target past weekdays (`source_day`). Backfill checks Monday→yesterday against active schedule before applying remaining time.
- **Dynamic Forward Overtime Credit**: For strict tasks, completed minutes past daily quota credit to tomorrow (`source_day = tomorrow`).
- **Evening Catch-Up Mode**: Micro-sprint queue (default 20 min) targeting weekly deficit tasks while excluding `work` and `english`.
- **Running Task Mutex**: `RunningTaskService` uses a `sync.Mutex` lock to serialize concurrent timer actions across clients.

## Building and Running

### Commands (Makefile)
- `make run`: Run server locally on `:3000` (requires `./config.yaml`).
- `make build`: Build binary to `bin/server`.
- `make test`: Run Go tests (`go test ./...`).
- `make fmt` / `make vet` / `make tidy`: Format, vet, and tidy Go module.
- `make compose-up`: Start API and MongoDB using Docker Compose.
- `make docker-build TAG=v1`: Build Docker image.

### Configuration
Loaded from `config.yaml` (see `config_example.yaml` for structure):
- `mongodb`: host, port, database (`tasker`).
- `telegram`: `api_key`, `room_id`, `enable_webapp_auth`.

## Development Conventions

- **Clean Architecture**: Handlers → Services → Storage interface → MongoDB.
- **API First**: Always update `openapi.yml` when modifying endpoint parameters or responses.
- **Logging**: Use `slog` for all new logging.
- **Error Responses**: Return JSON `{status: "error", message: "..."}` with standard HTTP 500 for backend errors.
