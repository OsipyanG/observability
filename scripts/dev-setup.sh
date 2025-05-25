#!/bin/bash

set -e

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Настройка среды разработки...${NC}"

# Проверка зависимостей
echo -e "${YELLOW}📋 Проверка зависимостей...${NC}"

check_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo -e "${RED}❌ $1 не установлен!${NC}" >&2
        exit 1
    else
        echo -e "${GREEN}✅ $1 найден${NC}"
    fi
}

check_command docker
check_command docker-compose

# Опциональные зависимости
if command -v curl >/dev/null 2>&1; then
    echo -e "${GREEN}✅ curl найден${NC}"
else
    echo -e "${YELLOW}⚠️  curl не установлен (рекомендуется для тестирования)${NC}"
fi

if command -v jq >/dev/null 2>&1; then
    echo -e "${GREEN}✅ jq найден${NC}"
else
    echo -e "${YELLOW}⚠️  jq не установлен (рекомендуется для тестирования)${NC}"
fi

# Создание необходимых директорий
echo -e "${YELLOW}📁 Создание директорий...${NC}"
mkdir -p data/{kafka,zookeeper,prometheus,grafana}
mkdir -p logs

# Установка прав
echo -e "${YELLOW}🔐 Настройка прав доступа...${NC}"
chmod +x scripts/*.sh

echo -e "${GREEN}✅ Среда разработки готова!${NC}"
echo -e "${YELLOW}💡 Используйте 'make up' для запуска инфраструктуры${NC}" 