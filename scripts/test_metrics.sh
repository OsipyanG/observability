#!/bin/bash

# Скрипт для тестирования метрик системы
set -e

PRODUCER_URL="http://localhost:8081"
PROMETHEUS_URL="http://localhost:9090"
GRAFANA_URL="http://localhost:3000"

echo "🚀 Тестирование метрик системы..."

# Функция для создания событий
create_events() {
    echo "📝 Создание тестовых событий..."
    
    # Создаем события пользователей
    for i in {1..5}; do
        curl -s -X POST "$PRODUCER_URL/api/v1/events/user" \
            -H "Content-Type: application/json" \
            -d "{\"data\": \"test user event $i\"}" > /dev/null
        echo "✅ Создано событие пользователя $i"
        sleep 0.5
    done
    
    # Создаем события заказов
    for i in {1..3}; do
        curl -s -X POST "$PRODUCER_URL/api/v1/events/order" \
            -H "Content-Type: application/json" \
            -d "{\"data\": \"test order event $i\"}" > /dev/null
        echo "✅ Создано событие заказа $i"
        sleep 0.5
    done
    
    # Создаем события платежей
    for i in {1..2}; do
        curl -s -X POST "$PRODUCER_URL/api/v1/events/payment" \
            -H "Content-Type: application/json" \
            -d "{\"data\": \"test payment event $i\"}" > /dev/null
        echo "✅ Создано событие платежа $i"
        sleep 0.5
    done
}

# Функция для проверки метрик
check_metrics() {
    echo "📊 Проверка метрик..."
    
    # Проверяем producer метрики
    echo "🔍 Producer метрики:"
    curl -s http://localhost:8082/metrics | grep "producer_events_published_total" | head -5
    
    # Проверяем consumer метрики
    echo "🔍 Consumer метрики:"
    curl -s http://localhost:9091/metrics | grep "consumer_service_events_consumed_total" | head -5
    
    # Проверяем Prometheus
    echo "🔍 Prometheus targets:"
    curl -s "$PROMETHEUS_URL/api/v1/targets" | jq -r '.data.activeTargets[] | "\(.labels.job): \(.health)"'
}

# Функция для проверки доступности сервисов
check_services() {
    echo "🔍 Проверка доступности сервисов..."
    
    services=(
        "Producer:$PRODUCER_URL/health"
        "Prometheus:$PROMETHEUS_URL/-/healthy"
        "Grafana:$GRAFANA_URL/api/health"
    )
    
    for service in "${services[@]}"; do
        name=$(echo $service | cut -d: -f1)
        url=$(echo $service | cut -d: -f2-)
        
        if curl -s -f "$url" > /dev/null; then
            echo "✅ $name доступен"
        else
            echo "❌ $name недоступен"
        fi
    done
}

# Функция для отображения полезных ссылок
show_links() {
    echo "🔗 Полезные ссылки:"
    echo "   Grafana: $GRAFANA_URL (admin/admin)"
    echo "   Prometheus: $PROMETHEUS_URL"
    echo "   Kafka UI: http://localhost:8080"
    echo "   Producer API: $PRODUCER_URL/api/v1"
    echo ""
    echo "📊 Дашборды Grafana:"
    echo "   System Overview: $GRAFANA_URL/d/system-overview"
    echo "   Microservices: $GRAFANA_URL/d/microservices-dashboard"
}

# Основная логика
main() {
    echo "🎯 Начинаем тестирование..."
    
    # Проверяем доступность сервисов
    check_services
    echo ""
    
    # Создаем события
    create_events
    echo ""
    
    # Ждем обработки
    echo "⏳ Ожидание обработки событий (10 секунд)..."
    sleep 10
    
    # Проверяем метрики
    check_metrics
    echo ""
    
    # Показываем ссылки
    show_links
    
    echo "✅ Тестирование завершено!"
}

# Запускаем если скрипт вызван напрямую
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 