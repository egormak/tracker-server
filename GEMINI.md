# tracker-server

The central REST API hub for the tracker system. Built with Go and Fiber, using MongoDB for persistence.

## Project Overview

- **Core Technologies**: Go 1.25+, Fiber, MongoDB, `slog` (structured logging).
- **Architecture**:
  - `cmd/server/main.go`: Entry point, sets up Fiber and logging.
  - `internal/api/handler/` & `internal/api/routes/`: Modern API layer (preferred for new endpoints).
  - `internal/handler/`: Legacy Fiber handlers (backwards compatibility).
  - `internal/services/`: Business logic (tasks, stats, plan, rest, schedule).
  - `internal/storage/`: Data access layer with MongoDB implementations.
  - `internal/domain/entity/`: Typed domain models.
  - `internal/notify/`: Telegram notification service.

## API Endpoints

The API is versioned under `/api/v1/`. Key route groups:
- **Tasks & Records**: `/api/v1/task/...`, `/api/v1/taskrecord/...`
- **Planning**: `/api/v1/task/plan/...`, `/api/v1/manage/procents/...`
- **Rest**: `/api/v1/rest/...`
- **Schedule**: `/api/v1/schedule/...`
- **Statistics**: `/api/v1/stats/...`
- **Timer**: `/api/v1/timer/...`, `/api/v1/manage/timer/...`
- **Roles**: `/api/v1/roles/...`

Refer to `openapi.yml` for the full API specification.

## Building and Running

### Commands (Makefile)
- `make run`: Run server locally (requires `config.yaml`).
- `make build`: Build binary to `bin/server`.
- `make test`: Run Go tests.
- `make fmt` / `make vet`: Format and analyze code.
- `make compose-up`: Start API and MongoDB using Docker Compose.
- `make docker-build TAG=v1`: Build Docker image.

### Configuration
Configuration is loaded from `config.yaml`. Use `config_example.yaml` as a template.
Important settings:
- `mongodb`: Host, port, and database name.
- `telegram`: API key and Room ID for notifications.

## Development Conventions

- **Clean Architecture**: Follow Handlers -> Services -> Storage flow.
- **Logging**: Use `slog` for all new logging.
- **Error Handling**: Return JSON with `status: "error"` and a message for API errors.
- **API First**: Update `openapi.yml` when changing API contracts.
- **Schedule Logic**: Ensure synchronization between entities, services, and storage when modifying schedule features.
