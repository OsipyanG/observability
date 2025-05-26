.PHONY: help up down restart logs clean build-app test-app

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

# Переменные
COMPOSE_FILE=infrastructure/docker-compose.yaml
PROJECT_NAME=diploma
SCRIPTS_DIR=scripts

help: ## Показать справку
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

up: ## Запустить всю инфраструктуру
	@echo "$(GREEN)Запуск инфраструктуры...$(NC)"
	cd infrastructure && docker-compose up -d
	@echo "$(GREEN)Инфраструктура запущена!$(NC)"
	@echo "$(YELLOW)Доступные сервисы:$(NC)"
	@echo "  - Kafka UI:      http://localhost:8080"
	@echo "  - Producer Service:    http://localhost:9091/metrics"
	@echo "  - Consumer Service:    http://localhost:9094/metrics"
	@echo "  - Prometheus:    http://localhost:9090"
	@echo "  - Grafana:       http://localhost:3000 (admin/admin)"
	@echo "  - Alertmanager:  http://localhost:9093"

clean: ## Очистить все данные (volumes, images, networks)
	@echo "$(RED)Очистка всех данных...$(NC)"
	cd infrastructure && docker-compose down -v --remove-orphans
	docker system prune -f
	docker volume prune -f


down: ## Остановить всю инфраструктуру
	@echo "$(YELLOW)Остановка инфраструктуры...$(NC)"
	cd infrastructure && docker-compose down

restart: down up ## Перезапустить инфраструктуру

rebuild: clean up 

logs: ## Показать логи всех сервисов
	cd infrastructure && docker-compose logs -f

status: ## Показать статус сервисов
	@echo "$(GREEN)Статус сервисов:$(NC)"
	cd infrastructure && docker-compose ps

build-app: ## Пересобрать только приложение
	@echo "$(GREEN)Пересборка приложения...$(NC)"
	cd infrastructure && docker-compose build producer-service consumer-service

restart-app: ## Перезапустить только приложение
	@echo "$(YELLOW)Перезапуск приложения...$(NC)"
	cd infrastructure && docker-compose restart sample-app

dev-setup: ## Первоначальная настройка для разработки
	@echo "$(GREEN)Настройка среды разработки...$(NC)"
	@chmod +x $(SCRIPTS_DIR)/dev-setup.sh
	@$(SCRIPTS_DIR)/dev-setup.sh

monitor: ## Открыть все мониторинговые интерфейсы
	@echo "$(GREEN)Открытие мониторинговых интерфейсов...$(NC)"
	@command -v open >/dev/null 2>&1 && { \
		open http://localhost:8080 & \
		open http://localhost:8081/health & \
		open http://localhost:9090 & \
		open http://localhost:3000 & \
		echo "$(GREEN)Интерфейсы открыты в браузере$(NC)"; \
	} || echo "$(YELLOW)Команда 'open' недоступна. Откройте URL вручную.$(NC)"

load-test: ## Запустить нагрузочное тестирование (200 RPS, 5 мин)
	@echo "$(GREEN)Запуск оптимизированного нагрузочного тестирования...$(NC)"
	@chmod +x $(SCRIPTS_DIR)/fast_load_test.sh
	@$(SCRIPTS_DIR)/fast_load_test.sh


install-load-deps: ## Установить зависимости для нагрузочного тестирования
	@echo "$(GREEN)Установка зависимостей для нагрузочного тестирования...$(NC)"
	@if command -v pipx >/dev/null 2>&1; then \
		echo "$(YELLOW)Используем pipx для установки locust...$(NC)"; \
		pipx install locust; \
	elif [[ "$(shell uname)" == "Darwin" ]]; then \
		echo "$(YELLOW)macOS обнаружена, используем --user флаг...$(NC)"; \
		pip3 install --user -r $(SCRIPTS_DIR)/requirements.txt; \
	else \
		pip3 install -r $(SCRIPTS_DIR)/requirements.txt; \
	fi
