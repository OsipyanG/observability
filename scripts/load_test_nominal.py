#!/usr/bin/env python3
"""
Nominal load testing script for Diploma Project (300 RPS)
"""

import json
import random
import logging
from locust import HttpUser, task, between, events
from locust.runners import MasterRunner, WorkerRunner

# Reduce locust logging
logging.getLogger("locust").setLevel(logging.WARNING)

class NominalUser(HttpUser):
    """User behavior for nominal load testing"""
    
    host = "http://localhost:8081"
    wait_time = between(0.5, 1.0)  # Adjusted wait time for 300 RPS
    
    def on_start(self):
        """Called when a user starts"""
        self.client.get("/health")
    
    @task
    def create_user_event(self):
        """Create user events - main load"""
        payload = {
            "data": f"User {random.randint(1, 1000000)} registered",
            "timestamp": random.randint(1000000000, 9999999999)
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
        print("🧪 Starting nominal load test - Target: 300 RPS")
        print("�� Test will run for 3 minutes with 300 concurrent users")
    elif isinstance(environment.runner, WorkerRunner):
        print("Worker node started")

@events.test_start.add_listener
def on_test_start(environment, **_kwargs):
    """Called when test starts"""
    print("🚀 Load test started...")

@events.test_stop.add_listener
def on_test_stop(environment, **_kwargs):
    """Called when test stops"""
    print("🏁 Load test completed!")
    
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
        print(f"   Average RPS: {stats.total.current_rps:.2f}")
        print(f"   Average Response Time: {stats.total.avg_response_time:.2f}ms") 