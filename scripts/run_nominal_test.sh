#!/bin/bash

# Nominal Load Test Script
# Target: 300 RPS for 3 minutes

echo "🧪 Starting Nominal Load Test (300 RPS, 3 minutes)..."

# Check if producer service is running
if ! curl -s http://localhost:8081/health > /dev/null; then
    echo "❌ Producer service is not running on localhost:8081"
    echo "Please start the system with: make up"
    exit 1
fi

echo "✅ Producer service is running"

# Generate timestamp and test directory name
TIMESTAMP=$(date +%Y-%m-%d_%H-%M-%S)
TEST_DIR="results/nominal_test_${TIMESTAMP}"

# Create test-specific directory
mkdir -p "${TEST_DIR}"

echo "📊 Test configuration:"
echo "  - Duration: 3 minutes"
echo "  - Users: 300"
echo "  - Target RPS: 300"
echo "  - Results directory: ${TEST_DIR}/"
echo ""

# Run locust test with reduced logging
locust -f load_test_nominal.py \
    --host=http://localhost:8081 \
    --users=300 \
    --spawn-rate=20 \
    --run-time=3m \
    --headless \
    --print-stats \
    --only-summary \
    --html="${TEST_DIR}/report.html" \
    --csv="${TEST_DIR}/stats" \
    --logfile="${TEST_DIR}/test.log" \
    --loglevel=WARNING

echo ""
echo "✅ Nominal load test completed!"
echo ""
echo "📈 Test Results Summary:"
echo "========================"

# Parse and display summary from CSV if available
if [ -f "${TEST_DIR}/stats_stats.csv" ]; then
    echo "📊 Performance Metrics:"
    echo "----------------------"
    
    # Extract key metrics from CSV
    tail -n +2 "${TEST_DIR}/stats_stats.csv" | while IFS=',' read -r type name request_count failure_count median_response_time average_response_time min_response_time max_response_time average_content_size requests_per_second failures_per_second p50 p66 p75 p80 p90 p95 p98 p99 p999 p9999; do
        if [ "$type" = "Aggregated" ]; then
            echo "  Total Requests: $request_count"
            echo "  Failed Requests: $failure_count"
            echo "  Success Rate: $(echo "scale=2; (($request_count - $failure_count) * 100) / $request_count" | bc -l)%"
            echo "  Average RPS: $requests_per_second"
            echo "  Average Response Time: ${average_response_time}ms"
            echo "  P95 Response Time: ${p95}ms"
            echo "  P99 Response Time: ${p99}ms"
        fi
    done
fi

echo ""
echo "📁 Test results saved in: ${TEST_DIR}/"
echo "  - HTML Report: ${TEST_DIR}/report.html"
echo "  - CSV Data: ${TEST_DIR}/stats_stats.csv"
echo "  - Log File: ${TEST_DIR}/test.log"
echo ""
echo "🌐 Open HTML report: file://$(pwd)/${TEST_DIR}/report.html" 