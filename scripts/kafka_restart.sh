#!/bin/bash

# Kafka Restart Script
# Останавливает Kafka контейнер через docker-compose, ждет 60 секунд, затем запускает снова

set -e

KAFKA_SERVICE="kafka"
KAFKA_CONTAINER="project-kafka"
WAIT_TIME=60
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/../infrastructure/docker-compose.yaml"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция логирования
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] ✓ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] ⚠ $1${NC}"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ✗ $1${NC}"
}

# Проверяем наличие docker-compose файла
check_compose_file() {
    if [ ! -f "${COMPOSE_FILE}" ]; then
        log_error "Docker-compose файл не найден: ${COMPOSE_FILE}"
        exit 1
    fi
}

# Проверяем, существует ли сервис в compose
check_service_exists() {
    if ! docker-compose -f "${COMPOSE_FILE}" config --services | grep -q "^${KAFKA_SERVICE}$"; then
        log_error "Сервис ${KAFKA_SERVICE} не найден в docker-compose.yaml"
        exit 1
    fi
}

# Получаем статус сервиса
get_service_status() {
    local status
    status=$(docker-compose -f "${COMPOSE_FILE}" ps -q "${KAFKA_SERVICE}" 2>/dev/null)
    
    if [ -n "$status" ]; then
        # Контейнер существует, проверяем его статус
        docker inspect --format '{{.State.Status}}' "$status" 2>/dev/null || echo "unknown"
    else
        echo "not_found"
    fi
}

# Ждем пока Kafka будет готов
wait_for_kafka_ready() {
    local max_attempts=30
    local attempt=1
    
    log "Ожидаем готовности Kafka..."
    
    while [ $attempt -le $max_attempts ]; do
        # Проверяем что контейнер запущен
        if [ "$(get_service_status)" = "running" ]; then
            # Проверяем доступность Kafka API
            if docker-compose -f "${COMPOSE_FILE}" exec -T "${KAFKA_SERVICE}" kafka-broker-api-versions --bootstrap-server localhost:9092 &>/dev/null; then
                log_success "Kafka готов к работе (попытка ${attempt}/${max_attempts})"
                return 0
            fi
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    log_error "Kafka не готов после ${max_attempts} попыток"
    return 1
}

# Проверяем health статус
check_kafka_health() {
    local health_status
    health_status=$(docker inspect "${KAFKA_CONTAINER}" --format='{{.State.Health.Status}}' 2>/dev/null || echo "no-healthcheck")
    
    if [ "$health_status" = "healthy" ]; then
        return 0
    elif [ "$health_status" = "no-healthcheck" ]; then
        # Если нет healthcheck, проверяем через API
        return 0
    else
        return 1
    fi
}

# Основная логика
main() {
    log "Начинаем процедуру перезапуска Kafka через docker-compose..."
    
    # Проверяем наличие compose файла
    check_compose_file
    
    # Проверяем существование сервиса
    check_service_exists
    
    # Получаем текущий статус
    initial_status=$(get_service_status)
    log "Текущий статус сервиса ${KAFKA_SERVICE}: ${initial_status}"
    
    # Останавливаем сервис
    log "Останавливаем сервис ${KAFKA_SERVICE} через docker-compose..."
    if docker-compose -f "${COMPOSE_FILE}" stop "${KAFKA_SERVICE}"; then
        log_success "Сервис остановлен"
    else
        log_error "Ошибка при остановке сервиса"
        exit 1
    fi
    
    # Проверяем что действительно остановился
    log "Проверяем статус остановки..."
    sleep 2
    stopped_status=$(get_service_status)
    log "Статус после остановки: ${stopped_status}"
    
    # Ждем указанное время
    log "Ожидаем ${WAIT_TIME} секунд перед запуском..."
    for ((i=WAIT_TIME; i>0; i--)); do
        printf "\r${YELLOW}Осталось: %02d секунд${NC}" $i
        sleep 1
    done
    printf "\n"
    
    # Запускаем сервис
    log "Запускаем сервис ${KAFKA_SERVICE} через docker-compose..."
    if docker-compose -f "${COMPOSE_FILE}" start "${KAFKA_SERVICE}"; then
        log_success "Сервис запущен"
    else
        log_error "Ошибка при запуске сервиса"
        exit 1
    fi
    
    # Ждем готовности Kafka
    if wait_for_kafka_ready; then
        log_success "Kafka успешно перезапущен и готов к работе!"
        
        # Показываем дополнительную информацию
        log "Проверяем health статус..."
        if check_kafka_health; then
            log_success "Health check: OK"
        else
            log_warning "Health check: не прошел или не настроен"
        fi
        
    else
        log_warning "Kafka запущен, но может быть еще не готов"
        log "Попробуйте подождать еще немного или проверьте логи:"
        log "docker-compose -f ${COMPOSE_FILE} logs ${KAFKA_SERVICE}"
    fi
    
    # Показываем финальный статус
    final_status=$(get_service_status)
    log "Финальный статус сервиса: ${final_status}"
    
    # Показываем статус всех связанных сервисов
    log "Статус всех сервисов:"
    docker-compose -f "${COMPOSE_FILE}" ps
}

# Обработка сигналов
trap 'log_error "Скрипт прерван пользователем"; exit 130' INT TERM

# Проверяем наличие необходимых команд
check_dependencies() {
    local missing_deps=()
    
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        missing_deps+=("docker-compose")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Отсутствуют необходимые зависимости: ${missing_deps[*]}"
        exit 1
    fi
}

# Проверяем зависимости
check_dependencies

# Запускаем основную логику
main "$@" 