.PHONY: help build run test clean dev deps lint migrate db-shell db-reset test-unit test-coverage

# Colors for Windows (simple version)
GREEN  := 
YELLOW := 
WHITE  := 
RESET  := 

## Help
help: ## Show this help
	@echo.
	@echo Usage:
	@echo   make ^<target^>
	@echo.
	@echo Targets:

## Build
build: ## Build the project with Docker
	docker-compose build

## Run
run: ## Run the project with Docker
	docker-compose up --build

run-detached: ## Run in detached mode
	docker-compose up --build -d

stop: ## Stop running containers
	docker-compose down

## Development
dev: deps ## Run locally for development
	@echo "${GREEN}Starting server in development mode...${RESET}"
	@DATABASE_URL=postgres://user:password@localhost:5432/db?sslmode=disable PORT=8080 go run cmd/server/main.go

dev-with-db: ## Run locally with database in Docker
	@echo "${GREEN}Starting database...${RESET}"
	docker-compose up db -d
	@sleep 5
	@echo "${GREEN}Applying migrations...${RESET}"
	docker-compose run migrate
	@echo "${GREEN}Starting server...${RESET}"
	@DATABASE_URL=postgres://user:password@localhost:5432/db?sslmode=disable PORT=8080 go run cmd/server/main.go

## Dependencies
deps: ## Download Go dependencies
	go mod download
	go mod verify

deps-tidy: ## Tidy Go dependencies
	go mod tidy

## Database
migrate: ## Run database migrations
	docker-compose run migrate

migrate-down: ## Rollback database migrations
	docker-compose run migrate -path /migrations -database postgres://user:password@db:5432/db?sslmode=disable down

db-shell: ## Connect to database
	docker-compose exec db psql -U user -d db

db-reset: clean ## Reset database completely
	@echo "${YELLOW}Resetting database...${RESET}"
	docker-compose down -v
	docker-compose up db -d
	@sleep 5
	docker-compose run migrate

db-logs: ## Show database logs
	docker-compose logs db

## Testing
test: ## Run all tests
	go test ./... -v

test-unit: ## Run unit tests only
	go test ./internal/... -v

test-service: ## Run service tests only
	go test ./internal/service/... -v

test-handlers: ## Run handlers tests only
	go test ./internal/handlers/... -v

test-repo: ## Run repository tests only
	go test ./internal/repo/... -v

test-coverage: ## Run tests with coverage
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${RESET}"

test-coverage-text: ## Run tests with coverage (text output)
	go test ./... -cover

## Linting and Code Quality
lint: ## Run linter (if golangci-lint is installed)
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "${YELLOW}golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest${RESET}"; \
	fi

fmt: ## Format Go code
	go fmt ./...

vet: ## Vet Go code
	go vet ./...

## Cleanup
clean: ## Clean up containers and volumes
	docker-compose down -v
	@echo "${GREEN}Cleaned up Docker containers and volumes${RESET}"

clean-cache: ## Clean Go cache
	go clean -cache
	go clean -testcache
	@echo "${GREEN}Cleaned Go cache${RESET}"

clean-all: clean clean-cache ## Clean everything
	@rm -f coverage.out coverage.html 2>/dev/null || true
	@echo "${GREEN}Cleaned everything${RESET}"

## Monitoring
logs: ## Show application logs
	docker-compose logs -f app

logs-all: ## Show all logs
	docker-compose logs -f

status: ## Show container status
	docker-compose ps

health: ## Check service health
	@curl -f http://localhost:8080/health || echo "${YELLOW}Service is not running${RESET}"

## API Testing (requires running service)
api-test: ## Quick API test (requires running service)
	@echo "${GREEN}Testing API endpoints...${RESET}"
	@curl -s http://localhost:8080/health | jq . || curl -s http://localhost:8080/health

api-test-full: ## Full API test scenario
	@echo "${GREEN}Running full API test scenario...${RESET}"
	@./scripts/test-api.sh || echo "${YELLOW}Create scripts/test-api.sh for full testing${RESET}"

## Build for production
build-prod: ## Build production image
	docker build -t pr-review-assigner:prod --target prod .

## Default target
.DEFAULT_GOAL := help