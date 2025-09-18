SHELL := /bin/bash

BIN_DIR := bin
BIN := $(BIN_DIR)/server
CMD := ./cmd/server
PKG := ./...
IMAGE := ghcr.io/egormak/tracker-server
TAG ?= $(shell date +%F)

.PHONY: help run build test fmt vet tidy docker-build docker-run docker-prod docker-stop clean web-dev web-build web-preview web-docker-build web-docker-run compose-up compose-down compose-logs all

help: ## Show available targets
	@echo "Usage: make <target>"
	@echo "Targets:"
	@echo "  run           Run the server locally (needs ./config.yaml)"
	@echo "  build         Build the server binary to $(BIN)"
	@echo "  test          Run all tests"
	@echo "  fmt           Format code (go fmt)"
	@echo "  vet           Static analysis (go vet)"
	@echo "  tidy          Sync go.mod/go.sum"
	@echo "  docker-build  Build Docker image $(IMAGE):$(TAG)"
	@echo "  docker-run    Run Docker image in dev mode (maps 3000)"
	@echo "  docker-prod   Run Docker image as named container 'tracker'"
	@echo "  docker-stop   Stop and remove 'tracker' container"
	@echo "  web-dev       Run React dev server (web/)"
	@echo "  web-build     Build React app to web/dist"
	@echo "  web-preview   Preview built React app"
	@echo "  web-docker-build  Build web UI Docker image"
	@echo "  web-docker-run    Run web UI Docker image on :5173"
	@echo "  compose-up    Run API+Web+Mongo via docker-compose (web on :8080)"
	@echo "  compose-down  Stop compose stack"
	@echo "  compose-logs  Tail logs"
	@echo "  all           Backend fmt/vet/build + web build"
	@echo "  clean         Remove built binaries"

run: ## Run the server locally (needs ./config.yaml)
	go run $(CMD)

build: ## Build the server binary
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD)

test: ## Run all tests
	go test $(PKG)

fmt: ## Format code
	go fmt $(PKG)

vet: ## Static analysis
	go vet $(PKG)

tidy: ## Sync go.mod/go.sum
	go mod tidy

docker-build: ## Build Docker image
	docker build -t $(IMAGE):$(TAG) .

docker-run: ## Run Docker image in dev mode (maps port 3000; mounts ./config.yaml)
	docker run -it --rm -p 3000:3000 -v $(PWD)/config.yaml:/config.yaml $(IMAGE):$(TAG)

docker-prod: ## Run Docker image in background as 'tracker' container (maps 8080->3000)
	docker stop tracker 2>/dev/null || true
	docker rm tracker 2>/dev/null || true
	docker run -d -p 8080:3000 --name tracker --network=tracker -v /etc/tracker/config.yaml:/config.yaml $(IMAGE):$(TAG)

docker-stop: ## Stop and remove 'tracker' container
	docker stop tracker 2>/dev/null || true
	docker rm tracker 2>/dev/null || true

clean: ## Remove built binaries
	rm -rf $(BIN_DIR)

# --- Web UI helpers ---
WEB_DIR := web
WEB_IMAGE := ghcr.io/egormak/tracker-web

web-dev: ## Run React dev server (web/)
	cd $(WEB_DIR) && npm run dev

web-build: ## Build React app (web/dist)
	cd $(WEB_DIR) && npm install && npm run build

web-preview: ## Preview built React app locally
	cd $(WEB_DIR) && npm run preview

web-docker-build: ## Build Docker image for web UI
	docker build -t $(WEB_IMAGE):$(TAG) $(WEB_DIR)

web-docker-run: ## Run Docker image for web UI (maps 5173)
	docker run -it --rm -p 5173:80 $(WEB_IMAGE):$(TAG)

all: fmt vet build web-build ## Backend fmt/vet/build + web build

compose-up: ## Start API + Web + Mongo (web:8080)
	docker compose up -d --build

compose-down: ## Stop compose stack
	docker compose down -v

compose-logs: ## Tail compose logs
	docker compose logs -f --tail=200
