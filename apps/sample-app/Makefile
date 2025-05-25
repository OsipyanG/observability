.PHONY: build run test clean docker-build docker-run

# Переменные
APP_NAME=sample-app
DOCKER_IMAGE=sample-app:latest

# Сборка приложения
build:
	go build -o bin/$(APP_NAME) ./cmd/server

# Запуск приложения
run: build
	./bin/$(APP_NAME)

# Запуск тестов
test:
	go test -v ./...

# Запуск тестов с покрытием
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Линтинг
lint:
	golangci-lint run

# Форматирование кода
fmt:
	go fmt ./...
	goimports -w .

# Обновление зависимостей
deps:
	go mod tidy
	go mod download

# Очистка
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker сборка
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Docker запуск
docker-run: docker-build
	docker run -p 8080:8080 --env KAFKA_BROKER_LIST=host.docker.internal:9092 $(DOCKER_IMAGE)

# Запуск всей инфраструктуры
compose-up:
	cd .. && docker-compose up -d

# Остановка инфраструктуры
compose-down:
	cd .. && docker-compose down

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  build         - Сборка приложения"
	@echo "  run           - Запуск приложения"
	@echo "  test          - Запуск тестов"
	@echo "  test-coverage - Запуск тестов с покрытием"
	@echo "  lint          - Линтинг кода"
	@echo "  fmt           - Форматирование кода"
	@echo "  deps          - Обновление зависимостей"
	@echo "  clean         - Очистка артефактов"
	@echo "  docker-build  - Сборка Docker образа"
	@echo "  docker-run    - Запуск в Docker"
	@echo "  compose-up    - Запуск всей инфраструктуры"
	@echo "  compose-down  - Остановка инфраструктуры" 