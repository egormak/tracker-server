# Copilot Instructions for Tracker Server

## Architecture Overview

**Three-tier Go/Fiber application for time tracking with MongoDB and React UI**

- **Entry**: `cmd/server/main.go` initializes Fiber, MongoDB connection, Telegram notifier, and routes
- **Layers**: HTTP handlers (`internal/api/handler/`) → Services (`internal/services/`) → Storage (`internal/storage/mongo/`)
- **Key abstraction**: All storage operations go through `storage.Storage` interface (see `internal/storage/storage.go`)
- **Notification**: `internal/notify/notify.go` interface with Telegram implementation for task start/stop messages

### Core Domain Concepts
- **Three roles**: `work`, `learn`, `rest` (hardcoded in `internal/storage/mongo/mongo.go`)
- **Task tracking**: Tasks have a defined schedule (`time_schedule`), daily completion (`time_done`), and priority
- **Plan percent system**: Tasks organized into groups (`plan`, `work`, `learn`, `rest`) with percentage allocations that determine next task selection
- **Rest balance**: Completing tasks adds rest minutes; rest activities deduct from balance

## Critical Data Flow Patterns

### Adding a Task Record (POST `/api/v1/taskrecord`)
1. Handler (`taskrecord_handler.go`) validates request → Service (`taskrecord_service.go`)
2. Service fetches task role → creates `entity.TaskRecord` with today's date
3. **Three storage operations**: `AddTaskRecord()`, `AddRoleMinutes()`, `AddRest()` (rest balance update)
4. This is the main interaction point for recording time spent

### Plan Percent Task Selection (GET `/api/v1/task/plan/percent`)
1. Gets current plan group ordinal → fetches group percent
2. Finds task name for that group+percent → calculates time remaining for today
3. **After returning task**: Client must rotate to next percent or group (not automatic)
4. Legacy endpoint `/api/v1/task/plan-percent/change` rotates the group

## Layer-Specific Patterns

### Handlers (`internal/api/handler/`)
- Parse request body, validate required fields, call service method
- Return JSON with `status` and `message`/`data` fields
- Use `log/slog` for structured logging: `slog.Info()`, `slog.Error()`
- Error status code 500 for all errors (not RESTful, but consistent)

### Services (`internal/services/`)
- Each service has a storage interface (e.g., `TaskRecordStorage`) defining only needed methods
- Services compose storage operations and contain business logic
- Return errors wrapped with context: `fmt.Errorf("descriptive context: %s", err)`
- Date format: `time.Now().Format("2 January 2006")` throughout codebase

### Storage (`internal/storage/mongo/`)
- Database: `tasker`, Collections: `task_info`, `tasks`, `task_list`, `role_info`
- Special documents: `"Rest Info"` for rest balance, `"Procent Info"` for plan percentages
- Constants defined in `mongo.go`: `dbName`, collection names, role types, plan types

## Configuration & Environment

**Required `config.yaml` in repo root**:
```yaml
mongodb:
  host: 127.0.0.1  # Use 'mongo' for Docker Compose
  port: "27017"
  name: tracker
telegram:
  api_key: ""      # Optional for notifications
  room_id: 0
```

- Config loaded via `config.LoadConfig()` at startup
- No environment variables used; all config file-based
- Docker Compose: Set `mongodb.host: mongo` to use service name

## Development Workflows

### Running Locally
```bash
# Backend (requires config.yaml and MongoDB running)
make run                    # Starts on :3000

# Frontend (proxies /api → http://localhost:3000)
cd web && npm install && npm run dev  # http://localhost:5173
```

### Full Stack with Docker Compose
```bash
make compose-up            # API (:3000), Web (:8080), Mongo (:27017)
make compose-logs          # Tail logs
make compose-down          # Stop and cleanup
```

### Code Quality & Build
```bash
make fmt vet tidy          # Format, analyze, tidy modules
make build                 # Binary to bin/server
make test                  # Run all tests (currently no test files exist)
```

## API Design Conventions

- **Base path**: `/api/v1/`
- **Naming inconsistency**: Mix of legacy (`/rest-get`) and new (`/rest/get`) endpoints; prefer new style
- **Route registration**: All routes in `internal/api/routes/routes.go` with handler instantiation
- **Dual handler systems**: New handlers in `internal/api/handler/`, legacy in `internal/handler/` (being migrated)
- **OpenAPI spec**: `openapi.yml` is the authoritative API contract; update when adding/changing endpoints

### Key Endpoints
- `POST /api/v1/taskrecord` - Add task time record (main interaction)
- `GET /api/v1/task/plan/percent` - Get next task by plan percentage
- `GET /api/v1/stats/done/today` - Today's task completion stats
- `GET /api/v1/rest/get` - Current rest balance
- `POST /api/v1/manage/task/create` - Create new task definition

## Web UI Integration

- **Tech**: React + TypeScript + Vite + MUI
- **API client**: `web/src/api/client.ts` with typed interfaces matching OpenAPI
- **Dev proxy**: Vite proxies `/api` to backend (configured in `vite.config.ts`)
- **Production**: Set `VITE_API_BASE_URL` environment variable or use same-origin deployment
- **Pages**: Dashboard, Plan, Record, Rest, Manage, Timer (see `web/src/pages/`)

## Common Gotchas

1. **No tests exist yet**: Project has test infrastructure (`make test`) but no `*_test.go` files
2. **Date format consistency**: Always use `"2 January 2006"` for task records (not RFC3339)
3. **Handler migration in progress**: Some routes use old handlers (`internal/handler/`), some use new (`internal/api/handler/`)
4. **Plan percent state**: System doesn't auto-advance to next task; client must call change endpoint
5. **Rest balance**: Adding task records automatically adds rest time; no manual adjustment needed unless spending rest
6. **MongoDB connection**: No authentication configured; adjust for production use

## Adding New Features

### New API Endpoint Checklist
1. Add entity types to `internal/domain/entity/` if needed
2. Add storage interface method to `internal/storage/storage.go`
3. Implement MongoDB method in `internal/storage/mongo/`
4. Create/extend service in `internal/services/` with storage interface subset
5. Create/extend handler in `internal/api/handler/`
6. Register route in `internal/api/routes/routes.go`
7. Update `openapi.yml` with new endpoint specification
8. Add corresponding TypeScript types and functions to `web/src/api/client.ts`

### Storage Layer Pattern
```go
// Service defines only what it needs
type MyServiceStorage interface {
    GetFoo(id string) (Foo, error)
    SaveFoo(foo Foo) error
}

// Full storage implements everything
func (s *Storage) GetFoo(id string) (Foo, error) {
    collection := s.Client.Database(dbName).Collection("foos")
    // ... MongoDB query
}
```

## External Dependencies

- **Fiber v2**: HTTP framework (Express-like for Go)
- **MongoDB Driver**: `go.mongodb.org/mongo-driver/mongo`
- **slog**: Structured logging (Go 1.21+ standard library)
- **Telegram Bot API**: Optional notifications via `internal/notify/telegram/`
- **YAML v3**: Config file parsing (`gopkg.in/yaml.v3`)
