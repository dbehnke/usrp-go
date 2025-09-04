# 🧪 USRP Audio Router Hub - Integration Testing

Comprehensive Docker-based integration testing suite for validating the complete audio routing platform with realistic service simulations.

## 🎯 Testing Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                Docker Test Environment                      │
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
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            Test Validation Engine                    │  │
│  │   • End-to-end audio flow verification              │  │
│  │   • Performance monitoring and metrics              │  │
│  │   • Quality analysis and packet loss detection     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### **Run Complete Integration Test Suite**
```bash
# Run all tests with automatic cleanup
make test-integration

# Or step-by-step for debugging
make test-integration-build   # Build containers
make test-integration-up      # Start environment  
make test-integration-run     # Run tests
make test-integration-logs    # View logs
make test-integration-down    # Stop environment
```

### **Interactive Testing**
```bash
# Start test environment for manual testing
make test-integration-up

# Audio Router Hub status
curl http://localhost:9090/status

# View real-time logs
make test-integration-logs

# Monitor network traffic
docker-compose -f test/integration/docker-compose.yml exec audio-router \
  tcpdump -i any -n 'udp'

# Cleanup when done
make test-integration-clean
```

## 🧪 Test Components

### **1. Mock AllStarLink Server** (`test/containers/allstar-mock/`)

**Realistic USRP packet generation:**
- ✅ **Proper USRP Headers**: 32-byte AllStarLink-compatible headers
- ✅ **Audio Test Patterns**: Sine waves, DTMF, frequency sweeps, white noise
- ✅ **PTT Simulation**: Realistic push-to-talk timing patterns
- ✅ **Amateur Radio Metadata**: Callsigns, talk groups, sequence numbers
- ✅ **Multiple Node Support**: Different node IDs and configurations

**Test Patterns Available:**
```bash
# 440Hz sine wave (default)
docker run allstar-mock -pattern sine_440hz -callsign W1AW

# 1kHz sine wave  
docker run allstar-mock -pattern sine_1khz -callsign W2XYZ

# DTMF digit sequence
docker run allstar-mock -pattern dtmf_sequence -callsign KC1ABC

# Frequency sweep (300Hz-3kHz)
docker run allstar-mock -pattern frequency_sweep -callsign N0CALL

# White noise for quality testing
docker run allstar-mock -pattern white_noise -callsign VK1TEST
```

### **2. Mock WhoTalkie Service** (`test/containers/whotalkie-mock/`)

**Internet service simulation:**
- ✅ **Opus Audio Handling**: Receives and validates Opus-encoded packets
- ✅ **Format Conversion**: FFmpeg integration for format testing
- ✅ **Multiple Instances**: Primary and backup service simulation
- ✅ **Quality Metrics**: Audio quality analysis and logging
- ✅ **Health Monitoring**: Service health and performance metrics

### **3. Mock Discord Voice Gateway** (`test/containers/discord-mock/`)

**Discord protocol simulation:**
- ✅ **WebSocket Protocol**: Simulates Discord voice gateway behavior
- ✅ **Voice Channel Management**: Bot connection and audio streaming
- ✅ **Audio Resampling**: 48kHz stereo Discord format support
- ✅ **Latency Simulation**: Realistic network conditions
- ✅ **Multiple Bot Support**: Multiple Discord bot instances

### **4. Generic UDP/TCP Services** (`test/containers/generic-mock/`)

**Custom protocol testing:**
- ✅ **Both Protocols**: UDP and TCP service simulation  
- ✅ **Raw Audio Streaming**: Direct audio packet handling
- ✅ **Custom Formats**: Configurable audio formats and protocols
- ✅ **Load Testing**: High-throughput testing capabilities

## 📊 Test Scenarios

### **Basic Audio Flow Tests**
```bash
# Test: AllStarLink → Audio Router → WhoTalkie
./test/scripts/run-integration-tests.sh test_allstar_to_whotalkie

# Expected: 
#   - USRP packets received from AllStar mock
#   - Format conversion PCM → Opus
#   - Opus packets delivered to WhoTalkie mock
#   - Packet loss < 1%
#   - Latency < 50ms
```

### **Multi-Service Routing Tests**
```bash
# Test: N-to-N routing with priority management
./test/scripts/run-integration-tests.sh test_multi_service_routing

# Expected:
#   - Multiple services sending/receiving simultaneously
#   - Priority-based preemption working
#   - No audio loops or feedback
#   - Statistics accuracy
```

### **Format Conversion Matrix Tests**
```bash
# Test: All format combinations
./test/scripts/run-integration-tests.sh test_audio_quality

# Expected:
#   - PCM → Opus → PCM roundtrip quality > 95%
#   - μ-law → Opus → μ-law legacy support
#   - Sample rate conversions (8kHz ↔ 48kHz)
#   - Minimal quality degradation
```

### **Stress and Load Tests**
```bash
# Test: High concurrent load
./test/scripts/run-integration-tests.sh test_performance

# Expected:
#   - Handle > 100 concurrent streams
#   - Memory usage stable (no leaks)
#   - CPU usage reasonable
#   - Packet processing > 500/sec
```

### **Error Condition Tests**
```bash
# Test: Error handling and recovery
./test/scripts/run-integration-tests.sh test_error_conditions

# Expected:
#   - Graceful handling of invalid packets
#   - Service failure detection and recovery
#   - Network timeout handling
#   - Resource cleanup on errors
```

## 📈 Test Validation

### **Audio Quality Metrics**
- **Packet Loss**: < 1% acceptable
- **Audio Correlation**: > 95% for sine wave tests  
- **Latency**: < 50ms end-to-end
- **Jitter**: < 10ms variance
- **Format Conversion Quality**: SNR > 40dB

### **Performance Metrics**
- **Throughput**: > 500 packets/second
- **Concurrent Services**: Support 10+ simultaneous services
- **Memory Usage**: Stable, no leaks over 1 hour test
- **CPU Usage**: < 50% under normal load
- **Network Bandwidth**: Efficient utilization with compression

### **Reliability Metrics** 
- **Service Availability**: 99.9% uptime during tests
- **Error Recovery**: Automatic recovery from transient failures
- **Resource Cleanup**: Proper cleanup on shutdown
- **Configuration Validation**: Invalid configs handled gracefully

## 🐳 Container Details

### **AllStarLink Mock Container**
```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY allstar-mock.go go.mod go.sum ./
RUN go build -o allstar-mock allstar-mock.go

FROM alpine:latest
COPY --from=builder /build/allstar-mock ./
EXPOSE 34001/udp
ENTRYPOINT ["./allstar-mock"]
```

**Key Features:**
- **1,000+ lines of Go code** for realistic USRP simulation
- **Multiple audio patterns** for comprehensive testing
- **Statistics reporting** with detailed metrics
- **Configurable parameters** (node ID, callsign, patterns)
- **Network simulation** with configurable latency/jitter

### **Test Configuration**

**Multi-Service Test Configuration** (`test/integration/configs/audio-router-test.json`):
```json
{
  "services": [
    {
      "id": "allstar_mock_1",
      "type": "usrp", 
      "name": "AllStarLink Mock Node 12345",
      "network": {"listen_port": 32001, "remote_addr": "allstar-mock-1"},
      "routing": {"priority": 5, "send_to_types": ["whotalkie", "discord"]}
    },
    {
      "id": "whotalkie_primary",
      "type": "whotalkie",
      "audio": {"format": "opus", "sample_rate": 48000},
      "routing": {"priority": 3, "send_to_types": ["usrp"]}
    },
    // ... more services
  ]
}
```

## 🔧 Development and Debugging

### **Adding New Test Scenarios**
1. **Create test function** in `test/scripts/run-integration-tests.sh`
2. **Add to test array** in main() function
3. **Update Docker configuration** if new services needed
4. **Document expected behavior** and validation criteria

### **Debugging Failed Tests**
```bash
# View detailed logs
make test-integration-logs

# Check specific service logs
docker-compose -f test/integration/docker-compose.yml logs audio-router
docker-compose -f test/integration/docker-compose.yml logs allstar-mock-1

# Interactive debugging
docker-compose -f test/integration/docker-compose.yml exec audio-router sh

# Network debugging
docker-compose -f test/integration/docker-compose.yml exec audio-router \
  tcpdump -i any -n 'udp and port 32001'
```

### **Performance Profiling**
```bash
# CPU profiling of Audio Router Hub
docker-compose -f test/integration/docker-compose.yml exec audio-router \
  go tool pprof http://localhost:9090/debug/pprof/profile

# Memory profiling
docker-compose -f test/integration/docker-compose.yml exec audio-router \
  go tool pprof http://localhost:9090/debug/pprof/heap
```

## 🎯 Benefits of Integration Testing

### **For Development**
- **End-to-end validation**: Complete audio flow testing
- **Realistic scenarios**: Real protocol behavior simulation
- **Performance validation**: Load and stress testing
- **Regression prevention**: Automated testing on every change

### **For Deployment**
- **Configuration validation**: Test configs before production
- **Service compatibility**: Verify integration with external services
- **Scalability testing**: Validate performance at scale
- **Reliability assurance**: Error condition and recovery testing

### **For Users**
- **Quality assurance**: Verified audio quality and reliability
- **Documentation**: Clear examples of expected behavior
- **Troubleshooting**: Known-good configurations and diagnostics
- **Confidence**: Proven integration with amateur radio systems

## 🌟 Future Enhancements

### **Additional Test Scenarios**
- **WebRTC integration testing**
- **SIP protocol testing**  
- **P25/DMR protocol bridging**
- **Mobile app integration testing**

### **Advanced Audio Analysis**
- **Spectral analysis** of audio quality
- **Voice activity detection** testing
- **Echo and feedback detection**
- **Adaptive bitrate testing**

### **Monitoring and Alerting**
- **Grafana dashboards** for real-time monitoring
- **Prometheus metrics** collection
- **Alert on test failures**
- **Performance regression detection**

---

**🎉 The integration testing suite provides comprehensive validation of the entire USRP Audio Router Hub platform, ensuring production-ready quality and reliability for amateur radio applications!**