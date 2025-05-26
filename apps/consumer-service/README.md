# Consumer Service

Микросервис для потребления событий из Apache Kafka с метриками Prometheus.

## Описание

Consumer Service - это Go-приложение, которое:
- Потребляет события из Kafka топика
- Обрабатывает различные типы событий (создание, обновление, удаление пользователей)
- Собирает метрики производительности
- Предоставляет метрики через Prometheus endpoint

## Архитектура

Сервис построен по принципам Clean Architecture:

```
cmd/
  server/           # Точка входа приложения
internal/
  config/           # Конфигурация
  domain/           # Доменные модели
  usecase/          # Бизнес-логика
  infrastructure/
    kafka/          # Kafka consumer
    metrics/        # Prometheus метрики
```

## Метрики

Сервис собирает следующие метрики:

### Consumer метрики
- `consumer_events_consumed_total` - Общее количество потребленных событий
- `consumer_events_failed_total` - Количество неудачных событий
- `consumer_processing_duration_seconds` - Время обработки событий
- `consumer_lag` - Отставание consumer по партициям
- `consumer_commit_duration_seconds` - Время коммита offset
- `consumer_batch_size` - Размер batch сообщений
- `consumer_kafka_reader_stats` - Статистика Kafka reader

## Конфигурация

Конфигурация осуществляется через переменные окружения:

### Приложение
- `APP_NAME` - Имя приложения (по умолчанию: consumer-service)
- `APP_VERSION` - Версия приложения (по умолчанию: 1.0.0)
- `APP_ENV` - Окружение (development/staging/production)
- `APP_DEBUG` - Режим отладки (по умолчанию: false)

### Логирование
- `LOG_LEVEL` - Уровень логирования (debug/info/warn/error)
- `LOG_FORMAT` - Формат логов (json/text)

### Kafka
- `KAFKA_BROKER_LIST` - Список брокеров Kafka (по умолчанию: localhost:9092)
- `KAFKA_TOPIC` - Топик для потребления (по умолчанию: events)
- `KAFKA_GROUP_ID` - ID группы consumer (по умолчанию: consumer-service)
- `KAFKA_CLIENT_ID` - ID клиента (по умолчанию: consumer-service)
- `KAFKA_MIN_BYTES` - Минимальный размер batch (по умолчанию: 10000)
- `KAFKA_MAX_BYTES` - Максимальный размер batch (по умолчанию: 10000000)
- `KAFKA_MAX_WAIT` - Максимальное время ожидания (по умолчанию: 1s)
- `KAFKA_COMMIT_INTERVAL` - Интервал коммита (по умолчанию: 1s)
- `KAFKA_START_OFFSET` - Начальный offset (earliest/latest)
- `KAFKA_MAX_RETRIES` - Максимальное количество повторов (по умолчанию: 3)
- `KAFKA_RETRY_BACKOFF` - Задержка между повторами (по умолчанию: 100ms)

### Метрики
- `METRICS_ENABLED` - Включить метрики (по умолчанию: true)
- `METRICS_PORT` - Порт для метрик (по умолчанию: :9091)
- `METRICS_PATH` - Путь для метрик (по умолчанию: /metrics)

## Быстрый старт

### Локальная разработка

1. Установите зависимости:
```bash
make deps
```

2. Создайте файл конфигурации:
```bash
make env-example
cp .env.example .env
# Отредактируйте .env под ваши нужды
```

3. Запустите приложение:
```bash
make run
```

### Docker

1. Соберите образ:
```bash
make docker-build
```

2. Запустите контейнер:
```bash
make docker-run
```

## Команды Make

- `make help` - Показать справку
- `make build` - Собрать приложение
- `make run` - Запустить приложение
- `make test` - Запустить тесты
- `make lint` - Запустить линтер
- `make fmt` - Форматировать код
- `make docker-build` - Собрать Docker образ
- `make docker-run` - Запустить Docker контейнер
- `make metrics` - Показать метрики
- `make clean` - Очистить артефакты сборки

## Обработка событий

Сервис обрабатывает следующие типы событий:

### user.created
Событие создания пользователя:
```json
{
  "id": "event-uuid",
  "type": "user.created",
  "source": "user-service",
  "version": "1.0",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "user_id": "user-123",
    "username": "john_doe",
    "email": "john@example.com",
    "action": "create"
  }
}
```

### user.updated
Событие обновления пользователя:
```json
{
  "id": "event-uuid",
  "type": "user.updated",
  "source": "user-service",
  "version": "1.0",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "user_id": "user-123",
    "username": "john_doe_updated",
    "email": "john.updated@example.com",
    "action": "update"
  }
}
```

### user.deleted
Событие удаления пользователя:
```json
{
  "id": "event-uuid",
  "type": "user.deleted",
  "source": "user-service",
  "version": "1.0",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "user_id": "user-123",
    "action": "delete"
  }
}
```

## Мониторинг

### Метрики Prometheus

Метрики доступны по адресу: `http://localhost:9091/metrics`

### Health Check

Docker контейнер включает health check, который проверяет доступность метрик endpoint.

## Graceful Shutdown

Сервис поддерживает graceful shutdown при получении сигналов SIGINT или SIGTERM:
1. Останавливает потребление новых сообщений
2. Завершает обработку текущих сообщений
3. Закрывает соединения с Kafka
4. Останавливает метрики сервер

## Разработка

### Структура проекта

```
consumer-service/
├── cmd/
│   └── server/
│       └── main.go              # Точка входа
├── internal/
│   ├── config/
│   │   └── config.go            # Конфигурация
│   ├── domain/
│   │   └── event.go             # Доменные модели
│   ├── infrastructure/
│   │   ├── kafka/
│   │   │   └── consumer.go      # Kafka consumer
│   │   └── metrics/
│   │       └── consumer_metrics.go # Метрики
│   └── usecase/
│       └── event_processor.go   # Обработка событий
├── Dockerfile                   # Docker образ
├── Makefile                     # Команды сборки
├── go.mod                       # Go модули
└── README.md                    # Документация
```

### Добавление новых типов событий

1. Добавьте новый тип в `internal/domain/event.go`:
```go
const (
    NewEventType EventType = "new.event"
)
```

2. Добавьте обработчик в `internal/usecase/event_processor.go`:
```go
case domain.NewEventType:
    return p.processNewEvent(ctx, event)
```

3. Реализуйте метод обработки:
```go
func (p *EventProcessor) processNewEvent(ctx context.Context, event *domain.Event) error {
    // Логика обработки
    return nil
}
```

## Лицензия

MIT License 