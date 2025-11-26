#!/bin/bash

# ============================================================================
# Flin Complete Test Suite - Starts Server + Runs All Tests
# ============================================================================
# This script:
#   1. Builds the Flin server
#   2. Starts the Flin server with HTTP API
#   3. Waits for the server to be ready
#   4. Runs comprehensive API tests
#   5. Cleans up (stops the server)
#
# Usage:
#   chmod +x test-with-server.sh
#   ./test-with-server.sh
# ============================================================================

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
FLIN_DIR="/home/shohag/Personal/flin"
STUDIO_DIR="/home/shohag/Personal/flin/studio"
API_URL="http://localhost:8888"
BINARY_PORT=":6380"
HTTP_PORT="127.0.0.1:8080"
RAFT_PORT="127.0.0.1:9080"
API_PORT=":8888"
WAIT_TIMEOUT=45
FAILED_TESTS=0
PASSED_TESTS=0

# ============================================================================
# Helper Functions
# ============================================================================

print_banner() {
    echo ""
    echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║                    FLIN COMPLETE TEST SUITE                                   ║${NC}"
    echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_header() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║ $1${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_step() {
    echo -e "${CYAN}» $1${NC}"
}

print_test() {
    echo -e "${YELLOW}  → $1${NC}"
}

print_success() {
    echo -e "${GREEN}  ✓ $1${NC}"
    ((PASSED_TESTS++))
}

print_error() {
    echo -e "${RED}  ✗ $1${NC}"
    ((FAILED_TESTS++))
}

print_info() {
    echo -e "${BLUE}  ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}  ⚠ $1${NC}"
}

cleanup() {
    print_header "Cleanup"
    print_step "Stopping Flin server..."
    
    if pgrep -f "flin-server" > /dev/null; then
        pkill -f "flin-server" || true
        sleep 1
        print_success "Server stopped"
    else
        print_info "Server was already stopped"
    fi
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# ============================================================================
# Build & Start Server
# ============================================================================

start_server() {
    print_header "Building Flin Server"
    
    cd "$FLIN_DIR"
    
    if [ ! -f "bin/flin-server" ]; then
        print_step "Building binary..."
        if go build -o bin/flin-server ./cmd/server; then
            print_success "Build successful"
        else
            print_error "Build failed"
            return 1
        fi
    else
        print_info "Binary already exists"
    fi
    
    print_header "Starting Flin Server"
    print_step "Starting server with HTTP API on $API_PORT..."
    
    # Start server in background
    ./bin/flin-server \
        -node-id=node1 \
        -port=$BINARY_PORT \
        -http=$HTTP_PORT \
        -raft=$RAFT_PORT \
        > /tmp/flin-server.log 2>&1 &
    
    SERVER_PID=$!
    print_info "Server PID: $SERVER_PID"
    
    # Wait for API to be ready
    print_step "Waiting for API to be ready (max $WAIT_TIMEOUT seconds)..."
    
    counter=0
    while [ $counter -lt $WAIT_TIMEOUT ]; do
        if curl -s "$API_URL/health" > /dev/null 2>&1; then
            print_success "API is ready!"
            sleep 1
            return 0
        fi
        
        echo -ne "\r  [$(($counter+1))/$WAIT_TIMEOUT] Still waiting..."
        counter=$((counter+1))
        sleep 1
    done
    
    print_error "API failed to start within $WAIT_TIMEOUT seconds"
    print_info "Server logs:"
    tail -20 /tmp/flin-server.log
    return 1
}

# ============================================================================
# Test Functions
# ============================================================================

test_health() {
    print_test "GET /health"
    
    response=$(curl -s -w "\n%{http_code}" "$API_URL/health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Health check passed"
        print_info "Response: $body"
    else
        print_error "Health check failed (HTTP $http_code)"
    fi
}

test_status() {
    print_test "GET /status"
    
    response=$(curl -s -w "\n%{http_code}" "$API_URL/status")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Status endpoint works"
        print_info "Response: $body"
    else
        print_error "Status endpoint failed (HTTP $http_code)"
    fi
}

test_kv_set() {
    print_test "POST /kv/set - Set key 'test:key1' = 'value1'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/kv/set" \
        -H "Content-Type: application/json" \
        -d '{"key":"test:key1","value":"value1"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        print_success "Set key successful"
    else
        print_error "Set key failed (HTTP $http_code) - $body"
    fi
}

test_kv_get() {
    print_test "GET /kv/get?key=test:key1"
    
    response=$(curl -s -w "\n%{http_code}" "$API_URL/kv/get?key=test:key1")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Get key successful"
        print_info "Response: $body"
    elif [ "$http_code" = "404" ]; then
        print_warning "Key not found (expected if set failed)"
    else
        print_error "Get key failed (HTTP $http_code)"
    fi
}

test_kv_update() {
    print_test "POST /kv/update - Update key 'test:key1' = 'updated_value'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/kv/update" \
        -H "Content-Type: application/json" \
        -d '{"key":"test:key1","value":"updated_value"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Update key successful"
    elif [ "$http_code" = "404" ]; then
        print_warning "Key not found (expected if set failed)"
    else
        print_error "Update key failed (HTTP $http_code) - $body"
    fi
}

test_kv_delete() {
    print_test "POST /kv/delete - Delete key 'test:key1'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/kv/delete" \
        -H "Content-Type: application/json" \
        -d '{"key":"test:key1"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Delete key successful"
    else
        print_error "Delete key failed (HTTP $http_code) - $body"
    fi
}

test_kv_keys() {
    print_test "GET /kv/keys - List all keys"
    
    response=$(curl -s -w "\n%{http_code}" "$API_URL/kv/keys")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "List keys successful"
        print_info "Response: $body"
    else
        print_error "List keys failed (HTTP $http_code)"
    fi
}

test_queue_push() {
    print_test "POST /queues/push - Push message to 'test:queue1'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/queues/push" \
        -H "Content-Type: application/json" \
        -d '{"queue":"test:queue1","message":"test message 1"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        print_success "Push message successful"
    else
        print_error "Push message failed (HTTP $http_code) - $body"
    fi
}

test_queue_pop() {
    print_test "POST /queues/pop - Pop message from 'test:queue1'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/queues/pop" \
        -H "Content-Type: application/json" \
        -d '{"queue":"test:queue1"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Pop message successful"
        print_info "Response: $body"
    elif [ "$http_code" = "404" ]; then
        print_warning "Queue empty (expected if push failed)"
    else
        print_error "Pop message failed (HTTP $http_code)"
    fi
}

test_queue_create() {
    print_test "POST /queues/create - Create new queue 'test:new-queue'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/queues/create" \
        -H "Content-Type: application/json" \
        -d '{"name":"test:new-queue"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Create queue successful"
    else
        print_error "Create queue failed (HTTP $http_code) - $body"
    fi
}

test_queue_list() {
    print_test "GET /queues - List all queues"
    
    response=$(curl -s -w "\n%{http_code}" "$API_URL/queues")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "List queues successful"
        print_info "Response: $body"
    else
        print_error "List queues failed (HTTP $http_code)"
    fi
}

test_queue_delete() {
    print_test "POST /queues/delete - Delete queue 'test:new-queue'"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/queues/delete" \
        -H "Content-Type: application/json" \
        -d '{"name":"test:new-queue"}')
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        print_success "Delete queue successful"
    else
        print_error "Delete queue failed (HTTP $http_code) - $body"
    fi
}

# ============================================================================
# Main Test Suite
# ============================================================================

run_tests() {
    print_header "Running API Tests"
    
    print_header "Health & Status Tests"
    test_health
    test_status
    
    print_header "KV Store Tests"
    test_kv_set
    test_kv_get
    test_kv_update
    test_kv_delete
    test_kv_keys
    
    print_header "Queue Tests"
    test_queue_push
    test_queue_pop
    test_queue_create
    test_queue_list
    test_queue_delete
}

print_summary() {
    print_header "Test Summary"
    
    TOTAL_TESTS=$((PASSED_TESTS + FAILED_TESTS))
    PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo -e "${BLUE}Total:  $TOTAL_TESTS${NC}"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed! (${PASS_RATE}%)${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ Some tests failed (${PASS_RATE}% pass rate)${NC}"
        return 1
    fi
}

# ============================================================================
# Data Seeding
# ============================================================================

seed_data() {
    print_header "Seeding Demo Data"
    
    # Create queues
    print_step "Creating demo queues..."
    curl -s -X POST "$API_URL/queues/create" -H "Content-Type: application/json" -d '{"name":"orders:processed"}' > /dev/null
    curl -s -X POST "$API_URL/queues/create" -H "Content-Type: application/json" -d '{"name":"notifications:email"}' > /dev/null
    print_success "Created 'orders:processed' and 'notifications:email'"

    # Push messages
    print_step "Pushing messages..."
    for i in {1..10}; do
        curl -s -X POST "$API_URL/queues/push" -H "Content-Type: application/json" -d "{\"queue\":\"orders:processed\",\"message\":\"Order #$i processed at $(date)\"}" > /dev/null
    done
    print_success "Pushed 10 messages to 'orders:processed'"

    # Set KV data
    print_step "Setting KV pairs..."
    curl -s -X POST "$API_URL/kv/set" -H "Content-Type: application/json" -d '{"key":"config:app_name","value":"Flin Studio Demo"}' > /dev/null
    curl -s -X POST "$API_URL/kv/set" -H "Content-Type: application/json" -d '{"key":"user:admin:status","value":"active"}' > /dev/null
    curl -s -X POST "$API_URL/kv/set" -H "Content-Type: application/json" -d '{"key":"stats:visits","value":"1024"}' > /dev/null
    print_success "Set demo KV pairs"
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    print_banner
    
    print_header "Starting Test Suite"
    print_info "Flin Directory: $FLIN_DIR"
    print_info "API URL: $API_URL"
    
    # Start server
    if ! start_server; then
        print_error "Failed to start server"
        exit 1
    fi
    
    # Run tests
    run_tests
    
    # Seed data
    seed_data
    
    # Print summary
    print_summary
    RESULT=$?
    
    print_header "Server Running"
    print_info "Server is still running at $API_URL"
    print_info "You can now use Flin Studio at http://localhost:3000"
    print_info "Press [ENTER] to stop the server and exit..."
    read -r
    
    exit $RESULT
}

# Run main
main
