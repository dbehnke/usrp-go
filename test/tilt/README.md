# 🚀 Tilt Development Environment for USRP Audio Router Hub

**Live development environment with automatic reloading, real-time monitoring, and interactive testing for amateur radio audio routing.**

## 🎯 What is This?

Tilt provides an **excellent development experience** for our complex multi-service audio routing platform. Instead of manually managing Docker containers, Tilt gives you:

- **🔄 Live Reload**: Code changes trigger automatic rebuilds and restarts
- **📊 Visual Dashboard**: Beautiful UI showing all service status and logs
- **🐛 Easy Debugging**: One-click access to logs and service introspection
- **⚡ Fast Iteration**: Optimized build caching and incremental updates

## 🚀 Quick Start

### **Prerequisites**
```bash
# Install Tilt
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

# Verify local Kubernetes (Docker Desktop, kind, or minikube)
kubectl cluster-info

# Verify Docker
docker version
```

### **Start Development Environment**
```bash
# Start everything with live reload
just dev

# Or directly with Tilt
tilt up

# Open Tilt dashboard (automatic in most cases)
just tilt-dashboard
```

### **Access Points**
Once running, you'll have access to:
- **🎛️ Tilt Dashboard**: http://localhost:10350 (main control center)
- **📊 Audio Router Hub**: http://localhost:9090/status
- **📻 AllStar Mock 1**: UDP port 34001 (440Hz sine wave)
- **📻 AllStar Mock 2**: UDP port 34002 (1kHz sine wave)
- **🌐 WhoTalkie Mocks**: UDP ports 8080, 8081
- **🎮 Discord Mock**: TCP port 8082

## 🎵 Development Workflow

### **The Magic of Live Reload**
```bash
# 1. Make changes to Go code
vim cmd/audio-router/main.go

# 2. Save the file
# → Tilt automatically detects changes
# → Rebuilds only what's needed  
# → Restarts the Audio Router
# → Updates dashboard status
# → All in ~2-3 seconds! ⚡
```

### **Real-time Monitoring**
The Tilt dashboard shows:
- **Service Status**: Green/red indicators for all services
- **Build Progress**: Real-time build logs and status
- **Service Logs**: Aggregated logs from all services with filtering
- **Resource Usage**: CPU, memory, network for each service
- **Port Forwards**: Easy access to all service endpoints

### **Interactive Testing**
```bash
# Run integration tests 
just tilt-test

# Or trigger specific tests from dashboard
tilt trigger integration-tests
tilt trigger audio-quality-check
tilt trigger performance-profile

# Manual testing
curl http://localhost:9090/status
nc -u localhost 34001  # Test AllStar mock
```

## 🏗️ Architecture Overview

### **Service Topology in Kubernetes**
```
┌─────────────────────────────────────────────────┐
│                Kubernetes Cluster               │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌──────────────┐    ┌─────────────────┐       │
│  │AllStar Mock 1│◄──►│ Audio Router    │       │
│  │ (sine 440Hz) │    │      Hub        │       │
│  │              │    │ (Live Reload)   │       │
│  └──────────────┘    │                 │       │
│                       │                 │       │
│  ┌──────────────┐    │                 │       │
│  │AllStar Mock 2│◄──►│                 │       │
│  │ (sine 1kHz)  │    └─────────────────┘       │
│  └──────────────┘                              │
│                                                 │
│  ┌──────────────┐    ┌─────────────────┐       │
│  │ WhoTalkie    │◄──►│    Monitoring   │       │
│  │   Mocks      │    │   (Optional)    │       │
│  └──────────────┘    └─────────────────┘       │
│                                                 │
│  ┌──────────────┐                              │
│  │ Discord Mock │                              │
│  │   Gateway    │                              │
│  └──────────────┘                              │
└─────────────────────────────────────────────────┘
```

### **Live Reload Architecture**
```
Local File Change → Tilt Detection → Docker Build → K8s Deploy → Service Restart
     (instant)       (~100ms)        (~2s)         (~1s)         (~500ms)
                                                                      ↓
                                                              Dashboard Update
```

## 📁 File Structure

### **Core Tilt Configuration**
```
Tiltfile                              # Main orchestration config
test/tilt/
├── README.md                         # This file
├── k8s/                              # Kubernetes manifests
│   ├── audio-router.yaml            # Audio Router deployment + config
│   ├── allstar-mocks.yaml           # AllStar mock services
│   ├── whotalkie-mocks.yaml         # WhoTalkie mock services
│   ├── discord-mock.yaml            # Discord mock service
│   ├── prometheus.yaml              # Monitoring (optional)
│   └── grafana.yaml                 # Metrics visualization
├── dockerfiles/                     # Optimized development Dockerfiles
│   ├── Dockerfile.audio-router      # Multi-stage with live reload
│   ├── Dockerfile.allstar-mock      # Mock AllStar server
│   └── Dockerfile.whotalkie-mock    # Mock WhoTalkie service
├── scripts/                         # Development automation
│   ├── run-tests.sh                 # Integration test runner
│   ├── validate-audio-quality.sh    # Audio quality validation
│   ├── capture-traffic.sh           # Network analysis
│   └── profile-performance.sh       # Performance profiling
└── services/                        # Mock service implementations
    ├── whotalkie-mock/              # WhoTalkie simulator
    ├── discord-mock/                # Discord voice gateway mock
    └── audio-generator/             # Test signal generator
```

## 🧪 Testing and Validation

### **Automated Integration Tests**
```bash
# Run comprehensive test suite
make tilt-test

# Individual test scenarios
./test/tilt/scripts/run-tests.sh health           # Health checks only
./test/tilt/scripts/run-tests.sh packet-flow      # Packet flow analysis
./test/tilt/scripts/run-tests.sh performance      # Performance testing

# Environment variables for testing
VERBOSE=true TEST_DURATION=120 make tilt-test
```

### **Manual Testing Scenarios**

#### **Test AllStar → Audio Router Flow**
```bash
# Check AllStar mock is generating packets
nc -u -l 32001 | hexdump -C  # Listen for USRP packets

# Verify Audio Router receives and processes
curl http://localhost:9090/status | jq .statistics

# Monitor logs in real-time
tilt logs audio-router --follow
```

#### **Test Multi-Service Routing**
```bash
# Send test DTMF from one AllStar mock
echo -e "\x55\x53\x52\x50..." | nc -u localhost 34001

# Verify routing to WhoTalkie mock
nc -u -l 8080 | hexdump -C

# Check routing statistics
curl http://localhost:9090/status | jq .services
```

### **Audio Quality Testing**
```bash
# Generate and validate audio patterns
tilt trigger audio-quality-check

# Manual audio analysis
./test/tilt/scripts/validate-audio-quality.sh --pattern sine_440hz
```

## 🔧 Development Tips & Tricks

### **Efficient Development Workflow**

#### **1. Use Service Labels for Organization**
Tilt groups services by labels in the dashboard:
- **Core**: Audio Router Hub (main system under test)
- **Mocks**: AllStar, WhoTalkie, Discord simulators  
- **Monitoring**: Prometheus, Grafana (if enabled)
- **Testing**: Integration test runners and validators

#### **2. Live Debugging**
```bash
# Exec into running Audio Router container
kubectl exec -it deployment/audio-router -- sh

# View real-time logs with filtering
tilt logs audio-router --follow | grep "ERROR\|WARN"

# Monitor network traffic
kubectl exec -it deployment/audio-router -- tcpdump -i any udp
```

#### **3. Configuration Hot-Reload**
```bash
# Edit configuration
vim test/tilt/k8s/audio-router.yaml

# Tilt automatically detects and applies changes
# ConfigMap updates trigger pod restart with new config
```

#### **4. Performance Profiling**
```bash
# CPU profiling
tilt trigger performance-profile

# Manual profiling
go tool pprof http://localhost:9090/debug/pprof/profile

# Memory analysis
curl http://localhost:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

### **Troubleshooting Common Issues**

#### **Services Not Starting**
```bash
# Check Kubernetes status
kubectl get pods
kubectl describe pod audio-router-xxx

# View build logs in Tilt dashboard
# Check resource constraints (CPU/memory)
```

#### **Network Connectivity Issues**
```bash
# Test service-to-service communication
kubectl exec -it deployment/audio-router -- nc -z allstar-mock-1-service 34001

# Check service discovery
kubectl get svc
nslookup allstar-mock-1-service
```

#### **Live Reload Not Working**
```bash
# Check file watching (common issue with mounted volumes)
tilt logs --follow | grep "File changed"

# Manual rebuild if needed
tilt trigger audio-router-build
```

## 🎯 Configuration Options

### **Tilt Configuration Variables**
Set in `Tiltfile` or via environment variables:
```bash
# Enable/disable monitoring stack
tilt up -- --enable_monitoring=false

# Change audio test pattern
tilt up -- --audio_pattern=dtmf_sequence

# Extended test duration
tilt up -- --test_duration=600

# Enable debug logging
tilt up -- --enable_debug_logging=true
```

### **Development vs Production**
```bash
# Development mode (default)
tilt up

# Production-like testing (optimized builds)
TILT_ENV=production tilt up
```

## 📊 Monitoring and Metrics

### **Built-in Monitoring**
When `enable_monitoring=true`:
- **Prometheus**: http://localhost:9091 (metrics collection)
- **Grafana**: http://localhost:3000 (visualization, admin/admin)

### **Available Metrics**
- **Audio Router Performance**: Packets/sec, latency, errors
- **Service Health**: Uptime, connection status, response times
- **Resource Usage**: CPU, memory, network I/O per service
- **Amateur Radio Metrics**: PTT events, callsign activity, talk groups

### **Custom Dashboards**
Grafana includes pre-configured dashboards for:
- **Audio Flow Analysis**: Real-time packet flow visualization
- **Service Performance**: Response times and throughput
- **Resource Monitoring**: Infrastructure resource usage
- **Error Analysis**: Error rates and patterns

## 🚀 Advanced Features

### **Multi-Environment Testing**
```bash
# Test with different configurations
cp test/tilt/configs/audio-router-dev.json test/tilt/configs/audio-router-experimental.json
# Edit experimental config
tilt up -- --config_file=experimental
```

### **Load Testing**
```bash
# High-load scenario
tilt up -- --enable_audio_generator=true --test_duration=3600

# Monitor under load
tilt trigger performance-profile
```

### **Integration with CI/CD**
```bash
# Headless mode for CI
tilt ci

# Export test results
tilt trigger integration-tests 2>&1 | tee test-results.log
```

## 🎉 Benefits Over Traditional Docker Compose

| Feature | Docker Compose | Tilt | Winner |
|---------|----------------|------|---------|
| **Live Reload** | Manual rebuild | Automatic | 🏆 Tilt |
| **Visual Dashboard** | Command line only | Beautiful UI | 🏆 Tilt |
| **Service Dependencies** | Basic | Advanced K8s features | 🏆 Tilt |
| **Build Optimization** | Full rebuilds | Incremental | 🏆 Tilt |
| **Development Experience** | Good | Excellent | 🏆 Tilt |
| **Setup Complexity** | Simple | Moderate | 🏆 Docker Compose |
| **Resource Usage** | Lower | Higher (K8s) | 🏆 Docker Compose |

## 🎯 When to Use Tilt vs Docker Compose

### **Use Tilt For:**
- ✅ **Active Development**: Making frequent code changes
- ✅ **Complex Services**: Multi-service debugging and monitoring
- ✅ **Team Development**: Shared development environment
- ✅ **Production-like Testing**: Kubernetes-based testing

### **Use Docker Compose For:**
- ✅ **Simple Deployment**: Quick one-off testing
- ✅ **CI/CD Pipelines**: Automated testing without K8s
- ✅ **Resource-Constrained**: Lower resource environments
- ✅ **Quick Prototyping**: Simple service interactions

---

## 🎵 Ready to Rock Your Amateur Radio Development! 📻

**Tilt transforms USRP Audio Router Hub development from tedious container management into a delightful, productive experience. Make a code change, see it instantly reflected across all your services, debug issues in real-time, and test complex audio routing scenarios with ease!**

```bash
# Start your amazing development experience
make tilt-up

# Open the dashboard and enjoy! 
make tilt-dashboard
```

**Happy Amateur Radio Hacking! 73! 📻⚡**