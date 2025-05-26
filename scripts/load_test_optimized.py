#!/usr/bin/env python3
"""
Optimized Locust load testing script for Diploma Project Sample App
Generates 200 RPS load for 5 minutes in headless mode
"""

import json
import random
from locust import HttpUser, task, between


class User(HttpUser):
    """Optimized user behavior for load testing"""
    
    # Reduced wait time for higher RPS
    wait_time = between(0.5, 1.5)
    
    # Base URL for the application
    host = "http://localhost:8081"
    
    def on_start(self):
        """Called when a user starts - simplified health check"""
        self.client.get("/health")
    
    @task(3)
    def create_user_event(self):
        """Create user events - 80% of requests"""
        payload = {
            "data": f"User {random.randint(1000, 9999)} registered"
        }
        self.client.post("/api/v1/events/user", json=payload)
    
    @task(3)
    def health_check(self):
        """Health check requests - 20% of requests"""
        self.client.get("/health") 