# Multi-stage build для оптимизации размера образа
ARG GO_VERSION=1.24.2

# Стадия сборки
FROM golang:${GO_VERSION}-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем пользователя для безопасности
RUN adduser -D -g '' appuser

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download && go mod verify

# Копируем исходный код
COPY . .

# Собираем приложение с оптимизациями
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o producer-service \
    ./cmd/server

# Финальная стадия
FROM scratch

# Копируем CA сертификаты для HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копируем timezone данные
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копируем пользователя
COPY --from=builder /etc/passwd /etc/passwd

# Копируем скомпилированное приложение
COPY --from=builder /build/producer-service /producer-service

# Используем непривилегированного пользователя
USER appuser

# Открываем порты
EXPOSE 8080 9090

# Устанавливаем переменные окружения по умолчанию
ENV SERVER_ADDRESS=":8080" \
    METRICS_PORT=":9090" \
    LOG_LEVEL="info" \
    LOG_FORMAT="json" \
    APP_ENV="production"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/producer-service", "-health-check"]

# Запускаем приложение
ENTRYPOINT ["/producer-service"] 