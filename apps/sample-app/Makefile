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
