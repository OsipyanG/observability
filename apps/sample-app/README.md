# Sample Event Service

Простое HTTP веб-приложение на Go, которое публикует события в Apache Kafka и экспортирует метрики для Prometheus.

## Архитектура

Проект следует принципам Clean Architecture:

```
cmd/server/          # Точка входа приложения
internal/
├── config/          # Конфигурация
├── domain/          # Доменные модели и интерфейсы
├── usecase/         # Бизнес-логика (use cases)
├── infrastructure/  # Внешние зависимости (Kafka, Prometheus)
└── delivery/http/   # HTTP транспортный слой
    ├── handlers/    # HTTP обработчики
    └── middleware/  # HTTP middleware
```

## API Endpoints

### События

- `POST /api/v1/events/user-created` - Создание события пользователя
- `POST /api/v1/events/order-placed` - Создание события заказа
- `POST /api/v1/events/payment-processed` - Создание события платежа

### Служебные

- `GET /health` - Проверка здоровья сервиса
- `GET /metrics` - Prometheus метрики

## Формат запроса

```json
{
  "data": "Описание события"
}
```

## Формат ответа

```json
{
  "status": "success",
  "message": "Event sent to Kafka",
  "event": {
    "id": "user_created_20240101120000_abc123",
    "type": "user_created",
    "data": "New user has been created",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## Метрики

- `http_requests_total` - Общее количество HTTP запросов
- `http_request_duration_seconds` - Длительность HTTP запросов
- `kafka_messages_total` - Количество сообщений отправленных в Kafka

## Переменные окружения

- `SERVER_ADDRESS` - Адрес сервера (по умолчанию `:8080`)
- `KAFKA_BROKER_LIST` - Список Kafka брокеров (по умолчанию `localhost:9092`)
- `KAFKA_TOPIC` - Kafka топик (по умолчанию `events`)
- `SERVER_READ_TIMEOUT` - Таймаут чтения (по умолчанию `15s`)
- `SERVER_WRITE_TIMEOUT` - Таймаут записи (по умолчанию `15s`)
- `SERVER_IDLE_TIMEOUT` - Таймаут простоя (по умолчанию `60s`)

## Запуск

### Локально
```bash
# Сборка и запуск
make run

# Или вручную
go build -o bin/sample-app ./cmd/server
./bin/sample-app
```

### Docker
```bash
# Сборка и запуск в Docker
make docker-run

# Или вручную
docker build -t sample-app .
docker run -p 8080:8080 sample-app
```

### С полной инфраструктурой
```bash
# Запуск Kafka, Prometheus, Grafana и приложения
make compose-up

# Остановка
make compose-down
```

## Разработка

```bash
# Форматирование кода
make fmt

# Запуск тестов
make test

# Тесты с покрытием
make test-coverage

# Линтинг (требует golangci-lint)
make lint

# Помощь
make help
```

## Примеры использования

```bash
# Создание события пользователя
curl -X POST http://localhost:8080/api/v1/events/user-created \
  -H "Content-Type: application/json" \
  -d '{"data": "User John created"}'

# Проверка здоровья
curl http://localhost:8080/health

# Просмотр метрик
curl http://localhost:8080/metrics
```
