.PHONY: help up down build restart clean status
.PHONY: load-test-nominal load-test-extreme
.PHONY: build-app restart-app install-deps

# Colors
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
BLUE=\033[0;34m
NC=\033[0m

# Variables
COMPOSE_FILE=infrastructure/docker-compose.yaml
SCRIPTS_DIR=scripts

# =============================================================================
# Help
# =============================================================================

help: ## Показать справку по командам
	@echo "$(GREEN)=== Diploma Project ===$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =============================================================================
# Main Commands
# =============================================================================

up: ## Запуск всех сервисов
	@echo "$(GREEN)🚀 Запуск системы...$(NC)"
	cd infrastructure && docker-compose up -d --build
	@$(MAKE) status

down: ## Остановка всех сервисов
	@echo "$(YELLOW)🛑 Остановка системы...$(NC)"
	cd infrastructure && docker-compose down

build: ## Пересборка всех образов
	@echo "$(YELLOW)🔨 Пересборка образов...$(NC)"
	cd infrastructure && docker-compose build --no-cache

restart: down up ## Перезапуск всех сервисов

# =============================================================================
# Application Management
# =============================================================================

build-app: ## Пересборка приложений
	@echo "$(YELLOW)🔨 Пересборка приложений...$(NC)"
	cd infrastructure && docker-compose build producer-service consumer-service

restart-app: ## Перезапуск приложений
	@echo "$(YELLOW)🔄 Перезапуск приложений...$(NC)"
	cd infrastructure && docker-compose up -d --build producer-service consumer-service

# =============================================================================
# Load Testing
# =============================================================================

load-test-nominal: ## Номинальный тест (300 RPS, 3 мин)
	@echo "$(GREEN)🧪 Запуск номинального теста...$(NC)"
	cd $(SCRIPTS_DIR) && chmod +x run_nominal_test.sh && ./run_nominal_test.sh

load-test-extreme: ## Экстремальный тест (15000 RPS, 3 мин)
	@echo "$(RED)🚀 Запуск экстремального теста...$(NC)"
	cd $(SCRIPTS_DIR) && chmod +x run_extreme_test.sh && ./run_extreme_test.sh

# =============================================================================
# Monitoring & Health
# =============================================================================

status: ## Статус всех сервисов
	@echo "$(BLUE)📊 Статус сервисов:$(NC)"
	@cd infrastructure && docker-compose ps

# =============================================================================
# Dependencies & Utilities
# =============================================================================

install-deps: ## Установить зависимости для тестирования
	@echo "$(GREEN)📦 Установка зависимостей...$(NC)"
	@if command -v pipx >/dev/null 2>&1; then \
		pipx install locust; \
	else \
		pip3 install locust requests; \
	fi

clean: ## Очистка системы
	@echo "$(RED)🧹 Очистка...$(NC)"
	cd infrastructure && docker-compose down -v --remove-orphans
	docker system prune -f
