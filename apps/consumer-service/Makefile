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

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@go build $(LDFLAGS) -o server ./cmd/server
	@echo "Build completed: ./server"

.PHONY: run
run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/server

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

.PHONY: mod-download
mod-download: ## Download go modules
	@echo "Downloading go modules..."
	@go mod download

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f server
	@rm -f coverage.out coverage.html
	@go clean

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -p 9091:9091 \
		-e LOG_LEVEL=info \
		-e KAFKA_BROKER_LIST=localhost:9092 \
		$(DOCKER_IMAGE):latest

.PHONY: docker-push
docker-push: ## Push Docker image
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):latest

.PHONY: deps
deps: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest

.PHONY: check
check: fmt lint test ## Run all checks (format, lint, test)

.PHONY: dev
dev: ## Run in development mode with hot reload
	@echo "Starting development mode..."
	@air -c .air.toml

.PHONY: metrics
metrics: ## Show metrics endpoint
	@echo "Metrics available at: http://localhost:9091/metrics"
	@curl -s http://localhost:9091/metrics | head -20

.PHONY: logs
logs: ## Show application logs (if running in Docker)
	@docker logs -f $(shell docker ps -q --filter ancestor=$(DOCKER_IMAGE):latest)

.PHONY: env-example
env-example: ## Create example environment file
	@echo "Creating .env.example..."
	@cat > .env.example << 'EOF'
# Application
APP_NAME=consumer-service
APP_VERSION=1.0.0
APP_ENV=development
APP_DEBUG=false

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Kafka
KAFKA_BROKER_LIST=localhost:9092
KAFKA_TOPIC=events
KAFKA_GROUP_ID=consumer-service
KAFKA_CLIENT_ID=consumer-service
KAFKA_MIN_BYTES=10000
KAFKA_MAX_BYTES=10000000
KAFKA_MAX_WAIT=1s
KAFKA_COMMIT_INTERVAL=1s
KAFKA_START_OFFSET=latest
KAFKA_MAX_RETRIES=3
KAFKA_RETRY_BACKOFF=100ms

# Metrics
METRICS_ENABLED=true
METRICS_PORT=:9091
METRICS_PATH=/metrics
EOF
	@echo ".env.example created"

.PHONY: install
install: deps build ## Install the application

.PHONY: all
all: clean deps check build ## Run all tasks 