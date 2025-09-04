# USRP Audio Router Hub - Just Command Runner
# Modern replacement for Makefile with better syntax and developer experience

# Default recipe - show help
default:
    @just --list

# Show all available commands with descriptions
help:
    @echo "🎵 USRP Audio Router Hub - Available Commands"
    @echo "============================================="
    @echo ""
    @echo "📋 Basic Development:"
    @just --list --unsorted | grep -E "(build|test|fmt|vet|lint|clean)" | head -10
    @echo ""
    @echo "🎛️ Audio Router Hub:"  
    @just --list --unsorted | grep -E "router" | head -10
    @echo ""
    @echo "🧪 Integration Testing:"
    @just --list --unsorted | grep -E "(integration|tilt)" | head -15
    @echo ""
    @echo "For detailed help: just help-detailed"

# Show detailed help with all commands organized
help-detailed:
    @echo "🎵 USRP Audio Router Hub - Complete Command Reference"
    @echo "===================================================="
    @echo ""
    @echo "📋 Basic Development:"
    @echo "  just build                  - Build all Go packages"
    @echo "  just test                   - Run all tests"  
    @echo "  just test-coverage          - Run tests with coverage"
    @echo "  just bench                  - Run benchmarks"
    @echo "  just fmt                    - Format Go code"
    @echo "  just vet                    - Vet Go code"
    @echo "  just lint                   - Lint code (requires golangci-lint)"
    @echo "  just clean                  - Clean build artifacts"
    @echo "  just deps                   - Install/update dependencies"
    @echo ""
    @echo "🎯 Examples and Demos:"
    @echo "  just run-example            - Run protocol examples"
    @echo "  just run-demo               - Run full UDP demo"
    @echo ""
    @echo "🔄 Audio Conversion:"
    @echo "  just audio-test             - Test audio conversion"
    @echo "  just audio-server           - Run audio bridge server"
    @echo "  just audio-client           - Run audio bridge client"
    @echo ""
    @echo "🌉 USRP Bridge Utility:"
    @echo "  just usrp-bridge            - Run USRP bridge utility"
    @echo "  just usrp-bridge-config     - Generate USRP bridge configuration"
    @echo "  just build-usrp-bridge      - Build USRP bridge binary"
    @echo ""
    @echo "🎮 Discord Integration:"
    @echo "  just discord-test           - Test Discord bot connection"
    @echo "  just discord-bridge         - Run Discord-USRP bridge"
    @echo "  just discord-server         - Run USRP test server for Discord"
    @echo ""
    @echo "🎛️ Audio Router Hub:"
    @echo "  just router                 - Run Audio Router Hub with default config"
    @echo "  just router-config          - Generate sample Audio Router configuration"
    @echo "  just router-with-config     - Run Audio Router Hub with configuration"
    @echo "  just build-router           - Build Audio Router Hub binary"
    @echo ""
    @echo "🧪 Integration Testing:"
    @echo "  just test-integration       - Run complete Docker-based integration tests"
    @echo "  just integration-build      - Build integration test containers"
    @echo "  just integration-up         - Start integration test environment"
    @echo "  just integration-down       - Stop integration test environment"
    @echo "  just integration-logs       - Show integration test logs"
    @echo "  just integration-clean      - Clean up integration test environment"
    @echo ""
    @echo "🚀 Tilt Development Environment:"
    @echo "  just dev                    - Start Tilt development environment (alias for tilt-up)"
    @echo "  just tilt-up                - Start Tilt development environment"
    @echo "  just tilt-down              - Stop Tilt development environment"
    @echo "  just tilt-logs              - Show Tilt service logs"
    @echo "  just tilt-test              - Run integration tests in Tilt"
    @echo "  just tilt-status            - Check Audio Router status"
    @echo "  just tilt-dashboard         - Open Tilt dashboard"
    @echo ""
    @echo "🔧 Build and Release:"
    @echo "  just release                - Create release binaries for all platforms"
    @echo ""
    @echo "💡 Quick Start for New Developers:"
    @echo "  just setup                  - One-time development setup"
    @echo "  just dev                    - Start development environment"
    @echo "  just test                   - Run tests to verify everything works"

# =============================================================================
# Basic Development Commands
# =============================================================================

# Build all Go packages
build:
    @echo "🔨 Building all Go packages..."
    go build ./...

# Run all tests
test:
    @echo "🧪 Running all tests..."
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "📊 Running tests with coverage..."
    go test -v -cover ./...

# Run benchmarks
bench:
    @echo "⚡ Running benchmarks..."
    go test -bench=. ./pkg/usrp/

# Format Go code
fmt:
    @echo "🎨 Formatting Go code..."
    go fmt ./...

# Vet Go code
vet:
    @echo "🔍 Vetting Go code..."
    go vet ./...

# Lint code (requires golangci-lint)
lint:
    @echo "🧹 Linting code..."
    golangci-lint run || echo "golangci-lint not installed, skipping..."

# Clean build artifacts
clean:
    @echo "🗑️ Cleaning build artifacts..."
    go clean ./...
    rm -rf bin/ dist/ *.log *.pcap

# Install and update dependencies
deps:
    @echo "📦 Installing/updating dependencies..."
    go mod download
    go mod tidy

# =============================================================================
# Examples and Demos  
# =============================================================================

# Run protocol examples (basic packet testing)
run-example:
    @echo "🎯 Running protocol examples..."
    go run cmd/examples/main.go

# Run full UDP demo with network testing
run-demo:
    @echo "🌐 Running full UDP demo..."
    go run cmd/examples/main.go demo

# =============================================================================
# Audio Conversion Commands
# =============================================================================

# Test audio format conversion
audio-test:
    @echo "🔄 Testing audio conversion..."
    go run cmd/audio-bridge/main.go test

# Run audio bridge server
audio-server:
    @echo "🖥️ Running audio bridge server..."
    go run cmd/audio-bridge/main.go server

# Run audio bridge client
audio-client:
    @echo "💻 Running audio bridge client..."
    go run cmd/audio-bridge/main.go client

# =============================================================================
# USRP Bridge Utility Commands
# =============================================================================

# Run USRP bridge utility
usrp-bridge:
    @echo "🌉 Running USRP bridge utility..."
    go run cmd/usrp-bridge/main.go

# Generate USRP bridge configuration
usrp-bridge-config:
    @echo "⚙️ Generating USRP bridge configuration..."
    go run cmd/usrp-bridge/main.go -generate-config

# Build USRP bridge binary
build-usrp-bridge:
    @echo "🔨 Building USRP bridge binary..."
    mkdir -p bin
    go build -o bin/usrp-bridge cmd/usrp-bridge/main.go

# =============================================================================
# Discord Integration Commands  
# =============================================================================

# Test Discord bot connection
discord-test:
    @echo "🎮 Testing Discord bot connection..."
    go run cmd/discord-bridge/main.go test

# Run Discord-USRP bridge
discord-bridge:
    @echo "🔗 Running Discord-USRP bridge..."
    go run cmd/discord-bridge/main.go bridge

# Run USRP test server for Discord bridge
discord-server:
    @echo "📡 Running USRP test server for Discord bridge..."
    go run cmd/discord-bridge/main.go server

# =============================================================================
# Audio Router Hub Commands
# =============================================================================

# Run Audio Router Hub with default configuration
router:
    @echo "🎛️ Running Audio Router Hub with default configuration..."
    go run cmd/audio-router/main.go

# Generate sample Audio Router Hub configuration
router-config:
    @echo "⚙️ Generating Audio Router Hub configuration..."
    go run cmd/audio-router/main.go -generate-config

# Run Audio Router Hub with custom configuration
router-with-config:
    @echo "🎛️ Running Audio Router Hub with configuration..."
    go run cmd/audio-router/main.go -config audio-router.json -verbose

# Build Audio Router Hub binary
build-router:
    @echo "🔨 Building Audio Router Hub binary..."
    mkdir -p bin
    go build -o bin/audio-router cmd/audio-router/main.go

# =============================================================================
# Integration Testing Commands
# =============================================================================

# Run complete Docker-based integration test suite
test-integration: integration-build integration-up
    @echo "🧪 Running complete integration test suite..."
    sleep 15  # Wait for all services to be ready
    docker-compose -f test/integration/docker-compose.yml exec test-validator /app/run-integration-tests.sh || true
    just integration-down

# Build integration test containers
integration-build:
    @echo "🔨 Building Docker containers for integration testing..."
    docker-compose -f test/integration/docker-compose.yml build

# Start integration test environment
integration-up:
    @echo "🚀 Starting integration test environment..."
    docker-compose -f test/integration/docker-compose.yml up -d
    sleep 10  # Wait for services to start

# Stop integration test environment
integration-down:
    @echo "🛑 Stopping integration test environment..."
    docker-compose -f test/integration/docker-compose.yml down

# Run integration tests (assumes environment is running)
integration-run:
    @echo "🏃 Running integration tests..."
    docker-compose -f test/integration/docker-compose.yml exec test-validator /app/run-integration-tests.sh

# Show integration test logs
integration-logs:
    @echo "📋 Showing integration test logs..."
    docker-compose -f test/integration/docker-compose.yml logs

# Clean up integration test environment
integration-clean:
    @echo "🧹 Cleaning up integration test environment..."
    docker-compose -f test/integration/docker-compose.yml down -v
    docker system prune -f

# =============================================================================
# Tilt Development Environment Commands
# =============================================================================

# Start Tilt development environment (primary development workflow)
tilt-up:
    @echo "🚀 Starting Tilt development environment..."
    @echo "📊 Dashboard will be available at http://localhost:10350"
    @echo "🎛️ Audio Router will be available at http://localhost:9090"
    tilt up

# Quick alias for starting development environment
dev: tilt-up

# Stop Tilt development environment
tilt-down:
    @echo "🛑 Stopping Tilt development environment..."
    tilt down

# Show Tilt service logs
tilt-logs:
    @echo "📋 Showing Tilt service logs..."
    tilt logs

# Run integration tests in Tilt environment
tilt-test:
    @echo "🧪 Running integration tests in Tilt..."
    tilt trigger integration-tests

# Check Audio Router Hub status
tilt-status:
    @echo "📊 Checking Audio Router Hub status..."
    curl -f http://localhost:9090/status || echo "❌ Audio Router not available"

# Open Tilt dashboard in browser
tilt-dashboard:
    @echo "🌐 Opening Tilt dashboard..."
    @if command -v open >/dev/null 2>&1; then \
        open http://localhost:10350; \
    elif command -v xdg-open >/dev/null 2>&1; then \
        xdg-open http://localhost:10350; \
    else \
        echo "Please open http://localhost:10350 in your browser"; \
    fi

# =============================================================================
# Build and Release Commands
# =============================================================================

# Create release binaries for all platforms
release: fmt vet test
    @echo "📦 Creating release binaries..."
    mkdir -p dist
    @echo "Building for Linux amd64..."
    GOOS=linux GOARCH=amd64 go build -o dist/usrp-example-linux-amd64 cmd/examples/main.go
    @echo "Building for macOS amd64..."
    GOOS=darwin GOARCH=amd64 go build -o dist/usrp-example-darwin-amd64 cmd/examples/main.go
    @echo "Building for macOS arm64..."  
    GOOS=darwin GOARCH=arm64 go build -o dist/usrp-example-darwin-arm64 cmd/examples/main.go
    @echo "Building for Windows amd64..."
    GOOS=windows GOARCH=amd64 go build -o dist/usrp-example-windows-amd64.exe cmd/examples/main.go
    @echo "✅ Release binaries created in dist/"

# =============================================================================
# Development Setup and Workflows
# =============================================================================

# One-time setup for new developers
setup:
    @echo "🎵 Setting up USRP Audio Router Hub development environment..."
    @echo ""
    @echo "📋 Checking prerequisites..."
    @if ! command -v go >/dev/null 2>&1; then \
        echo "❌ Go not found. Please install Go 1.25+"; \
        exit 1; \
    fi
    @echo "✅ Go found: $(shell go version)"
    
    @if ! command -v docker >/dev/null 2>&1; then \
        echo "❌ Docker not found. Please install Docker"; \
        echo "   macOS: brew install colima docker"; \
        echo "   Linux: curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh"; \
        exit 1; \
    fi
    @echo "✅ Docker found: $(shell docker --version)"
    
    @if ! command -v kubectl >/dev/null 2>&1; then \
        echo "⚠️  kubectl not found. Install for Tilt development:"; \
        echo "   macOS: brew install kubectl"; \
        echo "   Linux: See https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/"; \
    else \
        echo "✅ kubectl found: $(shell kubectl version --client --short 2>/dev/null || echo 'kubectl available')"; \
    fi
    
    @if ! command -v tilt >/dev/null 2>&1; then \
        echo "⚠️  Tilt not found. Install for development environment:"; \
        echo "   curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash"; \
    else \
        echo "✅ Tilt found: $(shell tilt version 2>/dev/null | head -1 || echo 'Tilt available')"; \
    fi
    
    @echo ""
    @echo "📦 Installing Go dependencies..."
    just deps
    
    @echo ""
    @echo "🧪 Running initial tests..."
    just test
    
    @echo ""
    @echo "⚙️ Generating sample configuration..."
    just router-config
    
    @echo ""
    @echo "🎉 Setup complete! Next steps:"
    @echo ""
    @echo "🚀 Start development environment:"
    @echo "   just dev                    # Start Tilt with live reload"
    @echo ""
    @echo "🧪 Run tests:"
    @echo "   just test                   # Unit tests"
    @echo "   just tilt-test             # Integration tests (requires 'just dev')"
    @echo ""
    @echo "🎛️ Try the Audio Router Hub:"
    @echo "   just router                # Run with default config"
    @echo "   just router-with-config    # Run with custom config"
    @echo ""
    @echo "📚 Documentation:"
    @echo "   docs/REQUIREMENTS.md       # System requirements"
    @echo "   test/tilt/README.md        # Development environment guide"
    @echo ""
    @echo "Happy Amateur Radio Development! 📻 73!"

# Quick development workflow - setup and start
quick-start: setup dev

# Full CI-like test suite
ci: fmt vet lint test test-integration

# Development quality check (faster than full CI)
check: fmt vet test

# =============================================================================
# Utility Commands
# =============================================================================

# Show current system status for debugging
status:
    @echo "🔍 USRP Audio Router Hub - System Status"
    @echo "======================================="
    @echo ""
    @echo "📋 Go Environment:"
    @go version 2>/dev/null || echo "❌ Go not available"
    @echo "GOPATH: ${GOPATH:-not set}"
    @echo "GOOS: $(shell go env GOOS)"
    @echo "GOARCH: $(shell go env GOARCH)"
    @echo ""
    @echo "🐳 Container Environment:" 
    @docker --version 2>/dev/null || echo "❌ Docker not available"
    @kubectl version --client --short 2>/dev/null || echo "❌ kubectl not available"
    @tilt version 2>/dev/null | head -1 || echo "❌ Tilt not available"
    @echo ""
    @echo "📊 Services Status:"
    @if curl -s -f http://localhost:9090/status >/dev/null 2>&1; then \
        echo "✅ Audio Router Hub: http://localhost:9090/status"; \
    else \
        echo "❌ Audio Router Hub: Not running"; \
    fi
    @if curl -s -f http://localhost:10350 >/dev/null 2>&1; then \
        echo "✅ Tilt Dashboard: http://localhost:10350"; \
    else \
        echo "❌ Tilt Dashboard: Not running"; \
    fi

# Show project structure and key files
info:
    @echo "🎵 USRP Audio Router Hub - Project Information"
    @echo "============================================="
    @echo ""
    @echo "📁 Project Structure:"
    @find . -type f -name "*.go" | head -10 | sed 's/^/  /'
    @echo "  ... and more Go files"
    @echo ""
    @echo "📋 Key Commands:"
    @echo "  just setup              # One-time development setup"
    @echo "  just dev                # Start development environment"  
    @echo "  just router             # Run Audio Router Hub"
    @echo "  just test               # Run all tests"
    @echo ""
    @echo "📚 Documentation:"
    @find docs/ -name "*.md" 2>/dev/null | sed 's/^/  /' || echo "  docs/ directory not found"
    @echo ""
    @echo "🧪 Testing:"
    @echo "  test/tilt/              # Tilt development environment"
    @echo "  test/integration/       # Docker-based integration tests"

# List all available recipes (same as default, but explicit)
list:
    @just --list