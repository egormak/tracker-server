# Gemini Code Understanding

## Project Overview

This project is a time-tracking server written in Go. It provides a REST API to track time spent on different activities, which are categorized into three main roles: "Work", "Rest", and "Learn". The server uses the Fiber web framework to expose the API, MongoDB as its database, and can send notifications via Telegram.

The application is containerized using Docker, and the `README.md` provides instructions for building and running the application with Docker.

## Key Technologies

*   **Go:** The primary programming language.
*   **Fiber:** A Go web framework.
*   **MongoDB:** The database for storing time-tracking data.
*   **Telegram:** Used for sending notifications.
*   **Docker:** For containerization and deployment.
*   **slog:** For structured logging.
*   **YAML:** For configuration.

## Building and Running

The project is designed to be built and run using Docker.

**Build the Docker image:**

```shell
docker build -t ghcr.io/egormak/tracker-server:$(date +%Y-%m-%d) .
```

**Run the application:**

The `README.md` provides several options for running the application (test, dev, prod). For local development, you can use the following command, making sure to have a `config.yaml` file in the root of the project:

```shell
docker run -it --rm -p 3000:3000 -v ${PWD}/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:$(date +%Y-%m-%d)
```

## API Endpoints

The main API endpoints are grouped under `/api/v1`. The primary route groups include:
*   `/api/v1/task/...`: Routes for managing tasks and task plans.
*   `/api/v1/taskrecord/...`: Routes for managing time records for tasks.
*   `/api/v1/rest/...`: Routes for managing rest time.
*   `/api/v1/manage/...`: Routes for application management, including plan percentages and timers.
*   `/api/v1/stats/...`: Routes for retrieving statistics.
*   `/api/v1/roles/...`: Routes for managing roles.
*   `/api/v1/schedule/...`: Routes for creating and managing schedules.
*   `/api/v1/timer/run/...`: Routes for starting, stopping, and checking the status of running tasks.


The file `internal/api/routes/routes.go` contains a complete list of all the available endpoints, including legacy routes.

## Development Conventions

### Project Structure

*   `cmd/`: Main entry points for the application.
*   `internal/api`: API related code.
*   `internal/config`: Configuration loading logic.
*   `internal/domain`: Domain entities and maybe some service interfaces.
*   `internal/handler`: HTTP Handlers grouped by feature (e.g., `manage`, `rest`, `welcome`).
*   `internal/services`: Business logic services.
*   `internal/storage`: Database access interfaces (`Storage`) and implementations (e.g., `mongo`).
*   `internal/notify`: Notification services (Telegram).

## Development Conventions

### Architecture
*   **Service-Based**: Business logic should reside in `internal/services`.
*   **Dependency Injection**: Handlers and services should receive their dependencies (like storage or notify interfaces) via their `New` constructor functions.
*   **Storage**: Database access is abstracted via the `Storage` interface in `internal/storage/storage.go`.

### Coding Standards
*   **Logging**: Use `log/slog` for all new logging.
    *   *Note*: `github.com/sirupsen/logrus` is present in the codebase but should be considered legacy. Prefer `slog`.
*   **Error Handling**:
    *   Return errors with context where possible.
    *   API errors should return a JSON response with `status: "error"` and a `message`.
*   **Configuration**:
    *   Configuration is loaded from `config.yaml` via the `config` package.
    *   Use `slog` to log configuration loading errors.

### API Responses
*   Success: `{"status": "success", "data": ...}` or `{"status": "accept", "message": ...}`
*   Error: `{"status": "error", "message": ...}`
