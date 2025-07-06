#!/bin/bash

# Test script for Activity Log Service
# Usage: ./test_requests.sh [create|get|list|all]

set -e

GRPC_ADDR="localhost:9000"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_service() {
    log_info "Checking if gRPC service is running..."
    if ! nc -z localhost 9000 2>/dev/null; then
        log_error "gRPC service is not running on port 9000"
        log_info "Please start the service with: make docker-run"
        exit 1
    fi
    log_info "gRPC service is running ✓"
}

test_create() {
    log_info "Testing CreateActivityLog..."
    cd "$PROJECT_DIR"
    
    go run requests/scripts/grpc_client.go \
        -addr="$GRPC_ADDR" \
        -method=create \
        -file=requests/grpc/create_activity_log.json
    
    log_info "CreateActivityLog test completed ✓"
}

test_get() {
    log_warn "Testing GetActivityLog requires a valid activity log ID"
    log_info "Please update requests/grpc/get_activity_log.json with a valid ID first"
    
    cd "$PROJECT_DIR"
    
    go run requests/scripts/grpc_client.go \
        -addr="$GRPC_ADDR" \
        -method=get \
        -file=requests/grpc/get_activity_log.json
    
    log_info "GetActivityLog test completed ✓"
}

test_list() {
    log_info "Testing ListActivityLogs..."
    cd "$PROJECT_DIR"
    
    go run requests/scripts/grpc_client.go \
        -addr="$GRPC_ADDR" \
        -method=list \
        -file=requests/grpc/list_activity_logs.json
    
    log_info "ListActivityLogs test completed ✓"
}

test_all() {
    log_info "Running all tests..."
    test_create
    echo ""
    test_list
    echo ""
    log_info "All tests completed! ✓"
    log_warn "Note: GetActivityLog test skipped (requires valid ID)"
}

# Main execution
case "${1:-all}" in
    create)
        check_service
        test_create
        ;;
    get)
        check_service
        test_get
        ;;
    list)
        check_service
        test_list
        ;;
    all)
        check_service
        test_all
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Usage: $0 [create|get|list|all]"
        exit 1
        ;;
esac