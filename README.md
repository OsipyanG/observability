# Diploma Project - Event-Driven Microservices with Monitoring

Учебный проект, демонстрирующий современную архитектуру микросервисов с полным стеком мониторинга и наблюдаемости.

## 🏗️ Структура проекта

```tree
diploma/
├── apps/                           # Приложения
│   └── sample-app/                # Go HTTP API
│       ├── cmd/server/            # Точка входа
│       ├── internal/              # Внутренняя логика
│       ├── Dockerfile             # Контейнер приложения
│       ├── Makefile              # Команды разработки
│       └── README.md             # Документация приложения
├── infrastructure/                # Инфраструктура как код
│   ├── docker-compose.yaml       # Оркестрация сервисов
│   ├── monitoring/               # Конфигурация мониторинга
│   │   ├── prometheus.yml        # Настройки Prometheus
│   │   ├── alert.rules.yml       # Правила алертинга
│   │   └── alertmanager.yml      # Конфигурация Alertmanager
│   ├── grafana/                  # Конфигурация Grafana
│   │   ├── dashboards/           # Дашборды
│   │   └── provisioning/         # Автоматическая настройка
│   └── kafka/                    # Конфигурация Kafka (будущее)
├── scripts/                      # Скрипты автоматизации
│   ├── dev-setup.sh             # Настройка среды разработки
│   └── test-api.sh              # Тестирование API
├── docs/                        # Документация проекта
├── Makefile                     # Основные команды проекта
├── .gitignore                   # Исключения Git
└── README.md                    # Этот файл
```

## 🚀 Архитектура

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Sample App    │───▶│     Kafka       │───▶│   Monitoring    │
│   (Go/HTTP)     │    │   (Events)      │    │  (Prometheus)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kafka UI      │    │  Kafka Exporter │    │    Grafana      │
│  (Management)   │    │   (Metrics)     │    │ (Visualization) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 Компоненты

### Основные сервисы

- **Sample App** - Go HTTP API для создания событий
- **Apache Kafka** - Брокер сообщений для event streaming
- **Zookeeper** - Координация для Kafka

### Мониторинг и наблюдаемость

- **Prometheus** - Сбор и хранение метрик
- **Grafana** - Визуализация метрик и дашборды
- **Alertmanager** - Управление алертами
- **Kafka Exporter** - Экспорт метрик Kafka

### Инструменты разработки

- **Kafka UI** - Веб-интерфейс для управления Kafka

## 📋 Требования

- Docker 20.10+
- Docker Compose 2.0+
- Make (опционально)
- curl и jq (для тестирования)

## 🎯 Быстрый старт

### 1. Клонирование и настройка

```bash
git clone <repository>
cd diploma

# Первоначальная настройка среды
make dev-setup
```

### 2. Запуск инфраструктуры

```bash
# Запуск всех сервисов
make up

# Проверка статуса
make status
```

### 3. Доступ к сервисам

После запуска доступны следующие интерфейсы:

| Сервис       | URL                   | Описание               |
| ------------ | --------------------- | ---------------------- |
| Sample App   | http://localhost:8081 | REST API приложения    |
| Kafka UI     | http://localhost:8080 | Управление Kafka       |
| Prometheus   | http://localhost:9090 | Метрики и алерты       |
| Grafana      | http://localhost:3000 | Дашборды (admin/admin) |
| Alertmanager | http://localhost:9093 | Управление алертами    |

### 4. Тестирование

```bash
# Автоматическое тестирование API
make test-app

# Открыть все мониторинговые интерфейсы
make monitor
```

## 🛠️ Команды управления

```bash
# Основные команды
make up           # Запустить инфраструктуру
make down         # Остановить инфраструктуру
make restart      # Перезапустить
make status       # Статус сервисов
make logs         # Логи всех сервисов

# Разработка
make dev-setup    # Настройка среды разработки
make build-app    # Пересобрать приложение
make restart-app  # Перезапустить только приложение
make logs-app     # Логи приложения

# Тестирование
make test-app     # Тестирование API
make monitor      # Открыть все интерфейсы

# Kafka
make kafka-topics   # Список топиков
make kafka-consume  # Подписка на события

# Очистка
make clean        # Полная очистка
make clean-volumes # Очистка только данных

# Справка
make help         # Все доступные команды
```

## 📊 API Endpoints

### События

- `POST /api/v1/events/user-created` - Создание события пользователя
- `POST /api/v1/events/order-placed` - Создание события заказа
- `POST /api/v1/events/payment-processed` - Создание события платежа

### Служебные

- `GET /health` - Проверка здоровья
- `GET /metrics` - Prometheus метрики

### Формат запроса

```json
{
  "data": "Описание события"
}
```

### Формат ответа

```json
{
  "status": "success",
  "message": "Event sent to Kafka",
  "event": {
    "id": "user_created_20240101120000_abc123",
    "type": "user_created",
    "data": "New user registered",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## 📈 Метрики и мониторинг

### Доступные метрики

- `http_requests_total` - Количество HTTP запросов
- `http_request_duration_seconds` - Длительность запросов
- `kafka_messages_total` - Количество сообщений в Kafka
- Стандартные метрики Kafka (через kafka-exporter)

### Дашборды Grafana

Доступны готовые дашборды:

- **Overview** - Общий обзор системы
- **Application Monitoring** - Детальный мониторинг Go приложения
- **Kafka Monitoring** - Специализированный мониторинг Kafka
- **System Monitoring** - Системные метрики и инфраструктура

### Алерты

Настроены алерты на:

- Высокую задержку (p99 > 250ms)
- Ошибки в Kafka
- Недоступность сервисов

## 🔧 Конфигурация

### Структура конфигурации

- `infrastructure/monitoring/` - Конфигурация мониторинга
- `infrastructure/grafana/` - Настройки Grafana и дашборды
- `infrastructure/docker-compose.yaml` - Оркестрация сервисов

### Переменные окружения

```bash
# Kafka
KAFKA_BROKER_LIST=kafka:29092
KAFKA_TOPIC=events

# Server
SERVER_ADDRESS=:8080
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s
SERVER_IDLE_TIMEOUT=60s
```

### Volumes

Данные сохраняются в именованных volumes:

- `kafka-data` - Данные Kafka
- `zookeeper-data` - Данные Zookeeper
- `prometheus-data` - Метрики Prometheus
- `grafana-data` - Конфигурация Grafana

## 🏫 Учебные цели

Этот проект демонстрирует:

1. **Event-Driven Architecture** - Асинхронная обработка событий
2. **Microservices** - Разделение ответственности между сервисами
3. **Observability** - Полный стек мониторинга и логирования
4. **DevOps practices** - Контейнеризация, IaC, автоматизация
5. **Clean Architecture** - Правильная структура Go приложения

## 🐛 Troubleshooting

### Проблемы с запуском

```bash
# Проверка статуса
make status

# Логи конкретного сервиса
docker-compose logs kafka
docker-compose logs app

# Полная перезагрузка
make clean
make up
```

### Проблемы с Kafka

```bash
# Проверка топиков
make kafka-topics

# Ручное создание топика
docker-compose exec kafka kafka-topics \
  --create --topic events \
  --bootstrap-server localhost:9092 \
  --partitions 3 --replication-factor 1
```

## 📚 Дополнительные материалы

- [Go приложение](./apps/sample-app/README.md) - Подробности о приложении
- [Конфигурация мониторинга](./infrastructure/monitoring/) - Настройки Prometheus и Alertmanager
- [Дашборды Grafana](./infrastructure/grafana/dashboards/) - Готовые дашборды
- [Скрипты автоматизации](./scripts/) - Утилиты для разработки

## 🤝 Разработка

```bash
# Локальная разработка приложения
cd compose/sample-app
make run

# Форматирование кода
make fmt

# Тесты
make test

# Линтинг
make lint
```
