#!/usr/bin/env python3
"""
Extreme load testing script 
"""

import json
import random
import time
import logging
from locust import HttpUser, task, between, events
from locust.runners import MasterRunner, WorkerRunner
from locust import LoadTestShape

# Reduce locust logging
logging.getLogger("locust").setLevel(logging.WARNING)

class ExtremePacing(LoadTestShape):
    """Custom load shape for extreme testing - 3 minutes total"""
    stages = [
        {"duration": 30, "users": 500, "spawn_rate": 50},   # Разогрев - 30 сек
        {"duration": 60, "users": 2000, "spawn_rate": 100}, # Наращивание - 1 мин
        {"duration": 60, "users": 3000, "spawn_rate": 200}, # Пик нагрузки - 1 мин
        {"duration": 30, "users": 1000, "spawn_rate": 100}, # Плавное снижение - 30 сек
    ]

    def tick(self):
        run_time = self.get_run_time()
        
        for stage in self.stages:
            if run_time < stage["duration"]:
                return (stage["users"], stage["spawn_rate"])
            run_time -= stage["duration"]
        return None

class ExtremeUser(HttpUser):
    """User behavior for extreme load testing"""
    
    host = "http://localhost:8081"
    wait_time = between(0.0001, 0.0002)
    
    def on_start(self):
        """Called when a user starts"""
        self.client.get("/health")
    
    @task
    def create_user_event(self):
        """Create user events - основная нагрузка"""
        payload = {
            "data": f"User {random.randint(1, 1000000)} action",
            "timestamp": int(time.time())
        }
        with self.client.post("/api/v1/events/user", 
                            json=payload,
                            catch_response=True) as response:
            if response.status_code != 200:
                response.failure(f"Failed with status {response.status_code}")

@events.init.add_listener
def on_locust_init(environment, **_kwargs):
    """Configure the test on initialization"""
    if isinstance(environment.runner, MasterRunner):
        print("🚀 Starting extreme load test - Target: 15000 RPS")
        print("⚠️  WARNING: This test will generate extreme load!")
        print("📊 Test stages (3 minutes total):")
        print("   1. Warmup: 500 users (30 sec)")
        print("   2. Ramp-up: 2000 users (1 min)")
        print("   3. Peak load: 3000 users (1 min)")
        print("   4. Cool-down: 1000 users (30 sec)")
    elif isinstance(environment.runner, WorkerRunner):
        print("Worker node started")

@events.test_start.add_listener
def on_test_start(environment, **_kwargs):
    """Called when test starts"""
    print("🚀 Extreme load test started...")

@events.test_stop.add_listener
def on_test_stop(environment, **_kwargs):
    """Called when test stops"""
    print("🏁 Extreme load test completed!")
    
    # Print final summary
    stats = environment.stats
    total_requests = stats.total.num_requests
    total_failures = stats.total.num_failures
    
    if total_requests > 0:
        success_rate = ((total_requests - total_failures) / total_requests) * 100
        print(f"📈 Final Results:")
        print(f"   Total Requests: {total_requests:,}")
        print(f"   Failed Requests: {total_failures:,}")
        print(f"   Success Rate: {success_rate:.2f}%")
        print(f"   Peak RPS: {stats.total.max_rps:.2f}")
        print(f"   Average RPS: {stats.total.current_rps:.2f}")
        print(f"   Average Response Time: {stats.total.avg_response_time:.2f}ms")
        print(f"   P95 Response Time: {stats.total.get_response_time_percentile(0.95):.2f}ms")
        print(f"   P99 Response Time: {stats.total.get_response_time_percentile(0.99):.2f}ms") 