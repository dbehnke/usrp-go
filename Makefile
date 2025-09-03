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
