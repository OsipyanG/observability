#!/bin/bash

# =============================================================================
# Full Observability Stack Setup Script
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="$PROJECT_ROOT/infrastructure/docker-compose.yaml"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if docker compose is available
check_docker_compose() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose is not installed"
        exit 1
    fi
    
    print_success "Docker Compose found: $DOCKER_COMPOSE_CMD"
}

# Function to validate configuration files
validate_configs() {
    print_status "Validating configuration files..."
    
    local configs=(
        "$PROJECT_ROOT/infrastructure/monitoring/loki-config.yaml"
        "$PROJECT_ROOT/infrastructure/monitoring/promtail-config.yaml"
        "$PROJECT_ROOT/infrastructure/monitoring/otel-collector-config.yaml"
        "$PROJECT_ROOT/infrastructure/monitoring/prometheus.yml"
        "$PROJECT_ROOT/infrastructure/grafana/provisioning/datasources/datasources.yml"
    )
    
    for config in "${configs[@]}"; do
        if [[ ! -f "$config" ]]; then
            print_error "Configuration file not found: $config"
            exit 1
        fi
        print_status "✓ Found: $(basename "$config")"
    done
    
    print_success "All configuration files validated"
}

# Function to stop existing services
stop_services() {
    print_status "Stopping existing services..."
    cd "$PROJECT_ROOT/infrastructure"
    
    if $DOCKER_COMPOSE_CMD ps | grep -q "Up"; then
        $DOCKER_COMPOSE_CMD down
        print_success "Services stopped"
    else
        print_status "No running services found"
    fi
}

# Function to start core infrastructure
start_infrastructure() {
    print_status "Starting core infrastructure..."
    cd "$PROJECT_ROOT/infrastructure"
    
    # Start in specific order to respect dependencies
    print_status "Starting Kafka infrastructure..."
    $DOCKER_COMPOSE_CMD up -d zookeeper kafka kafka-init
    
    # Wait for Kafka to be ready
    print_status "Waiting for Kafka to be ready..."
    sleep 30
    
    print_success "Kafka infrastructure started"
}

# Function to start observability stack
start_observability() {
    print_status "Starting observability stack..."
    cd "$PROJECT_ROOT/infrastructure"
    
    # Start monitoring services
    print_status "Starting Prometheus and Alertmanager..."
    $DOCKER_COMPOSE_CMD up -d prometheus alertmanager
    sleep 10
    
    print_status "Starting Loki and Promtail..."
    $DOCKER_COMPOSE_CMD up -d loki promtail
    sleep 10
    
    print_status "Starting Jaeger and OTEL Collector..."
    $DOCKER_COMPOSE_CMD up -d jaeger otel-collector
    sleep 10
    
    print_status "Starting Grafana..."
    $DOCKER_COMPOSE_CMD up -d grafana
    sleep 10
    
    print_status "Starting Kafka Exporter..."
    $DOCKER_COMPOSE_CMD up -d kafka-exporter
    
    print_success "Observability stack started"
}

# Function to start application services
start_applications() {
    print_status "Starting application services..."
    cd "$PROJECT_ROOT/infrastructure"
    
    print_status "Building and starting Producer Service..."
    $DOCKER_COMPOSE_CMD up -d producer-service
    sleep 15
    
    print_status "Building and starting Consumer Service..."
    $DOCKER_COMPOSE_CMD up -d consumer-service
    sleep 10
    
    print_success "Application services started"
}

# Function to check service health
check_health() {
    print_status "Checking service health..."
    cd "$PROJECT_ROOT/infrastructure"
    
    local services=("prometheus" "grafana" "loki" "jaeger" "producer-service" "consumer-service")
    local failed_services=()
    
    for service in "${services[@]}"; do
        if $DOCKER_COMPOSE_CMD ps "$service" | grep -q "Up"; then
            print_status "✓ $service: Running"
        else
            print_warning "✗ $service: Not running"
            failed_services+=("$service")
        fi
    done
    
    if [[ ${#failed_services[@]} -eq 0 ]]; then
        print_success "All services are healthy"
        return 0
    else
        print_error "Failed services: ${failed_services[*]}"
        return 1
    fi
}

# Function to show access URLs
show_urls() {
    print_success "Observability Stack is ready!"
    echo ""
    echo "🌐 Access URLs:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 Grafana Dashboard:        http://localhost:3000 (admin/admin)"
    echo "🔍 Prometheus:               http://localhost:9090"
    echo "📋 Alertmanager:             http://localhost:9093"
    echo "📝 Loki (API):               http://localhost:3100"
    echo "🔗 Jaeger UI:                http://localhost:16686"
    echo "📡 OTEL Collector Health:    http://localhost:13133"
    echo "🏭 Producer Service:         http://localhost:8081"
    echo "📈 Producer Metrics:         http://localhost:9091/metrics"
    echo "📈 Consumer Metrics:         http://localhost:9094/metrics"
    echo "🎛️  Kafka Exporter:          http://localhost:9308/metrics"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📊 Available Dashboards in Grafana:"
    echo "   • Producer Service Metrics"
    echo "   • Consumer Service Metrics" 
    echo "   • Kafka Infrastructure"
    echo "   • Application Logs (Loki)"
    echo "   • Distributed Tracing (Jaeger)"
    echo ""
    echo "🧪 Test the setup:"
    echo "   cd $PROJECT_ROOT/scripts"
    echo "   ./run_nominal_test.sh      # 300 RPS load test"
    echo "   ./run_extreme_test.sh      # 15K RPS stress test"
}

# Function to show troubleshooting info
show_troubleshooting() {
    echo ""
    echo "🔧 Troubleshooting:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 Check logs:              $DOCKER_COMPOSE_CMD logs [service-name]"
    echo "🔄 Restart service:         $DOCKER_COMPOSE_CMD restart [service-name]" 
    echo "🔍 Check status:            $DOCKER_COMPOSE_CMD ps"
    echo "🛑 Stop all:                $DOCKER_COMPOSE_CMD down"
    echo "🧹 Clean restart:           $DOCKER_COMPOSE_CMD down && $DOCKER_COMPOSE_CMD up -d"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Main execution
main() {
    echo "🚀 Setting up Full Observability Stack"
    echo "========================================"
    
    check_docker_compose
    validate_configs
    stop_services
    
    start_infrastructure
    start_observability
    start_applications
    
    # Wait a bit for services to fully initialize
    print_status "Waiting for services to initialize..."
    sleep 20
    
    if check_health; then
        show_urls
        show_troubleshooting
    else
        print_error "Setup completed with some issues. Check service logs."
        show_troubleshooting
        exit 1
    fi
}

# Handle script termination
cleanup() {
    print_warning "Script interrupted. Services may still be starting..."
    exit 1
}

trap cleanup INT TERM

# Check if script is run with arguments
if [[ $# -gt 0 ]]; then
    case "$1" in
        "stop")
            stop_services
            ;;
        "health")
            check_health
            ;;
        "urls")
            show_urls
            ;;
        *)
            echo "Usage: $0 [stop|health|urls]"
            echo "  stop   - Stop all services"
            echo "  health - Check service health" 
            echo "  urls   - Show access URLs"
            echo "  (no args) - Full setup"
            ;;
    esac
else
    main
fi 