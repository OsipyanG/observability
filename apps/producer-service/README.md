# Producer Service

Микросервис для публикации событий в Apache Kafka с использованием Clean Architecture и лучших практик Go разработки.

## Возможности

- 🚀 Публикация событий в Kafka с retry логикой
- 📊 Prometheus метрики для мониторинга
- 🔍 Структурированное логирование с Logrus
- 🛡️ Graceful shutdown и recovery middleware
- ✅ Валидация входных данных
- 🏗️ Clean Architecture с разделением слоев
- 🐳 Docker поддержка
- 📈 Health checks и readiness probes

## Архитектура

Сервис построен по принципам Clean Architecture:

```
cmd/server/          # Точка входа приложения
internal/
├── domain/          # Доменные модели и интерфейсы
├── usecase/         # Бизнес-логика
├── infrastructure/  # Внешние зависимости (Kafka, метрики)
├── delivery/        # HTTP handlers и middleware
└── config/          # Конфигурация
```

## API Endpoints

### События

- `POST /api/v1/events/user` - Создание события пользователя
- `POST /api/v1/events/order` - Создание события заказа  
- `POST /api/v1/events/payment` - Создание события платежа
- `GET /api/v1/events/stats` - Статистика событий

### Системные

- `GET /health` - Проверка здоровья сервиса
- `GET /ready` - Проверка готовности сервиса
- `GET /metrics` - Prometheus метрики (порт 9090)

## Конфигурация

Сервис настраивается через переменные окружения:

### Сервер
- `SERVER_ADDRESS` - Адрес сервера (по умолчанию: `:8080`)
- `SERVER_READ_TIMEOUT` - Таймаут чтения (по умолчанию: `15s`)
- `SERVER_WRITE_TIMEOUT` - Таймаут записи (по умолчанию: `15s`)
- `SERVER_SHUTDOWN_TIMEOUT` - Таймаут graceful shutdown (по умолчанию: `30s`)

### Kafka
- `KAFKA_BROKER_LIST` - Список брокеров Kafka (по умолчанию: `localhost:9092`)
- `KAFKA_TOPIC` - Топик для событий (по умолчанию: `events`)
- `KAFKA_CLIENT_ID` - ID клиента (по умолчанию: `producer-service`)
- `KAFKA_BATCH_SIZE` - Размер батча (по умолчанию: `100`)
- `KAFKA_MAX_RETRIES` - Максимальное количество повторов (по умолчанию: `3`)
- `KAFKA_COMPRESSION` - Тип сжатия: `none`, `gzip`, `snappy`, `lz4`, `zstd` (по умолчанию: `snappy`)

### Логирование
- `LOG_LEVEL` - Уровень логирования: `debug`, `info`, `warn`, `error` (по умолчанию: `info`)
- `LOG_FORMAT` - Формат логов: `json`, `text` (по умолчанию: `json`)

### Метрики
- `METRICS_ENABLED` - Включить метрики (по умолчанию: `true`)
- `METRICS_PORT` - Порт для метрик (по умолчанию: `:9090`)

### Приложение
- `APP_NAME` - Название приложения (по умолчанию: `producer-service`)
- `APP_VERSION` - Версия приложения (по умолчанию: `1.0.0`)
- `APP_ENV` - Окружение: `development`, `staging`, `production` (по умолчанию: `development`)

## Быстрый старт

### Локальная разработка

1. Установите зависимости:
```bash
make deps
```

2. Запустите Kafka и другие сервисы:
```bash
make compose-up
```

3. Запустите приложение:
```bash
make run
```

### Использование Docker

1. Соберите образ:
```bash
make docker-build
```

2. Запустите контейнер:
```bash
make docker-run
```

## Примеры использования

### Создание события пользователя

```bash
curl -X POST http://localhost:8080/api/v1/events/user \
  -H "Content-Type: application/json" \
  -d '{"data": "{\"user_id\": \"123\", \"email\": \"user@example.com\"}"}'
```

### Создание события заказа

```bash
curl -X POST http://localhost:8080/api/v1/events/order \
  -H "Content-Type: application/json" \
  -d '{"data": "{\"order_id\": \"456\", \"amount\": 100.50}"}'
```

### Получение статистики

```bash
curl http://localhost:8080/api/v1/events/stats
```

## Разработка

### Команды Makefile

```bash
make help           # Показать все доступные команды
make build          # Собрать приложение
make test           # Запустить тесты
make test-coverage  # Тесты с покрытием
make lint           # Запустить линтер
make fmt            # Форматировать код
make clean          # Очистить артефакты сборки
```

### Тестирование

```bash
# Запуск всех тестов
make test

# Тесты с покрытием
make test-coverage

# Бенчмарки
make bench

# Нагрузочное тестирование
make load-test
```

### Мониторинг

```bash
# Проверка здоровья
make health

# Проверка готовности
make ready

# Просмотр метрик
make metrics
```

## Метрики

Сервис экспортирует следующие Prometheus метрики:

### HTTP метрики
- `http_requests_total` - Общее количество HTTP запросов
- `http_request_duration_seconds` - Время выполнения HTTP запросов

### Producer метрики
- `producer_events_published_total` - Количество опубликованных событий
- `producer_events_failed_total` - Количество неудачных событий
- `producer_publish_duration_seconds` - Время публикации событий
- `producer_batch_size` - Размер батчей событий

## Логирование

Сервис использует структурированное логирование в формате JSON:

```json
{
  "level": "info",
  "msg": "Event published successfully",
  "event_id": "user_created_20240101120000_abc123",
  "event_type": "user_created",
  "duration": "15.2ms",
  "time": "2024-01-01T12:00:00Z"
}
```

## Развертывание

### Docker Compose

Сервис интегрирован в общий docker-compose.yaml проекта:

```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f producer-service
```

### Kubernetes

Для развертывания в Kubernetes используйте манифесты из директории `k8s/`.

## Безопасность

- Валидация всех входных данных
- Защита от CORS атак
- Безопасные HTTP заголовки
- Graceful shutdown для предотвращения потери данных
- Retry логика с exponential backoff

## Производительность

- Батчинг событий для оптимизации пропускной способности
- Сжатие сообщений Kafka (snappy по умолчанию)
- Connection pooling для HTTP клиентов
- Асинхронная обработка с goroutines

## Мониторинг и алертинг

Рекомендуемые алерты:

- Высокий процент ошибок публикации событий
- Превышение времени ответа HTTP запросов
- Недоступность Kafka брокеров
- Высокое потребление памяти/CPU

## Лицензия

MIT License
