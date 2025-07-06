#!/bin/bash

# Service Manager Script for Activity Log Microservices
# Usage: ./scripts/service-manager.sh [build|start|stop|restart|status] [service]
# Services: http, grpc, consumer, cron, all

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"
LOGS_DIR="$PROJECT_ROOT/logs"
PIDS_DIR="$PROJECT_ROOT/pids"

# Create necessary directories
mkdir -p "$LOGS_DIR" "$PIDS_DIR"

# Service definitions
declare -A SERVICES=(
    ["http"]="http-server"
    ["grpc"]="grpc-server"
    ["consumer"]="consumer"
    ["cron"]="cron-server"
)

declare -A SERVICE_PORTS=(
    ["http"]="8080"
    ["grpc"]="9090"
    ["consumer"]="N/A"
    ["cron"]="N/A"
)

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if service exists
check_service() {
    local service="$1"
    if [[ ! "${SERVICES[$service]}" ]]; then
        log_error "Unknown service: $service"
        log_info "Available services: ${!SERVICES[*]}"
        exit 1
    fi
}

# Build functions
build_service() {
    local service="$1"
    local binary="${SERVICES[$service]}"
    
    log_info "Building $service service..."
    cd "$PROJECT_ROOT"
    
    if go build -o "$BIN_DIR/$binary" "./cmd/$service-server" 2>/dev/null || 
       go build -o "$BIN_DIR/$binary" "./cmd/$service" 2>/dev/null; then
        log_success "$service service built successfully"
    else
        log_error "Failed to build $service service"
        exit 1
    fi
}

build_all() {
    log_info "Building all services..."
    for service in "${!SERVICES[@]}"; do
        build_service "$service"
    done
    log_success "All services built successfully"
}

# Start functions
start_service() {
    local service="$1"
    local binary="${SERVICES[$service]}"
    local pid_file="$PIDS_DIR/$service.pid"
    local log_file="$LOGS_DIR/$service.log"
    
    if [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        log_warning "$service service is already running (PID: $(cat "$pid_file"))"
        return 0
    fi
    
    if [[ ! -f "$BIN_DIR/$binary" ]]; then
        log_warning "$service binary not found, building..."
        build_service "$service"
    fi
    
    log_info "Starting $service service..."
    cd "$PROJECT_ROOT"
    
    # Set environment variables
    export CONFIG_PATH="configs/config.yaml"
    
    # Start the service in background
    nohup "$BIN_DIR/$binary" > "$log_file" 2>&1 &
    local pid=$!
    
    # Save PID
    echo "$pid" > "$pid_file"
    
    # Wait a moment and check if process is still running
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        local port="${SERVICE_PORTS[$service]}"
        if [[ "$port" != "N/A" ]]; then
            log_success "$service service started successfully (PID: $pid, Port: $port)"
        else
            log_success "$service service started successfully (PID: $pid)"
        fi
    else
        log_error "$service service failed to start"
        rm -f "$pid_file"
        exit 1
    fi
}

start_all() {
    log_info "Starting all services..."
    
    # Start in specific order: dependencies first
    start_service "grpc"
    sleep 1
    start_service "http"
    sleep 1
    start_service "consumer"
    sleep 1
    start_service "cron"
    
    log_success "All services started successfully"
}

# Stop functions
stop_service() {
    local service="$1"
    local pid_file="$PIDS_DIR/$service.pid"
    
    if [[ ! -f "$pid_file" ]]; then
        log_warning "$service service is not running"
        return 0
    fi
    
    local pid=$(cat "$pid_file")
    if kill -0 "$pid" 2>/dev/null; then
        log_info "Stopping $service service (PID: $pid)..."
        kill -TERM "$pid"
        
        # Wait for graceful shutdown
        local count=0
        while kill -0 "$pid" 2>/dev/null && [[ $count -lt 10 ]]; do
            sleep 1
            ((count++))
        done
        
        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            log_warning "Force killing $service service..."
            kill -KILL "$pid"
        fi
        
        log_success "$service service stopped successfully"
    else
        log_warning "$service service was not running"
    fi
    
    rm -f "$pid_file"
}

stop_all() {
    log_info "Stopping all services..."
    
    # Stop in reverse order
    stop_service "cron"
    stop_service "consumer"
    stop_service "http"
    stop_service "grpc"
    
    log_success "All services stopped successfully"
}

# Status functions
status_service() {
    local service="$1"
    local pid_file="$PIDS_DIR/$service.pid"
    
    if [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        local pid=$(cat "$pid_file")
        local port="${SERVICE_PORTS[$service]}"
        if [[ "$port" != "N/A" ]]; then
            echo -e "${GREEN}●${NC} $service service is running (PID: $pid, Port: $port)"
        else
            echo -e "${GREEN}●${NC} $service service is running (PID: $pid)"
        fi
    else
        echo -e "${RED}●${NC} $service service is not running"
        [[ -f "$pid_file" ]] && rm -f "$pid_file"
    fi
}

status_all() {
    log_info "Service Status:"
    for service in "${!SERVICES[@]}"; do
        status_service "$service"
    done
}

# Restart function
restart_service() {
    local service="$1"
    log_info "Restarting $service service..."
    stop_service "$service"
    sleep 2
    start_service "$service"
}

restart_all() {
    log_info "Restarting all services..."
    stop_all
    sleep 3
    start_all
}

# Logs function
logs_service() {
    local service="$1"
    local log_file="$LOGS_DIR/$service.log"
    
    if [[ -f "$log_file" ]]; then
        tail -f "$log_file"
    else
        log_error "Log file not found: $log_file"
        exit 1
    fi
}

# Help function
show_help() {
    echo "Activity Log Service Manager"
    echo ""
    echo "Usage: $0 [COMMAND] [SERVICE]"
    echo ""
    echo "Commands:"
    echo "  build      Build service(s)"
    echo "  start      Start service(s)"
    echo "  stop       Stop service(s)"
    echo "  restart    Restart service(s)"
    echo "  status     Show service status"
    echo "  logs       Show service logs (tail -f)"
    echo ""
    echo "Services:"
    echo "  http       HTTP/REST API server (Port: 8080)"
    echo "  grpc       gRPC API server (Port: 9090)"
    echo "  consumer   NATS message consumer"
    echo "  cron       Scheduled jobs server"
    echo "  all        All services"
    echo ""
    echo "Examples:"
    echo "  $0 build all"
    echo "  $0 start http"
    echo "  $0 status all"
    echo "  $0 logs consumer"
    echo "  $0 restart grpc"
}

# Main script logic
main() {
    local command="$1"
    local service="$2"
    
    case "$command" in
        build)
            if [[ "$service" == "all" || -z "$service" ]]; then
                build_all
            else
                check_service "$service"
                build_service "$service"
            fi
            ;;
        start)
            if [[ "$service" == "all" || -z "$service" ]]; then
                start_all
            else
                check_service "$service"
                start_service "$service"
            fi
            ;;
        stop)
            if [[ "$service" == "all" || -z "$service" ]]; then
                stop_all
            else
                check_service "$service"
                stop_service "$service"
            fi
            ;;
        restart)
            if [[ "$service" == "all" || -z "$service" ]]; then
                restart_all
            else
                check_service "$service"
                restart_service "$service"
            fi
            ;;
        status)
            if [[ "$service" == "all" || -z "$service" ]]; then
                status_all
            else
                check_service "$service"
                status_service "$service"
            fi
            ;;
        logs)
            if [[ -z "$service" ]]; then
                log_error "Service name required for logs command"
                exit 1
            fi
            check_service "$service"
            logs_service "$service"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"