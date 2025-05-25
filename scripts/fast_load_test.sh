#!/bin/bash

# Fast Load Test Script - 200 RPS for 5 minutes
# Removed set -e to prevent early exit

echo "🚀 Fast Load Test (200 RPS, 5 minutes)"
echo "======================================"

# Check if app is running
echo "🔍 Checking if app is running..."
if ! curl -s http://localhost:8081/health > /dev/null; then
    echo "❌ App not running at http://localhost:8081"
    echo "💡 Start with: make up"
    exit 1
fi

# Check if locust is available
echo "🔍 Checking if locust is available..."
if ! command -v locust &> /dev/null; then
    echo "❌ Locust not found"
    echo "💡 Install with: pipx install locust"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "📁 Script directory: $SCRIPT_DIR"

# Check if locust file exists
LOCUST_FILE="$SCRIPT_DIR/load_test_optimized.py"
if [[ ! -f "$LOCUST_FILE" ]]; then
    echo "❌ Locust file not found: $LOCUST_FILE"
    exit 1
fi
echo "📄 Locust file: $LOCUST_FILE"

echo "✅ App is running"
echo "🎯 Target: 200 RPS for 5 minutes"
echo "👥 Users: 200 (optimized)"
echo "⚡ Spawn rate: 20 users/sec"
echo ""

# Create results directory
RESULTS_DIR="results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"
echo "📁 Results directory: $RESULTS_DIR"

echo "📊 Starting load test..."
echo "🔧 Command: locust --headless --users 200 --spawn-rate 20 --run-time 300s --host http://localhost:8081 --locustfile $LOCUST_FILE"

# Run optimized test
locust \
    --headless \
    --users 200 \
    --spawn-rate 20 \
    --run-time 300s \
    --host http://localhost:8081 \
    --locustfile "$LOCUST_FILE" \
    --html "$RESULTS_DIR/report.html" \
    --csv "$RESULTS_DIR/results" \
    --only-summary

LOCUST_EXIT_CODE=$?
echo "🔍 Locust exit code: $LOCUST_EXIT_CODE"

echo ""
echo "✅ Load test completed!"
echo "📁 Results: $RESULTS_DIR/"
echo "📊 HTML Report: $RESULTS_DIR/report.html"

# Show quick summary
if [[ -f "$RESULTS_DIR/results_stats.csv" ]]; then
    echo ""
    echo "📈 Quick Summary:"
    tail -n 1 "$RESULTS_DIR/results_stats.csv" | awk -F',' '
    {
        if ($1 == "Aggregated") {
            printf "  Total Requests: %s\n", $3
            printf "  Failed: %s\n", $4
            printf "  Avg Response: %sms\n", $6
            printf "  RPS: %s\n", $10
        }
    }'
else
    echo "⚠️  Results file not found: $RESULTS_DIR/results_stats.csv"
    echo "📁 Available files:"
    ls -la "$RESULTS_DIR/" || echo "No files in results directory"
fi