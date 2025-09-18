Tracker Web UI
================

Minimal React (Vite + TypeScript) UI for the Go tracker-server REST API.

Prerequisites
- Node.js 18+

Config
- API base URL is configurable via `VITE_API_BASE_URL` (defaults to same-origin '').
- During `npm run dev`, Vite proxies `/api` to `http://localhost:3000` to avoid CORS.

Setup
1. cd web
2. npm install
3. npm run dev
   - Open http://localhost:5173

Build
- npm run build
- npm run preview

Pages
- Dashboard: today stats and rest balance
- Plan: show next task by plan, rotate legacy plan group, set procents (legacy manage endpoint)
- Rest: add/spend rest time
- Record: add a task record
- Manage: create a task with role (work/learn/rest/plan)
- Timer: get/set timer

Notes
- Endpoints are based on `openapi.yml` and `internal/api/routes/routes.go`.
- Some features use legacy endpoints (`/api/v1/manage/procents`, `/api/v1/task/plan-percent/change`).
- If the API runs on a different host/port in production, set `VITE_API_BASE_URL` accordingly (e.g., `https://api.example.com`).

Docker (web only)
- Build: `docker build -t ghcr.io/egormak/tracker-web:$(date +%F) --build-arg VITE_API_BASE_URL=http://localhost:3000 .`
- Run: `docker run -it --rm -p 5173:80 ghcr.io/egormak/tracker-web:$(date +%F)`
- Open http://localhost:5173
