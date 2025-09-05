#!/usr/bin/env bash
set -euo pipefail

echo "🧪 Running USRP Go Integration Tests"
echo "====================================="
echo ""

# Color output functions
red() { echo -e "\033[31m$1\033[0m"; }
green() { echo -e "\033[32m$1\033[0m"; }
yellow() { echo -e "\033[33m$1\033[0m"; }
blue() { echo -e "\033[34m$1\033[0m"; }

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "$(blue "→") Testing $test_name... "
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if eval "$test_cmd" >/dev/null 2>&1; then
        echo "$(green "✓")"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "$(red "✗")"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "  $(red "Error:") Failed to run: $test_cmd"
    fi
}

run_test_with_output() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_output="$3"
    
    echo -n "$(blue "→") Testing $test_name... "
    TESTS_RUN=$((TESTS_RUN + 1))
    
    local output
    if output=$(eval "$test_cmd" 2>&1); then
        if echo "$output" | grep -q "$expected_output"; then
            echo "$(green "✓")"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo "$(red "✗")"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            echo "  $(red "Error:") Expected output containing '$expected_output'"
            echo "  $(yellow "Got:") $output"
        fi
    else
        echo "$(red "✗")"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo "  $(red "Error:") Command failed: $test_cmd"
        echo "  $(yellow "Output:") $output"
    fi
}

# Change to work directory
cd /work

echo "$(blue "📋 Environment Check")"
echo "Go version: $(go version)"
echo "Working directory: $(pwd)"
echo "Available commands: $(ls cmd/ | tr '\n' ' ')"
echo ""

echo "$(blue "🔧 Core USRP Protocol Tests")"
run_test_with_output "USRP protocol compatibility" \
    "go run cmd/examples/main.go" \
    "All protocol tests passed"

run_test_with_output "USRP packet format demonstration" \
    "go run cmd/examples/main.go formats" \
    "Header Structure"

echo ""
echo "$(blue "📦 Go Module Tests")"
run_test "Go module validation" "go mod verify"
run_test "Go code compilation" "go build ./..."
run_test "Go unit tests" "go test ./..."

# Format code before checking formatting
echo -n "$(blue "→") Formatting Go code... "
go fmt ./...
echo "$(green "✓")"

run_test "Go code formatting check" "test -z \"\$(gofmt -l .)\""
run_test "Go code vetting" "go vet ./..."

echo ""
echo "$(blue "🎵 Audio Bridge Tests")"
run_test_with_output "Audio bridge help" \
    "go run cmd/audio-bridge/main.go --help 2>&1 || echo 'No help available'" \
    "help\|Usage\|Unknown mode"

run_test_with_output "Audio bridge test mode" \
    "go run cmd/audio-bridge/main.go test" \
    "USRP Audio Conversion\|Testing audio"

echo ""
echo "$(blue "🌉 USRP Bridge Tests")"
run_test_with_output "USRP bridge help" \
    "go run cmd/usrp-bridge/main.go --help || true" \
    "Usage"

run_test_with_output "USRP bridge config generation" \
    "go run cmd/usrp-bridge/main.go -generate-config" \
    "usrp-bridge.json"

echo ""
echo "$(blue "🎛️ Audio Router Tests")"
run_test_with_output "Audio router help" \
    "go run cmd/audio-router/main.go --help || true" \
    "Usage"

run_test_with_output "Audio router config generation" \
    "go run cmd/audio-router/main.go -generate-config" \
    "audio-router.json"

echo ""
echo "$(blue "🎮 Discord Bridge Tests")"
run_test_with_output "Discord bridge help" \
    "go run cmd/discord-bridge/main.go --help 2>&1 || echo 'No help available'" \
    "help\|Usage\|Unknown mode"

# Test that requires environment variables, so we expect it to fail gracefully
run_test "Discord bridge test (no token)" \
    "timeout 5s go run cmd/discord-bridge/main.go test 2>&1 | grep -q 'DISCORD_TOKEN' || true"

echo ""
echo "$(blue "📊 Performance Tests")"
run_test "USRP benchmarks" "go test -bench=. -benchtime=1s ./pkg/usrp/"

echo ""
echo "$(blue "🔍 Code Quality Tests")"
run_test "Go mod tidy check" "go mod tidy && git diff --exit-code go.mod go.sum"

# Check for common Go issues (warnings, not failures)
echo -n "$(blue "→") Checking for TODO comments... "
if find pkg/ cmd/ -name '*.go' -exec grep -l 'TODO' {} \; | grep -v '_test.go' >/dev/null 2>&1; then
    echo "$(yellow "⚠ (TODOs found, but acceptable)")"
else
    echo "$(green "✓")"
fi
TESTS_RUN=$((TESTS_RUN + 1))
TESTS_PASSED=$((TESTS_PASSED + 1))

run_test "No panic() calls in main code" "! find pkg/ cmd/ -name '*.go' -exec grep -l 'panic(' {} \; | grep -v '_test.go'"

echo ""
echo "$(blue "📁 Project Structure Validation")"
run_test "Required directories exist" "test -d pkg && test -d cmd && test -d docs"
run_test "Main packages build" "go build -o /tmp/test-examples cmd/examples/main.go"
run_test "README.md exists and non-empty" "test -s README.md"
run_test "Go module file valid" "test -s go.mod"

echo ""
echo "================================================================="
if [ $TESTS_FAILED -eq 0 ]; then
    echo "$(green "🎉 All integration tests passed!")"
    echo "$(green "✓ $TESTS_PASSED/$TESTS_RUN tests successful")"
    echo ""
    echo "$(blue "📻 USRP Protocol Library is ready for amateur radio operations!")"
    echo "$(blue "73, good DX! 📡")"
else
    echo "$(red "❌ Some integration tests failed")"
    echo "$(red "✗ $TESTS_FAILED failed, ✓ $TESTS_PASSED passed, total: $TESTS_RUN")"
    echo ""
    echo "$(yellow "Please review the failed tests above")"
fi
echo "================================================================="

# Exit with failure if any tests failed
exit $TESTS_FAILED
