# Makefile for producer-service

# Variables
APP_NAME := producer-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Docker
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_REGISTRY := localhost:5000

# Directories
BUILD_DIR := ./build
BIN_DIR := $(BUILD_DIR)/bin

.PHONY: help build run test clean lint fmt vet deps docker-build docker-run docker-push

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development commands
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

run: ## Run the application locally
	@echo "Running $(APP_NAME)..."
	go run ./cmd/server

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code quality
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	go mod verify

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 -p 9090:9090 $(DOCKER_IMAGE)

docker-push: docker-build ## Push Docker image to registry
	@echo "Pushing Docker image to $(DOCKER_REGISTRY)..."
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -cache

# Development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Local development with Docker Compose
compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	cd ../../ && docker-compose up -d

compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	cd ../../ && docker-compose down

compose-logs: ## Show docker-compose logs
	@echo "Showing docker-compose logs..."
	cd ../../ && docker-compose logs -f $(APP_NAME)

# Health checks
health: ## Check application health
	@echo "Checking application health..."
	curl -f http://localhost:8080/health || echo "Health check failed"

ready: ## Check application readiness
	@echo "Checking application readiness..."
	curl -f http://localhost:8080/ready || echo "Readiness check failed"

metrics: ## Show application metrics
	@echo "Showing application metrics..."
	curl -s http://localhost:9090/metrics | grep -E "^(producer_|http_)" | head -20

# Load testing
load-test: ## Run simple load test
	@echo "Running load test..."
	@for i in {1..10}; do \
		curl -X POST http://localhost:8080/api/v1/events/user \
		-H "Content-Type: application/json" \
		-d '{"data": "Load test user $$i"}' & \
	done; wait
	@echo "Load test completed"

# Git hooks
install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	cp scripts/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit

# Release
release: test lint ## Create a release build
	@echo "Creating release build..."
	@mkdir -p $(BUILD_DIR)/release
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(APP_NAME)-linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(APP_NAME)-darwin-amd64 ./cmd/server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(APP_NAME)-windows-amd64.exe ./cmd/server

# Info
info: ## Show build information
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"
