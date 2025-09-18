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
  - `make test` – run all tests.
  - `make fmt` / `make vet` / `make tidy` – format, analyze, tidy modules.
  - `make docker-build TAG=$(date +%F)` – build image.
  - `make docker-run TAG=TAG` – run image mapping `3000` and mounting config.
- Direct (fallback):
  - `go run cmd/server/main.go`, `go build -o bin/server ./cmd/server`.
  - `docker build -t ghcr.io/egormak/tracker-server:TAG .`
  - `docker run -it --rm -p 3000:3000 -v $(pwd)/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:TAG`

## Coding Style & Naming Conventions
- Go 1.22. Use `gofmt`/`go fmt` and `go vet` before committing.
- Tabs, idiomatic Go. Package names lower_snakecase; files `feature_action.go`.
- Exported types/functions use PascalCase; unexported use lowerCamelCase.
- Keep layers clean: handlers → services → storage; avoid cross‑layer imports.

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
- Location: `web/` (Vite + React + TypeScript)
- Dev: `cd web && npm install && npm run dev` → http://localhost:5173
- During dev, Vite proxies `/api` → `http://localhost:3000`; set `VITE_API_BASE_URL` for other deployments.

## API Highlights
- See `openapi.yml` for the full spec.
- Common routes:
  - GET `/api/v1/stats/done/today` – today results
  - GET `/api/v1/task/plan/percent` – next task by plan percent
  - POST `/api/v1/taskrecord` – add record `{ task_name, time_done }`
  - Rest: GET `/api/v1/rest/get`, POST `/api/v1/rest/add`, `/api/v1/rest/spend`
  - Manage: POST `/api/v1/manage/task/create`
  - Timer: GET `/api/v1/timer/get`, POST `/api/v1/timer/set`
  - Legacy: GET `/api/v1/task/plan-percent/change`, POST `/api/v1/manage/procents`
