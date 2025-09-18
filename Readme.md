# Tracker Server
Time-tracking REST API (Go/Fiber + MongoDB) with a minimal React web UI.

## Overview
- Roles: Work, Learn, Rest
- Layers: handlers → services → storage (MongoDB)
- Spec: `openapi.yml`
- Web UI: `web/` (Vite + React + TS)

## Project Structure
- `cmd/server/main.go` – entrypoint
- `internal/api/handler` – HTTP handlers
- `internal/api/routes/routes.go` – route wiring
- `internal/services/` – business logic
- `internal/storage/mongo/` – Mongo adapters/DAOs
- `config/config.go` – loads `./config.yaml`
- `openapi.yml` – API contract
- `web/` – React client

## Configuration
Provide `config.yaml` in repo root (see `config_example.yaml`):
```yaml
mongodb:
  host: 127.0.0.1
  port: "27017"
  name: tracker
telegram:
  api_key: ""
  room_id: 0
```

## Run (local)
Backend:
```bash
make run  # runs on :3000
```
Frontend:
```bash
cd web
npm install
npm run dev  # http://localhost:5173 (proxies /api → http://localhost:3000)
```

## Build
Binary:
```bash
make build  # bin/server
```
Docker:
```bash
make docker-build TAG=$(date +%F)
make docker-run TAG=$(date +%F)
# or
docker build -t ghcr.io/egormak/tracker-server:$(date +%F) .
docker run -it --rm -p 3000:3000 -v ${PWD}/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:$(date +%F)
```

Docker Compose (API + Web + Mongo):
```bash
# Ensure config.yaml has mongodb.host set to 'mongo' and port '27017'
make compose-up
# Web UI: http://localhost:8080  (nginx proxies /api to the API service)
# API:    http://localhost:3000

# Stop and cleanup
make compose-down
```

## Make targets
```bash
make run            # run backend on :3000
make build          # build backend binary to bin/server
make docker-build   # build backend image (TAG overrideable)
make docker-run     # run backend image (maps 3000)
make fmt vet tidy   # formatting and analysis
make web-dev        # run React dev server (web/)
make web-build      # build React app to web/dist
make web-preview    # preview built React app
make web-docker-build TAG=$(date +%F)  # build web image
make web-docker-run  TAG=$(date +%F)   # run web image on :5173
make all            # backend fmt/vet/build + web build
```

## Web UI (web/)
- Vite dev server proxies `/api` to `http://localhost:3000` to avoid CORS.
- Configure a different API in production with `VITE_API_BASE_URL`.

Available pages:
- Dashboard – today’s stats and rest balance
- Plan – next-by-plan, rotate plan group (legacy), set procents
- Rest – add/spend minutes
- Record – add a task record
- Manage – create a task (work/learn/rest/plan)
- Timer – get/set timer

## Common API Endpoints
See `openapi.yml`. Highlights:
- GET `/api/v1/stats/done/today` – today’s task results
- GET `/api/v1/task/plan/percent` – next task by plan percent
- POST `/api/v1/taskrecord` – add task record `{ task_name, time_done }`
- Rest: GET `/api/v1/rest/get`, POST `/api/v1/rest/add`, POST `/api/v1/rest/spend`
- Manage: POST `/api/v1/manage/task/create`
- Timer: GET `/api/v1/timer/get`, POST `/api/v1/timer/set`

Legacy helpers (still wired):
- GET `/api/v1/task/plan-percent/change`
- POST `/api/v1/manage/procents`

## Notes
- Default server port: 3000
- Ensure MongoDB is reachable from the container/host
