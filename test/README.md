# 🧪 USRP Audio Router Hub - Integration Testing

Comprehensive Dagger-based integration testing suite for validating the complete audio routing platform with realistic service simulations and amateur radio protocol compliance testing.

## 🎯 Testing Architecture

The integration testing uses **Dagger** for containerized, reproducible testing that works consistently across local development and CI environments.

```
┌─────────────────────────────────────────────────────────────┐
│              Dagger Integration Test Pipeline               │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐    ┌──────────────────┐    ┌──────────┐  │
│  │ AllStarLink  │◄──►│  Audio Router    │◄──►│WhoTalkie │  │
│  │ Mock Server  │    │      Hub         │    │   Mock   │  │
│  │   (USRP)     │    │ (System Under    │    │ Service  │  │
│  │              │    │     Test)        │    │          │  │
│  └──────────────┘    │                  │    └──────────┘  │
│                       │                  │                  │
│  ┌──────────────┐    │                  │    ┌──────────┐  │
│  │   Discord    │◄──►│                  │◄──►│ Generic  │  │
│  │ Voice Mock   │    │                  │    │UDP/TCP   │  │
│  │   Gateway    │    └──────────────────┘    │ Service  │  │
│  └──────────────┘                            └──────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Audio Test Signal Generator                  │  │
│  │   • Multiple test patterns (sine, voice, DTMF)     │  │
│  │   • Realistic PTT timing and amateur radio data    │  │
│  │   • Format validation (PCM, Opus, μ-law)           │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Run Integration Tests

```bash
# Run complete integration test suite via Dagger
just dagger-test

# Get interactive shell in test container for debugging
just dagger-test-shell
```

### Prerequisites

- **Dagger CLI** - Automatically installed in CI, or install locally:
  ```bash
  curl -L https://dl.dagger.io/dagger/install.sh | sh
  sudo mv ./bin/dagger /usr/local/bin/dagger
  ```
- **Go 1.25+**
- **golangci-lint** (automatically installed in test container)
- **just** command runner

## 📋 Test Categories

The integration test suite includes **26+ comprehensive test cases**:

### 🔧 Core USRP Protocol Tests
- USRP protocol compatibility validation
- All packet types: Voice, DTMF, TLV, Ping, μ-law, ADPCM
- Binary serialization/deserialization accuracy
- Amateur radio packet format compliance

### 🎵 Audio Bridge Tests  
- Audio format conversion (PCM ↔ Opus ↔ μ-law)
- Real-time audio streaming validation
- Latency and quality measurements

### 🌉 Service Bridge Tests
- USRP-to-internet service bridging
- Multi-destination audio routing
- Discord voice integration
- WhoTalkie protocol compatibility

### 📦 Go Module & Code Quality Tests
- Module validation and dependency checks
- Code compilation across all packages
- Unit test execution
- Code formatting and linting (golangci-lint)
- Performance benchmarks

### 🎮 Mock Service Validation
- AllStarLink node simulation with test patterns
- Discord bot mock interactions
- Generic UDP/TCP service testing
- Realistic amateur radio timing and behavior

## 🔍 Test Execution Flow

1. **Environment Setup** - Dagger creates clean containerized environment
2. **Dependency Installation** - Go, golangci-lint, just, FFmpeg
3. **Code Quality Checks** - Linting and formatting validation
4. **Protocol Tests** - USRP packet format compliance
5. **Audio Tests** - Format conversion and streaming validation
6. **Service Integration** - End-to-end service communication
7. **Performance Tests** - Benchmarks and latency measurements
8. **Cleanup** - Automatic container cleanup

## 📊 Test Results

### Success Criteria
- ✅ All 26+ test cases pass
- ✅ Code quality checks pass (golangci-lint)
- ✅ No memory leaks or panics detected
- ✅ Performance benchmarks within acceptable ranges
- ✅ Amateur radio protocol compliance verified

### Example Output
```bash
🧪 Running USRP Go Integration Tests
=====================================

📋 Environment Check
Go version: go1.25 linux/amd64
Working directory: /work
Available commands: examples audio-bridge usrp-bridge discord-bridge audio-router

🔧 Core USRP Protocol Tests
→ Testing USRP protocol compatibility... ✓
→ Testing USRP packet format demonstration... ✓

🧹 Code Quality Prerequisites  
→ Testing just command runner availability... ✓
→ Testing golangci-lint availability... ✓
→ Testing Go code linting (via just)... ✓

📦 Go Module Tests
→ Testing Go module validation... ✓
→ Testing Go code compilation... ✓
→ Testing Go unit tests... ✓
→ Testing Go code formatting check... ✓
→ Testing Go code vetting... ✓

🎵 Audio Bridge Tests
→ Testing Audio bridge help... ✓
→ Testing Audio bridge test mode... ✓

🌉 USRP Bridge Tests
→ Testing USRP bridge help... ✓
→ Testing USRP bridge config generation... ✓

📊 Performance Tests
→ Testing USRP benchmarks... ✓

🔍 Code Quality Tests
→ Testing Go mod tidy check... ✓
→ Testing No panic() calls in main code... ✓

📁 Project Structure Validation
→ Testing Required directories exist... ✓
→ Testing Main packages build... ✓
→ Testing README.md exists and non-empty... ✓

=================================================================
🎉 All integration tests passed!
✓ 26/26 tests successful

📻 USRP Protocol Library is ready for amateur radio operations!
73, good DX! 📡
=================================================================
```

## 🔧 Development & Debugging

### Interactive Testing
```bash
# Get shell access to test container
just dagger-test-shell

# Inside the container, run individual tests:
go test ./pkg/usrp/
go run cmd/examples/main.go
golangci-lint run
```

### Local Development with Tilt
For interactive development with live reload:
```bash
# Start full development environment
just dev                 # Starts Tilt environment
just tilt-dashboard      # Opens http://localhost:10350
```

### CI Integration
The integration tests run automatically in GitHub Actions:
- **Trigger**: Push to `main` or `feature/*` branches, PRs to `main`
- **Environment**: Ubuntu Latest with Dagger CLI
- **Timeout**: 15 minutes
- **Branch Protection**: Required to pass before merging

## 🏗️ Architecture Details

### Dagger Pipeline (`ci/dagger/main.go`)
- **Language**: Go with Dagger SDK
- **Base Image**: `golang:1.25`
- **Dependencies**: Auto-installed golangci-lint, just, FFmpeg
- **Execution**: Runs `test/containers/test-validator/run-integration-tests.sh`
- **Caching**: Leverages Dagger's intelligent build caching

### Test Container Features
- **Isolated Environment**: Clean container per test run
- **Tool Availability**: All required tools pre-installed
- **Amateur Radio Focus**: USRP protocol compliance testing
- **Performance Monitoring**: Built-in benchmark execution
- **Quality Gates**: Comprehensive linting and formatting checks

## 📚 Resources

- **Integration Test Script**: `test/containers/test-validator/run-integration-tests.sh`
- **Dagger Pipeline**: `ci/dagger/main.go`
- **GitHub Workflow**: `.github/workflows/test-integration.yml`
- **Development Environment**: `test/tilt/README.md`
- **Project Documentation**: `README.md`

---

**73, Good DX!** 📻  
*Integration testing for amateur radio experimentation under FCC Part 97 regulations.*