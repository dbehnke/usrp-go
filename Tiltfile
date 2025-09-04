# Tiltfile for USRP Audio Router Hub Development Environment
# Provides live development environment with automatic reloading and service monitoring

# Load Tilt extensions
load('ext://restart_process', 'docker_build_with_restart')
load('ext://helm_resource', 'helm_resource', 'helm_repo')

# Configuration
config.define_bool("enable_audio_generator", default=True)
config.define_bool("enable_monitoring", default=True) 
config.define_bool("enable_debug_logging", default=False)
config.define_string("audio_pattern", default="sine_440hz")
config.define_string("test_duration", default="300") # 5 minutes

cfg = config.parse()

print("""
🎵 USRP Audio Router Hub - Tilt Development Environment
======================================================
✅ Live service orchestration with automatic reloading
✅ Real-time monitoring and debugging
✅ Interactive audio testing environment  
✅ Amateur radio protocol compliance testing

Configuration:
  Audio Generator: {}
  Monitoring: {}
  Debug Logging: {}
  Test Pattern: {}
  Test Duration: {}s

Access Points:
  🎛️  Audio Router Hub: http://localhost:9090
  📊 Tilt Dashboard: http://localhost:10350
  📈 Grafana: http://localhost:3000 (admin/admin)
  🔍 Prometheus: http://localhost:9091
""".format(
    "Enabled" if cfg.get("enable_audio_generator") else "Disabled",
    "Enabled" if cfg.get("enable_monitoring") else "Disabled", 
    "Enabled" if cfg.get("enable_debug_logging") else "Disabled",
    cfg.get("audio_pattern"),
    cfg.get("test_duration")
))

# Build configurations for all services
def build_audio_router():
    """Build the Audio Router Hub with live reload"""
    docker_build(
        'audio-router',
        context='.',
        dockerfile='./test/tilt/dockerfiles/Dockerfile.audio-router',
        # Live update: rebuild only on Go file changes
        live_update=[
            sync('./cmd/audio-router/', '/app/cmd/audio-router/'),
            sync('./pkg/', '/app/pkg/'),
            run('go build -o /app/audio-router /app/cmd/audio-router/main.go', trigger=[
                './cmd/audio-router/main.go',
                './pkg/usrp/',
                './pkg/audio/', 
                './pkg/discord/'
            ]),
            restart_container()
        ],
        target='dev'  # Multi-stage build target for development
    )

def build_allstar_mock():
    """Build AllStarLink mock server with live reload"""
    docker_build(
        'allstar-mock',
        context='./test/containers/allstar-mock',
        dockerfile='./test/tilt/dockerfiles/Dockerfile.allstar-mock',
        live_update=[
            sync('./test/containers/allstar-mock/', '/app/'),
            run('go build -o /app/allstar-mock /app/allstar-mock.go', trigger=[
                './test/containers/allstar-mock/allstar-mock.go'
            ]),
            restart_container()
        ]
    )

def build_whotalkie_mock():
    """Build WhoTalkie mock service"""
    docker_build(
        'whotalkie-mock',
        context='./test/tilt/services/whotalkie-mock',
        dockerfile='./test/tilt/dockerfiles/Dockerfile.whotalkie-mock'
    )

def build_discord_mock():
    """Build Discord voice gateway mock"""
    docker_build(
        'discord-mock', 
        context='./test/tilt/services/discord-mock',
        dockerfile='./test/tilt/dockerfiles/Dockerfile.discord-mock'
    )

def build_audio_generator():
    """Build audio test signal generator"""
    docker_build(
        'audio-generator',
        context='./test/tilt/services/audio-generator', 
        dockerfile='./test/tilt/dockerfiles/Dockerfile.audio-generator'
    )

# Build all container images
print("🔨 Building container images...")
build_audio_router()
build_allstar_mock() 
build_whotalkie_mock()
build_discord_mock()

if cfg.get("enable_audio_generator"):
    build_audio_generator()

# Deploy Kubernetes resources
print("🚀 Deploying Kubernetes resources...")

# Core Audio Router Hub
k8s_yaml('./test/tilt/k8s/audio-router.yaml')
k8s_resource('audio-router',
    port_forwards=['9090:9090', '32001:32001', '32002:32002', '32003:32003', '32004:32004'],
    resource_deps=['allstar-mock-1'],  # Ensure mocks start first
    labels=['core']
)

# AllStarLink Mock Servers  
k8s_yaml('./test/tilt/k8s/allstar-mocks.yaml')
k8s_resource('allstar-mock-1',
    port_forwards='34001:34001',
    labels=['mocks', 'usrp']
)
k8s_resource('allstar-mock-2', 
    port_forwards='34002:34002',
    labels=['mocks', 'usrp']
)

# WhoTalkie Mock Services
k8s_yaml('./test/tilt/k8s/whotalkie-mocks.yaml') 
k8s_resource('whotalkie-mock-1',
    port_forwards='8080:8080',
    labels=['mocks', 'whotalkie']
)
k8s_resource('whotalkie-mock-2',
    port_forwards='8081:8081', 
    labels=['mocks', 'whotalkie']
)

# Discord Mock Service
k8s_yaml('./test/tilt/k8s/discord-mock.yaml')
k8s_resource('discord-mock',
    port_forwards='8082:8082',
    labels=['mocks', 'discord'] 
)

# Audio Test Generator (optional)
if cfg.get("enable_audio_generator"):
    k8s_yaml('./test/tilt/k8s/audio-generator.yaml')
    k8s_resource('audio-generator',
        labels=['testing', 'audio']
    )

# Monitoring Stack (optional)
if cfg.get("enable_monitoring"):
    print("📊 Setting up monitoring stack...")
    
    # Prometheus
    k8s_yaml('./test/tilt/k8s/prometheus.yaml')
    k8s_resource('prometheus',
        port_forwards='9091:9090',
        labels=['monitoring']
    )
    
    # Grafana  
    k8s_yaml('./test/tilt/k8s/grafana.yaml')
    k8s_resource('grafana',
        port_forwards='3000:3000',
        labels=['monitoring']
    )

# Development helpers and local resources
print("🛠️  Setting up development helpers...")

# Integration test runner
local_resource(
    'integration-tests',
    cmd='./test/tilt/scripts/run-tests.sh',
    deps=['./test/tilt/scripts/', './test/containers/'],
    labels=['testing'],
    allow_parallel=True
)

# Audio quality validator  
local_resource(
    'audio-quality-check',
    cmd='./test/tilt/scripts/validate-audio-quality.sh',
    deps=['./test/tilt/scripts/'],
    labels=['testing', 'audio'],
    auto_init=False,  # Run manually
    trigger_mode=TRIGGER_MODE_MANUAL
)

# Log aggregation and analysis
local_resource(
    'log-analysis',
    cmd='./test/tilt/scripts/analyze-logs.sh', 
    deps=['./test/tilt/scripts/'],
    labels=['debugging'],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL
)

# Network traffic analysis
local_resource(
    'network-analysis',
    cmd='./test/tilt/scripts/capture-traffic.sh',
    deps=['./test/tilt/scripts/'],
    labels=['debugging', 'network'],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL
)

# Performance profiling
local_resource(
    'performance-profile',
    cmd='./test/tilt/scripts/profile-performance.sh',
    deps=['./test/tilt/scripts/'],
    labels=['performance'],
    auto_init=False,
    trigger_mode=TRIGGER_MODE_MANUAL
)

# Configuration validator
local_resource(
    'config-validator',
    cmd='go run ./cmd/audio-router/main.go -config ./test/tilt/configs/audio-router-dev.json -validate-only',
    deps=['./cmd/audio-router/', './test/tilt/configs/'],
    labels=['validation']
)

# Development workflow helpers
print("⚡ Setting up development workflow...")

# Hot reload for Go code changes
watch_file('./cmd/audio-router/main.go')
watch_file('./pkg/usrp/')
watch_file('./pkg/audio/')
watch_file('./pkg/discord/')

# Configuration file watching
watch_file('./test/tilt/configs/audio-router-dev.json')

# Mock service watching
watch_file('./test/containers/allstar-mock/allstar-mock.go')

# Startup sequence and health checks
print("🏥 Configuring health checks...")

# Custom health checks for audio services
def audio_router_health():
    return local('curl -f http://localhost:9090/status', quiet=True)

def allstar_mock_health():
    return local('nc -z localhost 34001', quiet=True)

# Service groups for organized dashboard
print("📋 Organizing service groups...")

# Group services logically in Tilt UI
k8s_resource('audio-router', labels=['core', 'audio-routing'])
k8s_resource('allstar-mock-1', labels=['mocks', 'amateur-radio'])  
k8s_resource('allstar-mock-2', labels=['mocks', 'amateur-radio'])
k8s_resource('whotalkie-mock-1', labels=['mocks', 'internet-services'])
k8s_resource('whotalkie-mock-2', labels=['mocks', 'internet-services']) 
k8s_resource('discord-mock', labels=['mocks', 'discord'])

# Print helpful information
print("""
🎉 USRP Audio Router Hub Development Environment Ready!

Quick Start:
1. Check Tilt dashboard: http://localhost:10350
2. View Audio Router status: http://localhost:9090/status  
3. Monitor logs in real-time via Tilt UI
4. Make code changes - services will auto-reload!

Manual Test Commands:
  tilt trigger integration-tests      # Run integration tests
  tilt trigger audio-quality-check    # Validate audio quality
  tilt trigger network-analysis       # Capture network traffic
  tilt trigger performance-profile    # Profile performance

Service Endpoints:
  🎛️  Audio Router Hub:     http://localhost:9090
  📻 AllStar Mock 1:       udp://localhost:34001
  📻 AllStar Mock 2:       udp://localhost:34002  
  🌐 WhoTalkie Mock 1:     udp://localhost:8080
  🌐 WhoTalkie Mock 2:     udp://localhost:8081
  🎮 Discord Mock:         tcp://localhost:8082

Development Tips:
- Edit Go files → automatic rebuild and restart
- View all logs in Tilt dashboard
- Use 'tilt down' to stop all services  
- Use 'tilt up --stream' for detailed output

Happy Amateur Radio Development! 📻 73!
""")

# Advanced features for power users
if config.main_dir.endswith('development') or config.main_dir.endswith('dev'):
    print("🔧 Development mode detected - enabling advanced features...")
    
    # Enable debug logging
    local_resource(
        'enable-debug-logging',
        cmd='kubectl patch deployment audio-router -p \\'{"spec":{"template":{"spec":{"containers":[{"name":"audio-router","env":[{"name":"DEBUG","value":"true"}]}]}}}}\\' || true',
        labels=['dev-tools']
    )
    
    # Enable Go race detection in development
    local_resource(
        'enable-race-detection', 
        cmd='echo "Race detection enabled in development builds"',
        labels=['dev-tools']
    )