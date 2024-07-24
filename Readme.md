# General
Tracker Server is a simple REST API for time tracking.
Tracker Server has 3 Roles:
- Work
- Rest
- Learn

## Role Work
Main goal of this role is to track time spent on work.
## Role Rest
Main goal of this role is to track time spent on rest.
It can be something like:
- Video
- Games
- Home Task
- Movies
## Role Learn
Main goal of this role is to track time spent on learning.
It can be something like:
- Read Info
- Programming (Dev)
- Administration (Ops)
- English (Language)

# Build Project
```shell
docker build -t ghcr.io/egormak/tracker-server:$(date +%Y-%m-%d) .
docker push ghcr.io/egormak/tracker-server:$(date +%Y-%m-%d)
```

# Run Project
## Dev
```shell
docker run -it --rm -p 3000:3000 -v ${PWD}/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:$(date +%Y-%m-%d)
```
## Prod
```shell
docker stop tracker
docker rm tracker
docker run -d -p 8080:3000 --name tracker  --network=tracker -v /etc/tracker/config.yaml:/config.yaml ghcr.io/egormak/tracker-server:$(date +%Y-%m-%d)
```

# Notes
In config file was set MongoDB Docker ip-address

# Base structure
```
TaskTimeRecord/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   └── task_handler.go
│   │   ├── middlewares/
│   │   │   └── auth_middleware.go
│   │   └── routes/
│   │       └── routes.go
│   ├── config/
│   │   └── config.go
│   ├── models/
│   │   └── task.go
│   ├── repository/
│   │   └── mongodb/
│   │       └── task_repository.go
│   ├── services/
│   │   ├── task_service.go
│   │   └── notification_service.go
│   └── utils/
│       └── logger.go
├── pkg/
│   └── telegram/
│       └── client.go
├── scripts/
│   └── db_init.go
├── tests/
│   ├── integration/
│   │   └── api_test.go
│   └── unit/
│       └── task_service_test.go
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```