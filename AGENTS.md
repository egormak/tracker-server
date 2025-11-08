# Repository Guidelines

## Project Structure & Module Organization
- Entry point: `cmd/server/main.go` (Fiber HTTP server).
- Core packages under `internal/`:
  - `api/` (`handler`, `routes`) – HTTP handlers and routing.
  - `services/` – business logic (tasks, stats, management).
  - `storage/mongo/` – MongoDB adapters and DAOs.
  - `domain/entity/`, `models/`, `options/` – typed entities and options.
  - `notify/telegram/` – Telegram notifications.
- Configuration: `config/config.go` reads `config.yaml` (see `config_example.yaml`).
- API spec: `openapi.yml`.
- Utilities: `test/main.go` (ad‑hoc manual check).
- Docker: `Dockerfile`.

## Build, Test, and Development Commands
- Primary (Makefile):
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
- Direct (fallback):
  - `go run cmd/server/main.go`, `go build -o bin/server ./cmd/server`.
  - `docker build -t ghcr.io/egormak/tracker-server:TAG .`
  - `docker run -it --rm -p 3000:3000 -v $(pwd)/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:TAG`

## Coding Style & Naming Conventions
- Go 1.25.1. Use `gofmt`/`go fmt` and `go vet` before committing.
- Tabs, idiomatic Go. Package names lower_snakecase; files `feature_action.go`.
- Exported types/functions use PascalCase; unexported use lowerCamelCase.
- Keep layers clean: handlers → services → storage; avoid cross‑layer imports.
- Logging: Use `slog` (with tint handler) for new code; `logrus` exists in main.go for legacy reasons.

## Testing Guidelines
- Prefer table‑driven tests in the same package: `*_test.go`.
- Unit test services and pure functions; use small fakes for storage.
- Run `go test ./...`; add coverage flags if needed: `go test ./... -cover`.
- Name tests by behavior: `TestService_DoThing`.

## Commit & Pull Request Guidelines
- Commit messages: imperative, concise subject (<72 chars), body explains why.
  - Examples: `Implement task handler for parameters`, `Fix error in plan percent check`.
- PRs must include: clear description, rationale, testing steps, config changes, and related issue links. Update `openapi.yml` when API changes.

## Security & Configuration Tips
- Do not hardcode secrets. Provide `config.yaml` (mount in Docker) with:
  - `mongodb.host`, `mongodb.port`, `mongodb.name`; `telegram.api_key`, `telegram.room_id`.
- Default server port: `3000`. Ensure MongoDB is reachable from the container.

## Web UI (React)
- Location: `web/` (Vite + React + TypeScript + MUI v6)
- Dev: `cd web && npm install && npm run dev` → http://localhost:5173
- During dev, Vite proxies `/api` → `http://localhost:3000`; set `VITE_API_BASE_URL` for other deployments.
- Pages: Dashboard, Plan, Record, Rest, Manage, Timer (see `web/src/pages/`)
- Docker: Web UI uses nginx:1.27-alpine with custom nginx.conf, serves on port 80
- Build: `make web-build` creates production build in `web/dist`

## API Highlights
- See `openapi.yml` for the full spec.
- Common routes:
  - GET `/api/v1/stats/done/today` – today results
  - GET `/api/v1/stats/tasks/today` – alias for stats/done/today (dashboard)
  - GET `/api/v1/task/plan/percent` – next task by plan percent
  - POST `/api/v1/taskrecord` – add record `{ task_name, time_done }`
  - Rest: GET `/api/v1/rest/get`, POST `/api/v1/rest/add`, `/api/v1/rest/spend`
  - Manage: POST `/api/v1/manage/task/create`
  - Plan Percents: GET `/api/v1/manage/plan-percents`, DELETE `/api/v1/manage/plan-percents/:group/:value`
  - Timer: GET `/api/v1/timer/get`, POST `/api/v1/timer/set`
  - Legacy: GET `/api/v1/task/plan-percent/change`, POST `/api/v1/manage/procents`
