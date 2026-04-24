COMPOSE ?= $(shell command -v podman-compose 2>/dev/null || echo .venv/bin/podman-compose)

.PHONY: help build dev dev-api dev-ui deps clean compose-up compose-down compose-build admin-password admin-reset

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the markovd binary
	go build -o bin/markovd ./cmd/markovd

dev-api: ## Run the Go API server
	go run ./cmd/markovd

dev-ui: ## Run the React dev server (Vite)
	cd ui && npm run dev

dev: ## Print instructions for local development
	@echo "Run 'make dev-api' and 'make dev-ui' in separate terminals"

deps: ## Install Go and JS dependencies
	go mod tidy
	cd ui && npm install

clean: ## Remove build artifacts
	rm -rf bin/

compose-build: ## Build container images
	$(COMPOSE) build

compose-up: ## Start all services
	$(COMPOSE) up -d

compose-down: ## Stop all services
	$(COMPOSE) down

admin-password: ## Extract the default admin password from API logs
	@$(COMPOSE) logs api 2>&1 | grep -A0 'Password:' | tail -1 | awk '{print $$NF}'

admin-reset: ## Reset the admin user (deletes all data) and show new password
	$(COMPOSE) exec -T postgres psql -U markovd -d markovd -c "DELETE FROM steps; DELETE FROM events; DELETE FROM runs; DELETE FROM workflows; DELETE FROM users;"
	$(COMPOSE) restart api
	@sleep 3
	@echo ""
	@$(MAKE) --no-print-directory admin-password
