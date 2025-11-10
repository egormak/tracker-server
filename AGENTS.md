# Repository Guidelines

## Project Structure & Module Organization
- Entry point: `cmd/server/main.go` (Fiber HTTP server + Fiber logger + slog/tint setup).
- Core packages under `internal/`:
  - `api/handler`, `api/routes` – **preferred** HTTP layer for new endpoints (task, rest, stats, manage, schedule).
  - `handler/` – legacy Fiber handlers still powering `/api/v1/record`, `/api/v1/manage/procents`, `/api/v1/roles`, etc. Touch only when maintaining backwards compatibility.
  - `services/` – business logic (tasks, stats, management, plan percents, rest, schedule, task records). `service/days.go` holds shared helpers.
  - `storage/` + `storage/mongo/` – storage interfaces/adapters (tasks, records, rest, timer, schedule `weekly_schedules` collection, etc.).
  - `domain/entity/`, `models/`, `options/` – typed domain models (tasks, rest, manage, schedule requests/rollovers).
  - `notify/` (interface) and `notify/telegram/` – Telegram notification adapters.
- Configuration: `config/config.go` reads `./config.yaml` (see `config_example.yaml`).
- Documentation: `openapi.yml` (authoritative API contract) + `docs/SCHEDULE_*.md` (schedule design, quick start, integration notes).
- Utilities: `test/main.go` (manual Mongo smoke), `bin/` for compiled binary.
- Frontend: `web/` (Vite + React + TS + MUI v6).
- Docker assets: `Dockerfile`, `docker-compose.yml`.

## Build, Test, and Development Commands
- Primary (Makefile):
  - `make help` – list available targets.
  - `make run` – run server locally (needs `./config.yaml`).
  - `make build` – build binary to `bin/server`.
  - `make test` – run all tests (currently no test files exist).
  - `make fmt` / `make vet` / `make tidy` – format, analyze, tidy modules.
  - `make docker-build TAG=$(date +%F)` – build backend image.
  - `make docker-run TAG=TAG` – run backend image mapping `3000` and mounting config.
  - `make docker-prod` – run as 'tracker' container (maps 8080→3000).
  - `make docker-stop` – stop and remove 'tracker' container.
  - `make web-dev` / `make web-build` / `make web-preview` – React dev/build/preview.
  - `make web-docker-build TAG=TAG` / `make web-docker-run` – Web UI Docker ops.
  - `make compose-up` – Start API+Web+Mongo stack (web on :8080).
  - `make compose-down` – Stop compose stack (removes volumes).
  - `make compose-logs` – Tail compose logs (shows last 200 lines).
  - `make all` – Backend fmt/vet/build + web build.
  - `make clean` – remove `bin/` artifacts.
- Direct (fallback):
  - `go run cmd/server/main.go`, `go build -o bin/server ./cmd/server`.
  - `docker build -t ghcr.io/egormak/tracker-server:TAG .`
  - `docker run -it --rm -p 3000:3000 -v $(pwd)/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:TAG`

## Coding Style & Naming Conventions
- Go 1.25.1. Use `gofmt`/`go fmt` and `go vet` before committing.
- Tabs, idiomatic Go. Package names lower_snakecase; files `feature_action.go`.
- Exported types/functions use PascalCase; unexported use lowerCamelCase.
- Keep layers clean: `internal/api` handlers → `internal/services` → `internal/storage`; keep `internal/handler/*` for legacy flows only.
- Schedule logic must keep `entity.ScheduleRequest`, `services.ScheduleService`, and `storage/mongo/schedule.go` synchronized (validation + persistence shape).
- Logging: Use `slog` (with tint handler) for new code; `logrus` exists in main.go for legacy reasons.

## Testing Guidelines
- Prefer table‑driven tests in the same package: `*_test.go`.
- Unit test services and pure functions; use small fakes for storage.
- Run `go test ./...`; add coverage flags if needed: `go test ./... -cover`.
- Name tests by behavior: `TestService_DoThing`.
- `test/main.go` is a manual Mongo connectivity helper and is not run by `go test`.

## Commit & Pull Request Guidelines
- Commit messages: imperative, concise subject (<72 chars), body explains why.
  - Examples: `Implement task handler for parameters`, `Fix error in plan percent check`.
- PRs must include: clear description, rationale, testing steps, config changes, and related issue links. Update `openapi.yml` when API changes.

## Security & Configuration Tips
- Do not hardcode secrets. Provide `config.yaml` (mount in Docker) with:
  - `mongodb.host`, `mongodb.port`, `mongodb.name`; `telegram.api_key`, `telegram.room_id`.
- Default server port: `3000`. Ensure MongoDB is reachable from the container.
- `config.yaml` must live in repo root; when using `docker-compose`, set `mongodb.host: mongo` so the API container can reach the bundled database.
- `telegram.room_id` is an `int64` in `config.Config`; keep values numeric (quotes in the example are for illustration only).

## Web UI (React)
- Location: `web/` (Vite + React + TypeScript + MUI v6)
- Dev: `cd web && npm install && npm run dev` → http://localhost:5173 (first run requires `npm install`)
- During dev, Vite proxies `/api` → `http://localhost:3000`; set `VITE_API_BASE_URL` for other deployments.
- Pages: Dashboard, Plan, Record, Rest, Manage, Timer (see `web/src/pages/`)
- Docker: Web UI uses nginx:1.27-alpine with custom nginx.conf, serves on port 80
- Build: `make web-build` runs `npm install` (if needed) then outputs to `web/dist`.

## API Highlights
- See `openapi.yml` for the full spec.
- Tasks & Records:
  - GET `/api/v1/task/plan/percent` – classic plan-based next task
  - GET `/api/v1/task/plan/percent/schedule` – schedule-aware next task
  - POST `/api/v1/taskrecord` – add record `{ task_name, time_done }`
- Rest & Manage:
  - GET `/api/v1/rest/get`, POST `/api/v1/rest/add`, POST `/api/v1/rest/spend`
  - POST `/api/v1/manage/task/create`
  - Plan percents: GET `/api/v1/manage/plan-percents`, DELETE `/api/v1/manage/plan-percents/:group/:value`
- Statistics:
  - GET `/api/v1/stats/done/today`
  - GET `/api/v1/stats/tasks/today` (dashboard alias)
- Schedule (see `docs/SCHEDULE_*.md` for workflows):
  - POST `/api/v1/schedule` (with optional `set_active`)
  - GET `/api/v1/schedule/active`, `/api/v1/schedule/active/today`, `/api/v1/schedule/active/rollover`
  - POST `/api/v1/schedule/apply` – create today's tasks from active schedule
  - CRUD via `/api/v1/schedule/:id`, `/api/v1/schedule/:id/activate`
- Timer & Legacy helpers (still routed via `internal/handler/*`):
  - Timer: GET `/api/v1/timer/get`, POST `/api/v1/timer/set`, `/api/v1/timer/del`, plus `/api/v1/manage/timer/*` legacy controls
  - Records/roles/manage legacy routes: `/api/v1/record/*`, `/api/v1/task/plan-percent/change`, `/api/v1/manage/procents`, `/api/v1/roles/*`, `/api/v1/manage/telegram/*`
