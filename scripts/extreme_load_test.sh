#!/bin/bash

# Extreme Load Test Script - Maximum RPS for MacBook M3 Pro
set -e

echo "🔥 EXTREME Load Test - Maximum RPS"
echo "=================================="
echo "🖥️  Target: MacBook M3 Pro (18GB RAM)"
echo "🎯 Goal: Maximum possible RPS"
echo ""

# Проверяем системные требования
echo "🔍 Проверка системных требований..."

# Проверяем лимиты файловых дескрипторов
CURRENT_ULIMIT=$(ulimit -n)
if [ "$CURRENT_ULIMIT" -lt 65536 ]; then
    echo "⚠️  Низкий лимит файловых дескрипторов: $CURRENT_ULIMIT"
    echo "💡 Запустите: ./scripts/optimize_macos.sh"
    echo "❌ Или установите: ulimit -n 65536"
    exit 1
fi
echo "✅ File descriptors: $CURRENT_ULIMIT"

# Проверяем доступную память
AVAILABLE_MEMORY=$(sysctl -n hw.memsize)
AVAILABLE_GB=$((AVAILABLE_MEMORY / 1024 / 1024 / 1024))
echo "✅ Available memory: ${AVAILABLE_GB}GB"

if [ "$AVAILABLE_GB" -lt 16 ]; then
    echo "⚠️  Рекомендуется минимум 16GB RAM для максимального RPS"
fi

# Проверяем приложение
echo "🔍 Проверка приложения..."
if ! curl -s http://localhost:8081/health > /dev/null; then
    echo "❌ Приложение не запущено на http://localhost:8081"
    echo "💡 Запустите: make up"
    exit 1
fi
echo "✅ Приложение доступно"

# Проверяем locust
if ! command -v locust &> /dev/null; then
    echo "❌ Locust не найден"
    echo "💡 Установите: pipx install locust"
    exit 1
fi
echo "✅ Locust доступен"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOCUST_FILE="$SCRIPT_DIR/load_test_extreme.py"

if [[ ! -f "$LOCUST_FILE" ]]; then
    echo "❌ Файл нагрузочного тестирования не найден: $LOCUST_FILE"
    exit 1
fi

# Создаем директорию результатов
RESULTS_DIR="extreme_results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo ""
echo "🚀 EXTREME PERFORMANCE SETTINGS:"
echo "👥 Users: 20,000 (виртуальных пользователей)"
echo "⚡ Spawn rate: 2,000 users/sec"
echo "⏱️  Duration: 5 minutes"
echo "🎯 Expected RPS: 15,000-30,000+"
echo "📁 Results: $RESULTS_DIR"
echo ""

# Предупреждение
echo "⚠️  ВНИМАНИЕ: Экстремальная нагрузка!"
echo "   - Может нагреть MacBook"
echo "   - Может замедлить систему"
echo "   - Следите за температурой"
echo ""

read -p "🤔 Продолжить? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Тест отменен"
    exit 1
fi

echo "🔥 Запуск экстремального нагрузочного тестирования..."
echo "📊 Команда: locust --headless --users 20000 --spawn-rate 2000 --run-time 300s"

# Запускаем тест с максимальными настройками
locust \
    --headless \
    --users 20000 \
    --spawn-rate 2000 \
    --run-time 300s \
    --host http://localhost:8081 \
    --locustfile "$LOCUST_FILE" \
    --html "$RESULTS_DIR/extreme_report.html" \
    --csv "$RESULTS_DIR/extreme_results" \
    --only-summary \
    --logfile "$RESULTS_DIR/locust.log" \
    --loglevel INFO

LOCUST_EXIT_CODE=$?

echo ""
echo "🏁 Экстремальный тест завершен!"
echo "📁 Результаты: $RESULTS_DIR/"
echo "📊 HTML отчет: $RESULTS_DIR/extreme_report.html"
echo "📋 Логи: $RESULTS_DIR/locust.log"

# Показываем краткую сводку
if [[ -f "$RESULTS_DIR/extreme_results_stats.csv" ]]; then
    echo ""
    echo "🔥 РЕЗУЛЬТАТЫ ЭКСТРЕМАЛЬНОГО ТЕСТА:"
    echo "=================================="
    tail -n 1 "$RESULTS_DIR/extreme_results_stats.csv" | awk -F',' '
    {
        if ($1 == "Aggregated") {
            printf "  🎯 Total Requests: %s\n", $3
            printf "  ❌ Failed: %s (%.2f%%)\n", $4, ($4/$3)*100
            printf "  ⚡ Average Response: %sms\n", $6
            printf "  🚀 Peak RPS: %s\n", $10
            printf "  📈 95th percentile: %sms\n", $8
        }
    }'
    
    # Анализ результатов
    TOTAL_REQUESTS=$(tail -n 1 "$RESULTS_DIR/extreme_results_stats.csv" | cut -d',' -f3)
    RPS=$(tail -n 1 "$RESULTS_DIR/extreme_results_stats.csv" | cut -d',' -f10)
    
    echo ""
    echo "📊 АНАЛИЗ ПРОИЗВОДИТЕЛЬНОСТИ:"
    if (( $(echo "$RPS > 10000" | bc -l) )); then
        echo "🏆 ОТЛИЧНО! RPS > 10,000 - максимальная производительность"
    elif (( $(echo "$RPS > 5000" | bc -l) )); then
        echo "✅ ХОРОШО! RPS > 5,000 - высокая производительность"
    elif (( $(echo "$RPS > 1000" | bc -l) )); then
        echo "⚠️  СРЕДНЕ! RPS > 1,000 - есть место для оптимизации"
    else
        echo "❌ НИЗКО! RPS < 1,000 - требуется серьезная оптимизация"
    fi
    
else
    echo "⚠️  Файл результатов не найден"
    echo "📁 Доступные файлы:"
    ls -la "$RESULTS_DIR/" || echo "Нет файлов в директории результатов"
fi

echo ""
echo "💡 Для дальнейшей оптимизации:"
echo "   1. Настройте переменные окружения из scripts/performance.env"
echo "   2. Оптимизируйте Kafka настройки"
echo "   3. Используйте профилирование Go (pprof)"
echo "   4. Мониторьте метрики в Grafana"

exit $LOCUST_EXIT_CODE 