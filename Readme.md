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
docker build -t egorm/tracker-server:v0.6.4 .
docker push egorm/tracker-server:v0.6.4
```

# Run Project
## Dev
```shell
docker run -it --rm -p 3000:3000 -v ${PWD}/config.yaml:/config.yaml egorm/tracker-server:v0.6.4
```
## Prod
```shell
docker stop tracker
docker rm tracker
docker run -d -p 8080:3000 --name tracker  --network=tracker -v /etc/tracker/config.yaml:/config.yaml egorm/tracker-server:v0.6.4
```

# Notes
In config file was set MongoDB Docker ip-address