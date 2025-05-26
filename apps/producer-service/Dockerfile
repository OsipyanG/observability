# Multi-stage build для оптимизации размера образа
ARG GO_VERSION=1.24.2

# Стадия сборки
FROM golang:${GO_VERSION}-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем исходный код
COPY . .

RUN go mod tidy

# Собираем приложение с оптимизациями
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o producer-service \
    ./cmd/server

# Финальная стадия
FROM scratch

# Копируем скомпилированное приложение
COPY --from=builder /build/producer-service /producer-service

# Открываем порты
EXPOSE 8080 9090

ENV SERVER_ADDRESS=":8080" \
    METRICS_PORT=":9090" \
    LOG_LEVEL="info" \
    LOG_FORMAT="json" \
    APP_ENV="production"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/producer-service", "-health-check"]

ENTRYPOINT ["/producer-service"] 