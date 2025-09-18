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
*   **Logrus:** For logging.
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

The main API endpoints are grouped under `/api/v1`. Here's a summary of the available routes:

*   `/api/v1/task/...`: Routes for managing tasks.
*   `/api/v1/taskrecord/...`: Routes for managing time records for tasks.
*   `/api/v1/rest/...`: Routes for managing rest time.
*   `/api/v1/manage/...`: Routes for managing the application.
*   `/api/v1/stats/...`: Routes for retrieving statistics.
*   `/api/v1/roles/...`: Routes for managing roles.

The file `internal/api/routes/routes.go` contains a complete list of all the available endpoints.

## Development Conventions

*   The project follows a standard Go project layout.
*   The code is organized into several packages, including `api`, `config`, `handler`, `models`, `repository`, `services`, and `utils`.
*   The application uses a service-based architecture, with services encapsulating the business logic.
*   Handlers are responsible for handling HTTP requests and responses.
*   The project uses `slog` for structured logging.
*   The project is in a state of active development, with some older code marked for future removal.
