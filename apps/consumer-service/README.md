# Consumer Service

Микросервис для потребления и обработки событий из Apache Kafka. Часть системы event-driven архитектуры.

## Возможности

- **Потребление событий из Kafka** с поддержкой батчинга
- **Обработка различных типов событий** (user_created, order_placed, payment_processed)
- **Retry логика** с exponential backoff
- **Метрики Prometheus** для мониторинга
- **Health checks** и readiness probes
- **Graceful shutdown** с настраиваемыми таймаутами
- **Структурированное логирование** с JSON форматом
- **Конфигурация через переменные окружения**

## Архитектура

Сервис построен по принципам Clean Architecture:

```
cmd/server/          # Точка входа приложения
internal/
├── domain/          # Доменные модели и интерфейсы
├── usecase/         # Бизнес-логика и обработчики событий
├── infrastructure/  # Внешние зависимости
│   ├── kafka/       # Kafka consumer
│   ├── metrics/     # Prometheus метрики
│   └── logging/     # Логирование
└── config/          # Конфигурация
```

## Быстрый старт

### Требования

- Go 1.21+
- Apache Kafka
- Docker (опционально)

### Установка

```bash
# Клонирование репозитория
git clone <repository-url>
cd apps/consumer-service

# Установка зависимостей
make deps

# Сборка
make build
```

### Запуск

```bash
# Локальный запуск
make run

# Запуск в development режиме
make run-dev

# Запуск с Docker
make docker-build
make docker-run
```

## Конфигурация

Сервис конфигурируется через переменные окружения:

### Основные настройки

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `APP_NAME` | Название приложения | `consumer-service` |
| `APP_VERSION` | Версия приложения | `1.0.0` |
| `APP_ENV` | Окружение (development/staging/production) | `development` |
| `APP_DEBUG` | Режим отладки | `false` |

### HTTP сервер

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `SERVER_ADDRESS` | Адрес HTTP сервера | `:8080` |
| `SERVER_READ_TIMEOUT` | Таймаут чтения | `15s` |
| `SERVER_WRITE_TIMEOUT` | Таймаут записи | `15s` |
| `SERVER_IDLE_TIMEOUT` | Таймаут простоя | `60s` |

### Kafka

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `KAFKA_BROKER_LIST` | Список брокеров Kafka | `localhost:9092` |
| `KAFKA_TOPIC` | Топик для потребления | `events` |
| `KAFKA_GROUP_ID` | ID группы потребителей | `consumer-service` |
| `KAFKA_MIN_BYTES` | Минимальный размер сообщения | `1` |
| `KAFKA_MAX_BYTES` | Максимальный размер сообщения | `10485760` (10MB) |
| `KAFKA_MAX_WAIT` | Максимальное время ожидания | `1s` |
| `KAFKA_START_OFFSET` | Начальный offset (-2=earliest, -1=latest) | `-1` |
| `KAFKA_COMMIT_INTERVAL` | Интервал коммита | `1s` |

### Consumer

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `CONSUMER_WORKER_COUNT` | Количество воркеров | `5` |
| `CONSUMER_BATCH_SIZE` | Размер батча | `100` |
| `CONSUMER_PROCESS_TIMEOUT` | Таймаут обработки | `30s` |
| `CONSUMER_RETRY_ATTEMPTS` | Количество попыток повтора | `3` |
| `CONSUMER_RETRY_DELAY` | Задержка между попытками | `1s` |
| `CONSUMER_MAX_CONCURRENCY` | Максимальная конкурентность | `10` |
| `CONSUMER_SHUTDOWN_TIMEOUT` | Таймаут graceful shutdown | `30s` |

### Метрики

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `METRICS_ENABLED` | Включить метрики | `true` |
| `METRICS_PORT` | Порт для метрик | `:9090` |
| `METRICS_PATH` | Путь для метрик | `/metrics` |
| `METRICS_NAMESPACE` | Namespace для метрик | `consumer` |
| `METRICS_SUBSYSTEM` | Subsystem для метрик | `service` |

### Логирование

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `LOG_LEVEL` | Уровень логирования (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | Формат логов (json/text) | `json` |
| `LOG_OUTPUT` | Вывод логов (stdout/stderr/file) | `stdout` |
| `LOG_FILENAME` | Имя файла для логов | `consumer-service.log` |

## API

### Health Checks

#### GET /health
Проверка здоровья сервиса.

**Ответ:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "consumer-service",
  "version": "1.0.0"
}
```

#### GET /ready
Проверка готовности сервиса.

**Ответ:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-01T12:00:00Z",
  "details": {
    "consumer": {
      "messages_consumed": 1000,
      "errors": 0,
      "last_message": "2024-01-01T12:00:00Z"
    },
    "processor": {
      "events_processed": 1000,
      "events_failed": 0,
      "processing_rate": 10.5
    }
  }
}
```

#### GET /stats
Статистика работы сервиса.

**Ответ:**
```json
{
  "consumer": {
    "messages_consumed": 1000,
    "bytes_consumed": 50000,
    "errors": 0,
    "last_message_time": "2024-01-01T12:00:00Z",
    "lag": 0
  },
  "processor": {
    "events_processed": 1000,
    "events_failed": 0,
    "processing_rate": 10.5,
    "average_latency": "15ms",
    "events_by_type": {
      "user_created": 300,
      "order_placed": 400,
      "payment_processed": 300
    },
    "active_workers": 5,
    "last_processed_time": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Метрики (Prometheus)

#### GET /metrics
Метрики в формате Prometheus.

**Основные метрики:**
- `consumer_service_events_consumed_total` - Общее количество потребленных событий
- `consumer_service_events_failed_total` - Общее количество неудачных событий
- `consumer_service_processing_duration_seconds` - Время обработки событий
- `consumer_service_batch_size` - Размер обработанных батчей
- `consumer_service_active_workers` - Количество активных воркеров
- `consumer_service_kafka_lag` - Lag Kafka consumer
- `consumer_service_kafka_offset` - Текущий offset Kafka
- `consumer_service_retry_attempts_total` - Количество попыток повтора
- `consumer_service_throughput_events_per_second` - Пропускная способность

## Обработка событий

Сервис обрабатывает следующие типы событий:

### user_created
Событие создания пользователя.

**Пример:**
```json
{
  "id": "event-123",
  "type": "user_created",
  "data": "{\"user_id\":\"user-456\",\"email\":\"user@example.com\"}",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0",
  "source": "user-service"
}
```

**Обработка:**
- Отправка welcome email
- Создание профиля пользователя
- Инициализация настроек по умолчанию

### order_placed
Событие размещения заказа.

**Пример:**
```json
{
  "id": "event-124",
  "type": "order_placed",
  "data": "{\"order_id\":\"order-789\",\"user_id\":\"user-456\",\"amount\":100.00}",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0",
  "source": "order-service"
}
```

**Обработка:**
- Резервирование товаров на складе
- Расчет стоимости доставки
- Отправка уведомления продавцу
- Обновление аналитики продаж

### payment_processed
Событие обработки платежа.

**Пример:**
```json
{
  "id": "event-125",
  "type": "payment_processed",
  "data": "{\"payment_id\":\"payment-321\",\"order_id\":\"order-789\",\"amount\":100.00}",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0",
  "source": "payment-service"
}
```

**Обработка:**
- Подтверждение заказа
- Отправка чека клиенту
- Обновление статуса заказа
- Запуск процесса доставки

## Разработка

### Команды Make

```bash
# Показать все доступные команды
make help

# Разработка
make build          # Сборка приложения
make run            # Запуск приложения
make run-dev        # Запуск в development режиме
make test           # Запуск тестов
make test-race      # Тесты с проверкой race conditions
make coverage       # Генерация отчета покрытия
make benchmark      # Запуск бенчмарков

# Качество кода
make lint           # Линтинг
make fmt            # Форматирование
make vet            # Go vet
make check          # Все проверки

# Docker
make docker-build   # Сборка Docker образа
make docker-run     # Запуск в Docker
make docker-push    # Публикация образа

# Мониторинг
make health         # Проверка здоровья
make ready          # Проверка готовности
make stats          # Статистика
make metrics        # Метрики Prometheus

# Нагрузочное тестирование
make load-test      # Нагрузочный тест

# Очистка
make clean          # Очистка артефактов сборки
make clean-docker   # Очистка Docker образов
```

### Тестирование

```bash
# Запуск всех тестов
make test

# Тесты с race detection
make test-race

# Генерация отчета покрытия
make coverage

# Бенчмарки
make benchmark
```

### Линтинг

```bash
# Установка golangci-lint
make dev-setup

# Запуск линтера
make lint

# Форматирование кода
make fmt
```

## Мониторинг и алертинг

### Метрики для мониторинга

1. **Пропускная способность:**
   - `consumer_service_events_consumed_total`
   - `consumer_service_throughput_events_per_second`

2. **Ошибки:**
   - `consumer_service_events_failed_total`
   - `consumer_service_retry_attempts_total`

3. **Производительность:**
   - `consumer_service_processing_duration_seconds`
   - `consumer_service_kafka_lag`

4. **Ресурсы:**
   - `consumer_service_active_workers`
   - `consumer_service_batch_size`

### Рекомендуемые алерты

1. **Высокий lag Kafka** (> 1000 сообщений)
2. **Высокий процент ошибок** (> 5%)
3. **Низкая пропускная способность** (< 1 событие/сек в течение 5 минут)
4. **Сервис недоступен** (health check fails)

## Docker

### Сборка образа

```bash
make docker-build
```

### Запуск контейнера

```bash
# Простой запуск
make docker-run

# Запуск с кастомными настройками
docker run --rm -p 8080:8080 -p 9090:9090 \
  -e KAFKA_BROKER_LIST=kafka:9092 \
  -e LOG_LEVEL=debug \
  consumer-service:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  consumer-service:
    image: consumer-service:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - KAFKA_BROKER_LIST=kafka:9092
      - KAFKA_TOPIC=events
      - LOG_LEVEL=info
    depends_on:
      - kafka
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Производительность

### Рекомендуемые настройки

**Для высокой пропускной способности:**
```bash
CONSUMER_WORKER_COUNT=10
CONSUMER_BATCH_SIZE=500
CONSUMER_MAX_CONCURRENCY=20
KAFKA_MAX_BYTES=52428800  # 50MB
```

**Для низкой задержки:**
```bash
CONSUMER_WORKER_COUNT=5
CONSUMER_BATCH_SIZE=10
KAFKA_MAX_WAIT=100ms
```

### Оптимизация

1. **Увеличьте количество воркеров** для CPU-интенсивных задач
2. **Увеличьте размер батча** для I/O-интенсивных задач
3. **Настройте размер буфера Kafka** в зависимости от размера сообщений
4. **Используйте компрессию** для больших сообщений

## Безопасность

### Рекомендации

1. **Не логируйте чувствительные данные**
2. **Используйте HTTPS** в production
3. **Настройте аутентификацию Kafka** (SASL/SSL)
4. **Ограничьте доступ к метрикам**
5. **Регулярно обновляйте зависимости**

### Переменные окружения для безопасности

```bash
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SASL_USERNAME=username
KAFKA_SASL_PASSWORD=password
```

## Troubleshooting

### Частые проблемы

1. **Сервис не может подключиться к Kafka**
   - Проверьте `KAFKA_BROKER_LIST`
   - Убедитесь, что Kafka доступен

2. **Высокий lag**
   - Увеличьте количество воркеров
   - Оптимизируйте обработку событий
   - Проверьте производительность обработчиков

3. **Много ошибок обработки**
   - Проверьте логи для деталей
   - Убедитесь в корректности формата событий
   - Проверьте доступность внешних сервисов

4. **Медленная обработка**
   - Увеличьте размер батча
   - Оптимизируйте бизнес-логику
   - Проверьте метрики производительности

### Логи

```bash
# Просмотр логов
docker logs consumer-service

# Логи в реальном времени
docker logs -f consumer-service

# Фильтрация по уровню
docker logs consumer-service 2>&1 | grep ERROR
```

## Лицензия

MIT License

## Контакты

- Команда разработки: [team@example.com](mailto:team@example.com)
- Документация: [docs.example.com](https://docs.example.com)
- Issues: [GitHub Issues](https://github.com/example/consumer-service/issues) 