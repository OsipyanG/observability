#!/bin/bash

echo "🔬 Go Application Profiling Script"
echo "=================================="

# Проверяем что приложение запущено
if ! curl -s http://localhost:8081/health > /dev/null; then
    echo "❌ Приложение не запущено на http://localhost:8081"
    exit 1
fi

# Создаем директорию для профилей
PROFILE_DIR="profiles_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$PROFILE_DIR"

echo "📁 Профили сохраняются в: $PROFILE_DIR"
echo ""

# Функция для сбора профилей
collect_profiles() {
    local duration=${1:-30}
    local prefix=${2:-"profile"}
    
    echo "🔍 Сбор профилей в течение ${duration} секунд..."
    
    # CPU профиль
    echo "  📊 CPU профиль..."
    curl -s "http://localhost:9090/debug/pprof/profile?seconds=${duration}" > "$PROFILE_DIR/${prefix}_cpu.prof" &
    CPU_PID=$!
    
    # Memory профиль
    echo "  💾 Memory профиль..."
    curl -s "http://localhost:9090/debug/pprof/heap" > "$PROFILE_DIR/${prefix}_heap.prof"
    
    # Goroutines профиль
    echo "  🔄 Goroutines профиль..."
    curl -s "http://localhost:9090/debug/pprof/goroutine" > "$PROFILE_DIR/${prefix}_goroutine.prof"
    
    # Mutex профиль
    echo "  🔒 Mutex профиль..."
    curl -s "http://localhost:9090/debug/pprof/mutex" > "$PROFILE_DIR/${prefix}_mutex.prof"
    
    # Block профиль
    echo "  ⏸️  Block профиль..."
    curl -s "http://localhost:9090/debug/pprof/block" > "$PROFILE_DIR/${prefix}_block.prof"
    
    # Ждем завершения CPU профиля
    wait $CPU_PID
    
    echo "  ✅ Профили собраны"
}

# Сбор базовых профилей (до нагрузки)
echo "1️⃣  Сбор базовых профилей (без нагрузки)..."
collect_profiles 10 "baseline"

echo ""
echo "2️⃣  Запуск нагрузочного тестирования..."
echo "💡 В другом терминале запустите: ./scripts/extreme_load_test.sh"
echo "⏳ Ожидание начала нагрузки (30 сек)..."
sleep 30

# Сбор профилей под нагрузкой
echo ""
echo "3️⃣  Сбор профилей под нагрузкой..."
collect_profiles 60 "under_load"

echo ""
echo "4️⃣  Ожидание стабилизации нагрузки (30 сек)..."
sleep 30

# Сбор профилей на пике нагрузки
echo ""
echo "5️⃣  Сбор профилей на пике нагрузки..."
collect_profiles 30 "peak_load"

echo ""
echo "✅ Профилирование завершено!"
echo "📁 Профили сохранены в: $PROFILE_DIR"
echo ""

# Генерируем отчеты
echo "📊 Генерация отчетов..."

# Проверяем наличие go tool pprof
if command -v go &> /dev/null; then
    echo "  🔍 Анализ CPU профиля..."
    go tool pprof -text "$PROFILE_DIR/under_load_cpu.prof" > "$PROFILE_DIR/cpu_analysis.txt" 2>/dev/null || echo "    ⚠️  Ошибка анализа CPU"
    
    echo "  💾 Анализ Memory профиля..."
    go tool pprof -text "$PROFILE_DIR/under_load_heap.prof" > "$PROFILE_DIR/memory_analysis.txt" 2>/dev/null || echo "    ⚠️  Ошибка анализа Memory"
    
    echo "  🔄 Анализ Goroutines..."
    go tool pprof -text "$PROFILE_DIR/under_load_goroutine.prof" > "$PROFILE_DIR/goroutines_analysis.txt" 2>/dev/null || echo "    ⚠️  Ошибка анализа Goroutines"
else
    echo "  ⚠️  Go не найден, пропускаем автоматический анализ"
fi

echo ""
echo "📋 РЕЗУЛЬТАТЫ ПРОФИЛИРОВАНИЯ:"
echo "=============================="
echo "📁 Директория: $PROFILE_DIR"
echo ""
echo "📊 Профили:"
echo "  - baseline_*.prof     - базовые профили (без нагрузки)"
echo "  - under_load_*.prof   - профили под нагрузкой"
echo "  - peak_load_*.prof    - профили на пике нагрузки"
echo ""
echo "📈 Анализ:"
echo "  - cpu_analysis.txt      - анализ использования CPU"
echo "  - memory_analysis.txt   - анализ использования памяти"
echo "  - goroutines_analysis.txt - анализ горутин"
echo ""
echo "💡 Команды для детального анализа:"
echo "  go tool pprof $PROFILE_DIR/under_load_cpu.prof"
echo "  go tool pprof $PROFILE_DIR/under_load_heap.prof"
echo "  go tool pprof -http=:8080 $PROFILE_DIR/under_load_cpu.prof"
echo ""
echo "🔍 Веб-интерфейс для анализа:"
echo "  go tool pprof -http=:8080 $PROFILE_DIR/under_load_cpu.prof"
echo "  Откроется на http://localhost:8080" 