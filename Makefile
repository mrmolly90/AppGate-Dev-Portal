# =============================================================================
# AppGate Dev-Portal — Makefile
#
# Common developer workflows: build, test, run, lint, deploy.
# =============================================================================

SHELL := /bin/bash

# Binary output names
API_BIN    := bin/dev-portal-api
WEB_DIR    := apps/web
COMPOSE    := deploy/docker/docker-compose.yml

.PHONY: help build test lint run-web run-api dev docker-build clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go BFF and pkg libraries
	go build -o $(API_BIN) ./apps/api

test: ## Run Go tests
	go test ./...

lint: ## Run golangci-lint (if installed)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skipping"

run-api: ## Run the BFF locally (env vars from .env if present)
	@test -f .env && set -a && source .env && set +a; \
	go run ./apps/api

run-web: ## Run the React portal dev server
	cd $(WEB_DIR) && npm install --no-audit --no-fund && npm run dev

dev: ## Run the full stack via docker compose
	docker compose -f $(COMPOSE) up --build

dev-down: ## Stop the full stack
	docker compose -f $(COMPOSE) down

docker-build: ## Build Docker images
	docker build -f deploy/docker/api.Dockerfile -t appgate/dev-portal-api:latest .
	docker build -f deploy/docker/web.Dockerfile -t appgate/dev-portal-web:latest .

terraform-validate: ## Validate Terraform config
	cd deploy/terraform && terraform validate

terraform-plan: ## Plan Terraform deployment
	cd deploy/terraform && terraform plan

clean: ## Remove build artifacts
	rm -rf bin/
	rm -rf $(WEB_DIR)/node_modules $(WEB_DIR)/dist

.PHONY: check-env
check-env:
	@test -n "$$DP_CP_BASE_URL" || echo "WARNING: DP_CP_BASE_URL not set; using default"