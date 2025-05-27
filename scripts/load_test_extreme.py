#!/usr/bin/env python3
"""
Extreme Load Testing Script for Maximum RPS
Optimized for MacBook M3 Pro with 18GB RAM
"""

import json
import random
import string
from locust import HttpUser, task, between, events
from locust.runners import MasterRunner


class ExtremePerfUser(HttpUser):
    """Экстремально оптимизированный пользователь для максимального RPS"""
    
    # Минимальная задержка между запросами
    wait_time = between(0.001, 0.005)
    
    # Переиспользуем соединения
    connection_timeout = 5.0
    network_timeout = 5.0
    
    def on_start(self):
        """Инициализация - только один раз"""
        # Предгенерируем данные для избежания накладных расходов
        self.user_ids = [random.randint(1, 1000000) for _ in range(1000)]
        self.data_templates = [
            "User {} registered",
            "User {} updated profile", 
            "User {} logged in",
            "User {} made purchase",
            "User {} viewed product"
        ]
        self.counter = 0
    
    @task(10)
    def create_user_event_fast(self):
        """Быстрое создание событий - 95% запросов"""
        user_id = self.user_ids[self.counter % len(self.user_ids)]
        template = self.data_templates[self.counter % len(self.data_templates)]
        
        payload = {
            "data": template.format(user_id)
        }
        
        self.counter += 1
        
        # Отправляем без ожидания детального ответа
        with self.client.post("/api/v1/events/user", 
                             json=payload, 
                             catch_response=True,
                             timeout=2) as response:
            if response.status_code >= 400:
                response.failure(f"HTTP {response.status_code}")
    
    @task(1)
    def health_check(self):
        """Проверка здоровья - 5% запросов"""
        with self.client.get("/health", 
                           catch_response=True,
                           timeout=1) as response:
            if response.status_code != 200:
                response.failure(f"Health check failed: {response.status_code}")


@events.init.add_listener
def on_locust_init(environment, **kwargs):
    """Оптимизация при инициализации"""
    if isinstance(environment.runner, MasterRunner):
        print("🚀 Extreme Performance Mode Activated")
        print("📊 Target: Maximum RPS on MacBook M3 Pro")


# Конфигурация для максимальной производительности
class ExtremePerfConfig:
    """Конфигурация для экстремальной производительности"""
    
    # Рекомендуемые параметры для MacBook M3 Pro
    RECOMMENDED_USERS = 20000  # Максимальное количество виртуальных пользователей
    RECOMMENDED_SPAWN_RATE = 2000  # Быстрое создание пользователей
    
    @staticmethod
    def get_optimal_settings():
        return {
            "users": ExtremePerfConfig.RECOMMENDED_USERS,
            "spawn_rate": ExtremePerfConfig.RECOMMENDED_SPAWN_RATE,
            "run_time": "300s",  # 5 минут
            "host": "http://localhost:8081"
        }


if __name__ == "__main__":
    print("🔥 Extreme Performance Load Test Configuration")
    print("=" * 50)
    config = ExtremePerfConfig.get_optimal_settings()
    for key, value in config.items():
        print(f"  {key}: {value}")
    print("\n💡 Запустите с помощью extreme_load_test.sh") 