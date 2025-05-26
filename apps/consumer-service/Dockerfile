# Multi-stage build для оптимизации размера образа
FROM golang:1.21-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем пользователя для безопасности
RUN adduser -D -g '' appuser

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем go mod файлы для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка приложения с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o consumer-service ./cmd/server

# Финальный образ
FROM scratch

# Копируем CA сертификаты для HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копируем timezone данные
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копируем пользователя
COPY --from=builder /etc/passwd /etc/passwd

# Копируем скомпилированное приложение
COPY --from=builder /build/consumer-service /consumer-service

# Используем непривилегированного пользователя
USER appuser

# Открываем порты
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/consumer-service", "-health-check"]

# Запускаем приложение
ENTRYPOINT ["/consumer-service"] 