# Consumer Service Makefile

# Variables
APP_NAME := consumer-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

# Docker
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := $(VERSION)

# Directories
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage

.PHONY: help build run test clean docker-build docker-run docker-push lint fmt vet deps tidy check coverage benchmark

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development commands
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

build-linux: ## Build for Linux
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux ./cmd/server

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	go run ./cmd/server

run-dev: ## Run with development settings
	@echo "Running $(APP_NAME) in development mode..."
	APP_ENV=development LOG_LEVEL=debug go run ./cmd/server

# Testing
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race -v ./...

test-short: ## Run short tests
	@echo "Running short tests..."
	go test -short -v ./...

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

benchmark: ## Run benchmarks
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

check: fmt vet lint test ## Run all checks

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	go mod vendor

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 -p 9090:9090 \
		-e KAFKA_BROKER_LIST=host.docker.internal:9092 \
		$(DOCKER_IMAGE):latest

docker-push: ## Push Docker image
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

# Health checks
health: ## Check application health
	@echo "Checking application health..."
	@curl -f http://localhost:8080/health || echo "Health check failed"

ready: ## Check application readiness
	@echo "Checking application readiness..."
	@curl -f http://localhost:8080/ready || echo "Readiness check failed"

stats: ## Get application statistics
	@echo "Getting application statistics..."
	@curl -s http://localhost:8080/stats | jq . || echo "Stats check failed"

metrics: ## Get Prometheus metrics
	@echo "Getting Prometheus metrics..."
	@curl -s http://localhost:9090/metrics || echo "Metrics check failed"

# Load testing
load-test: ## Run load test (requires hey tool)
	@echo "Running load test..."
	@command -v hey >/dev/null 2>&1 || { echo "hey tool is required. Install with: go install github.com/rakyll/hey@latest"; exit 1; }
	hey -n 1000 -c 10 http://localhost:8080/health

# Monitoring
logs: ## Show application logs (if running in Docker)
	@echo "Showing application logs..."
	docker logs -f $(APP_NAME) 2>/dev/null || echo "Container not running"

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -rf vendor/

clean-docker: ## Clean Docker images
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest 2>/dev/null || true

# Release
release: clean check build-linux docker-build ## Prepare release
	@echo "Release $(VERSION) prepared"

# Development setup
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/rakyll/hey@latest
	@echo "Development environment setup complete"

# Database/Infrastructure commands (if needed)
kafka-up: ## Start Kafka (requires docker-compose in parent directory)
	@echo "Starting Kafka..."
	cd ../../ && docker-compose up -d kafka zookeeper

kafka-down: ## Stop Kafka
	@echo "Stopping Kafka..."
	cd ../../ && docker-compose down

# Debugging
debug: ## Run with debugging enabled
	@echo "Running with debugging enabled..."
	APP_DEBUG=true LOG_LEVEL=debug go run ./cmd/server

profile: ## Run with profiling enabled
	@echo "Running with profiling enabled..."
	go run ./cmd/server -cpuprofile=cpu.prof -memprofile=mem.prof

# Version info
version: ## Show version information
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"
