#!/bin/bash
set -e

# Integration test runner for Tilt development environment
# Runs comprehensive tests against live services

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
AUDIO_ROUTER_URL="${AUDIO_ROUTER_URL:-http://localhost:9090}"
TEST_DURATION="${TEST_DURATION:-60}"
VERBOSE="${VERBOSE:-false}"

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] INFO:${NC} $1"
}

success() {
    echo -e "${GREEN}[$(date +'%H:%M:%S')] SUCCESS:${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%H:%M:%S')] ERROR:${NC} $1"
}

warning() {
    echo -e "${YELLOW}[$(date +'%H:%M:%S')] WARNING:${NC} $1"
}

debug() {
    if [ "$VERBOSE" = "true" ]; then
        echo -e "${PURPLE}[$(date +'%H:%M:%S')] DEBUG:${NC} $1"
    fi
}

# Test utilities
wait_for_service() {
    local service_name=$1
    local url=$2
    local timeout=${3:-30}
    
    log "Waiting for $service_name to be ready..."
    
    for i in $(seq 1 $timeout); do
        if curl -s -f "$url" >/dev/null 2>&1; then
            success "$service_name is ready"
            return 0
        fi
        debug "Attempt $i/$timeout: $service_name not ready yet..."
        sleep 1
    done
    
    error "$service_name failed to become ready within ${timeout}s"
    return 1
}

wait_for_udp_service() {
    local service_name=$1
    local host=$2
    local port=$3
    local timeout=${4:-30}
    
    log "Waiting for $service_name UDP service on $host:$port..."
    
    for i in $(seq 1 $timeout); do
        if nc -u -z "$host" "$port" 2>/dev/null; then
            success "$service_name UDP service is ready"
            return 0
        fi
        debug "Attempt $i/$timeout: $service_name UDP not ready yet..."
        sleep 1
    done
    
    error "$service_name UDP service failed to become ready within ${timeout}s"
    return 1
}

# Test functions
test_service_health() {
    log "Testing service health checks..."
    
    # Audio Router Hub health
    if ! wait_for_service "Audio Router Hub" "$AUDIO_ROUTER_URL/status" 30; then
        return 1
    fi
    
    # Check AllStar mock services
    if ! wait_for_udp_service "AllStar Mock 1" "localhost" "34001" 20; then
        warning "AllStar Mock 1 not available"
    fi
    
    if ! wait_for_udp_service "AllStar Mock 2" "localhost" "34002" 20; then
        warning "AllStar Mock 2 not available"  
    fi
    
    # Check WhoTalkie mock services
    if ! wait_for_udp_service "WhoTalkie Mock 1" "localhost" "8080" 20; then
        warning "WhoTalkie Mock 1 not available"
    fi
    
    success "Service health check completed"
    return 0
}

test_audio_router_status() {
    log "Testing Audio Router Hub API..."
    
    local status_response
    status_response=$(curl -s "$AUDIO_ROUTER_URL/status" 2>/dev/null)
    
    if [ $? -ne 0 ] || [ -z "$status_response" ]; then
        error "Failed to get status from Audio Router Hub"
        return 1
    fi
    
    debug "Status response: $status_response"
    success "Audio Router Hub API is responding"
    return 0
}

test_packet_flow() {
    log "Testing packet flow between services..."
    
    local test_duration=${1:-30}
    local capture_file="/tmp/tilt_packet_capture_$$.pcap"
    
    # Start packet capture in background
    debug "Starting packet capture for ${test_duration}s..."
    timeout $test_duration tcpdump -i any -w "$capture_file" \
        "udp and (port 34001 or port 34002 or port 32001 or port 32002 or port 8080)" \
        >/dev/null 2>&1 &
    local tcpdump_pid=$!
    
    # Let test run
    sleep $test_duration
    
    # Stop capture
    kill $tcpdump_pid 2>/dev/null || true
    wait $tcpdump_pid 2>/dev/null || true
    
    # Analyze captured packets
    if [ -f "$capture_file" ]; then
        local packet_count
        packet_count=$(tcpdump -r "$capture_file" 2>/dev/null | wc -l)
        
        if [ "$packet_count" -gt 0 ]; then
            success "Captured $packet_count packets during test"
            debug "Packet flow analysis:"
            
            # Analyze traffic patterns
            local allstar_packets
            allstar_packets=$(tcpdump -r "$capture_file" "src port 34001 or src port 34002" 2>/dev/null | wc -l)
            debug "  AllStar mock packets: $allstar_packets"
            
            local router_packets  
            router_packets=$(tcpdump -r "$capture_file" "dst port 32001 or dst port 32002" 2>/dev/null | wc -l)
            debug "  Router received packets: $router_packets"
            
            rm -f "$capture_file"
            return 0
        else
            warning "No packets captured during test"
            rm -f "$capture_file"
            return 1
        fi
    else
        error "Packet capture file not found"
        return 1
    fi
}

test_service_discovery() {
    log "Testing service discovery and configuration..."
    
    # Check if services are properly configured
    local status_response
    status_response=$(curl -s "$AUDIO_ROUTER_URL/status" 2>/dev/null)
    
    if echo "$status_response" | grep -q "services"; then
        success "Service discovery working"
        
        # Extract service count if possible (would need jq in real implementation)
        debug "Service configuration loaded successfully"
        return 0
    else
        error "Service discovery not working properly"
        return 1
    fi
}

test_performance_basic() {
    log "Testing basic performance metrics..."
    
    local start_time=$(date +%s)
    local request_count=50
    local successful_requests=0
    
    for i in $(seq 1 $request_count); do
        if curl -s -f "$AUDIO_ROUTER_URL/status" >/dev/null 2>&1; then
            ((successful_requests++))
        fi
        
        if [ $((i % 10)) -eq 0 ]; then
            debug "Performance test: $i/$request_count requests"
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local success_rate=$((successful_requests * 100 / request_count))
    
    if [ $duration -gt 0 ]; then
        local rps=$((request_count / duration))
        log "Performance: $successful_requests/$request_count requests successful (${success_rate}%)"
        log "Average: $rps requests/second"
    fi
    
    if [ $success_rate -ge 95 ]; then
        success "Performance test passed (${success_rate}% success rate)"
        return 0
    else
        error "Performance test failed (${success_rate}% success rate)"
        return 1
    fi
}

test_configuration_validation() {
    log "Testing configuration validation..."
    
    # Test that the Audio Router is using the correct configuration
    local config_test_result
    config_test_result=$(kubectl exec deployment/audio-router -- \
        ls -la /app/config/audio-router-dev.json 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        success "Configuration file is accessible"
        debug "Config file details: $config_test_result"
        return 0
    else
        error "Configuration validation failed"
        return 1
    fi
}

# Test reporting
generate_test_report() {
    local total_tests=$1
    local passed_tests=$2
    local test_results=("${@:3}")
    
    local failed_tests=$((total_tests - passed_tests))
    local success_rate=$((passed_tests * 100 / total_tests))
    
    echo ""
    echo "======================================================"
    echo "🎵 USRP Audio Router Hub - Tilt Integration Test Report"
    echo "======================================================"
    echo "Date: $(date)"
    echo "Environment: Tilt Development"
    echo "Test Duration: ${TEST_DURATION}s"
    echo ""
    echo "Results Summary:"
    echo "  Total Tests: $total_tests"
    echo "  Passed: $passed_tests"
    echo "  Failed: $failed_tests" 
    echo "  Success Rate: ${success_rate}%"
    echo ""
    
    # Individual test results
    echo "Individual Test Results:"
    for i in "${!test_results[@]}"; do
        echo "  $((i + 1)). ${test_results[i]}"
    done
    
    echo ""
    if [ $failed_tests -eq 0 ]; then
        echo -e "${GREEN}🎉 ALL TESTS PASSED! 🎉${NC}"
        echo "Your USRP Audio Router Hub is working correctly!"
    else
        echo -e "${YELLOW}⚠️  Some tests failed. Check Tilt logs for details.${NC}"
        echo "Tilt Dashboard: http://localhost:10350"
        echo "Audio Router Status: $AUDIO_ROUTER_URL/status"
    fi
    
    echo ""
    echo "Quick Debug Commands:"
    echo "  tilt logs audio-router        # View Audio Router logs"
    echo "  tilt logs allstar-mock-1      # View AllStar Mock 1 logs" 
    echo "  kubectl get pods              # Check pod status"
    echo "  curl $AUDIO_ROUTER_URL/status # Check Audio Router status"
    echo ""
    echo "======================================================"
}

# Main test execution
main() {
    echo ""
    echo -e "${CYAN}🧪 Starting Tilt Integration Tests for USRP Audio Router Hub${NC}"
    echo -e "${CYAN}================================================================${NC}"
    
    log "Test configuration:"
    log "  Audio Router URL: $AUDIO_ROUTER_URL"
    log "  Test Duration: ${TEST_DURATION}s"
    log "  Verbose Mode: $VERBOSE"
    echo ""
    
    local tests=(
        "test_service_health"
        "test_audio_router_status"
        "test_service_discovery"
        "test_configuration_validation"
        "test_packet_flow"
        "test_performance_basic"
    )
    
    local test_names=(
        "Service Health Checks"
        "Audio Router API Status"
        "Service Discovery"
        "Configuration Validation"
        "Packet Flow Analysis"
        "Basic Performance Test"
    )
    
    local total_tests=${#tests[@]}
    local passed_tests=0
    local test_results=()
    
    for i in "${!tests[@]}"; do
        local test_name="${test_names[i]}"
        local test_function="${tests[i]}"
        
        log "Running: $test_name..."
        echo "---"
        
        if $test_function; then
            success "$test_name PASSED"
            test_results+=("✅ $test_name - PASSED")
            ((passed_tests++))
        else
            error "$test_name FAILED"
            test_results+=("❌ $test_name - FAILED")
        fi
        
        echo ""
    done
    
    generate_test_report $total_tests $passed_tests "${test_results[@]}"
    
    # Exit with appropriate code
    if [ $passed_tests -eq $total_tests ]; then
        exit 0
    else
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        echo "Tilt Integration Test Runner for USRP Audio Router Hub"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  help                Show this help message"
        echo "  packet-flow         Run packet flow test only"
        echo "  performance         Run performance test only"
        echo "  health              Run health checks only"
        echo ""
        echo "Environment Variables:"
        echo "  AUDIO_ROUTER_URL    Audio Router URL (default: http://localhost:9090)"
        echo "  TEST_DURATION       Test duration in seconds (default: 60)"
        echo "  VERBOSE             Enable verbose logging (default: false)"
        echo ""
        exit 0
        ;;
    "packet-flow")
        test_packet_flow $TEST_DURATION
        exit $?
        ;;
    "performance")
        test_performance_basic
        exit $?
        ;;
    "health")
        test_service_health
        exit $?
        ;;
    *)
        main "$@"
        ;;
esac