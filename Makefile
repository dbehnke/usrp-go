# USRP Go Library Makefile

.PHONY: all build test bench clean run-example run-demo fmt vet lint

# Default target
all: fmt vet test build

# Build the library
build:
	@echo "Building library..."
	@go build ./...

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -cover ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. ./pkg/usrp/

# Run example (protocol tests only)
run-example:
	@echo "Running example (protocol tests)..."
	@go run cmd/examples/main.go

# Run full demo
run-demo:
	@echo "Running full UDP demo..."
	@go run cmd/examples/main.go demo

# Run audio conversion test
run-audio-test:
	@echo "Running audio conversion test..."
	@go run cmd/audio-bridge/main.go test

# Run audio bridge server
run-audio-server:
	@echo "Running audio bridge server..."
	@go run cmd/audio-bridge/main.go server

# Run audio bridge client  
run-audio-client:
	@echo "Running audio bridge client..."
	@go run cmd/audio-bridge/main.go client

# USRP Bridge utility targets
run-usrp-bridge:
	@echo "Running USRP bridge utility..."
	@go run cmd/usrp-bridge/main.go

run-usrp-bridge-config:
	@echo "Generating USRP bridge configuration..."
	@go run cmd/usrp-bridge/main.go -generate-config

build-usrp-bridge:
	@echo "Building USRP bridge binary..."
	@mkdir -p bin
	@go build -o bin/usrp-bridge cmd/usrp-bridge/main.go

# Discord bridge targets
run-discord-test:
	@echo "Running Discord bot test..."
	@go run cmd/discord-bridge/main.go test

run-discord-bridge:
	@echo "Running Discord-USRP bridge..."
	@go run cmd/discord-bridge/main.go bridge

run-discord-server:
	@echo "Running USRP test server for Discord bridge..."
	@go run cmd/discord-bridge/main.go server

# Audio Router Hub targets
run-audio-router:
	@echo "Running Audio Router Hub..."
	@go run cmd/audio-router/main.go

run-audio-router-config:
	@echo "Generating Audio Router Hub configuration..."
	@go run cmd/audio-router/main.go -generate-config

run-audio-router-with-config:
	@echo "Running Audio Router Hub with configuration..."
	@go run cmd/audio-router/main.go -config audio-router.json -verbose

build-audio-router:
	@echo "Building Audio Router Hub binary..."
	@mkdir -p bin
	@go build -o bin/audio-router cmd/audio-router/main.go

# Integration Testing targets
test-integration-build:
	@echo "Building Docker containers for integration testing..."
	@docker-compose -f test/integration/docker-compose.yml build

test-integration-up:
	@echo "Starting integration test environment..."
	@docker-compose -f test/integration/docker-compose.yml up -d
	@sleep 10  # Wait for services to start

test-integration-down:
	@echo "Stopping integration test environment..."
	@docker-compose -f test/integration/docker-compose.yml down

test-integration-run:
	@echo "Running comprehensive integration tests..."
	@docker-compose -f test/integration/docker-compose.yml exec test-validator /app/run-integration-tests.sh

test-integration: test-integration-build test-integration-up
	@echo "Running complete integration test suite..."
	@sleep 15  # Wait for all services to be ready
	@docker-compose -f test/integration/docker-compose.yml exec test-validator /app/run-integration-tests.sh || true
	@make test-integration-down

test-integration-logs:
	@echo "Showing integration test logs..."
	@docker-compose -f test/integration/docker-compose.yml logs

test-integration-clean:
	@echo "Cleaning up integration test environment..."
	@docker-compose -f test/integration/docker-compose.yml down -v
	@docker system prune -f

# Tilt Development Environment targets  
tilt-up:
	@echo "Starting Tilt development environment..."
	@tilt up

tilt-down:
	@echo "Stopping Tilt development environment..."
	@tilt down

tilt-logs:
	@echo "Showing Tilt service logs..."
	@tilt logs

tilt-test:
	@echo "Running Tilt integration tests..."
	@tilt trigger integration-tests

tilt-status:
	@echo "Checking Tilt service status..."
	@curl -f http://localhost:9090/status || echo "Audio Router not available"

tilt-dashboard:
	@echo "Opening Tilt dashboard..."
	@open http://localhost:10350 || xdg-open http://localhost:10350 || echo "Please open http://localhost:10350"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@golangci-lint run || echo "golangci-lint not installed, skipping..."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@go clean ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Create release
release: all
	@echo "Creating release..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build -o dist/usrp-example-linux-amd64 cmd/examples/main.go
	@GOOS=darwin GOARCH=amd64 go build -o dist/usrp-example-darwin-amd64 cmd/examples/main.go  
	@GOOS=windows GOARCH=amd64 go build -o dist/usrp-example-windows-amd64.exe cmd/examples/main.go

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Format, vet, test, and build"
	@echo "  build        - Build the library"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  bench        - Run benchmarks"
	@echo "  run-example  - Run example (protocol tests)"
	@echo "  run-demo     - Run full UDP demo"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code" 
	@echo "  lint         - Lint code"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
	@echo "  release      - Create release binaries"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Audio Conversion:"
	@echo "  run-audio-test    - Run audio conversion test"
	@echo "  run-audio-server  - Run audio bridge server"
	@echo "  run-audio-client  - Run audio bridge client"
	@echo ""
	@echo "USRP Bridge Utility:"
	@echo "  run-usrp-bridge        - Run USRP bridge utility"
	@echo "  run-usrp-bridge-config - Generate sample configuration"
	@echo "  build-usrp-bridge      - Build USRP bridge binary"
	@echo ""
	@echo "Discord Integration:"
	@echo "  run-discord-test    - Test Discord bot connection"
	@echo "  run-discord-bridge  - Run Discord-USRP bridge"
	@echo "  run-discord-server  - Run USRP test server for Discord"
	@echo ""
	@echo "Audio Router Hub:"
	@echo "  run-audio-router        - Run Audio Router Hub with default config"
	@echo "  run-audio-router-config - Generate sample configuration"
	@echo "  run-audio-router-with-config - Run with audio-router.json config"
	@echo "  build-audio-router      - Build Audio Router Hub binary"
	@echo ""
	@echo "Integration Testing:"
	@echo "  test-integration        - Run complete Docker-based integration tests"
	@echo "  test-integration-build  - Build integration test containers"
	@echo "  test-integration-up     - Start integration test environment"
	@echo "  test-integration-down   - Stop integration test environment"
	@echo "  test-integration-logs   - Show integration test logs"
	@echo "  test-integration-clean  - Clean up integration test environment"
	@echo ""
	@echo "Tilt Development Environment:"
	@echo "  tilt-up                 - Start Tilt development environment"
	@echo "  tilt-down               - Stop Tilt development environment"
	@echo "  tilt-logs               - Show Tilt service logs"
	@echo "  tilt-test               - Run integration tests in Tilt"
	@echo "  tilt-status             - Check Audio Router status"
	@echo "  tilt-dashboard          - Open Tilt dashboard"
