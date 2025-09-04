# 📋 USRP Audio Router Hub - System Requirements

Complete system requirements and setup instructions for developing and running the USRP Audio Router Hub platform.

## 🖥️ Operating System Support

| OS | Status | Notes |
|----|--------|-------|
| **macOS** | ✅ Recommended | Best development experience with Colima |
| **Linux** | ✅ Fully Supported | Native Docker support |
| **Windows** | ✅ Supported | WSL2 recommended for best experience |

---

## 🍎 macOS Setup (Recommended: Colima)

### **Why Colima over Docker Desktop?**
- **🆓 Free and Open Source**: No licensing restrictions
- **⚡ Lightweight**: Lower resource usage than Docker Desktop
- **🔧 Developer Friendly**: Better integration with development tools
- **🚀 Fast**: Optimized performance for macOS
- **💾 Disk Efficient**: More efficient disk usage

### **macOS Prerequisites**
```bash
# Install Homebrew (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required tools
brew install git go colima docker kubectl tilt just
```

### **Colima Setup for USRP Development**
```bash
# Start Colima with optimized settings for our audio platform
colima start --cpu 4 --memory 8 --disk 60 --kubernetes

# Verify installation
docker version
kubectl cluster-info
colima status
```

**Recommended Colima Configuration:**
```bash
# ~/.colima/default/colima.yaml (auto-created after first start)
# Optimized for USRP Audio Router Hub development

cpu: 4                    # 4 CPU cores for multi-service testing
memory: 8                 # 8GB RAM for all services + monitoring
disk: 60                  # 60GB disk for images and data

# Kubernetes enabled for Tilt development
kubernetes:
  enabled: true
  version: v1.28.4

# Port forwarding for development
network:
  address: true           # Enable VM IP access

# Volume mounts for development
mounts:
  - location: ~/development
    writable: true
```

### **Starting/Stopping Colima**
```bash
# Start Colima (do this before development)
colima start

# Stop Colima (when done for the day)
colima stop

# Restart Colima (if issues arise)
colima restart

# Check status
colima status
```

### **macOS Development Workflow**
```bash
# 1. Start Colima
colima start

# 2. Start USRP development environment
cd /path/to/usrp-go
make tilt-up

# 3. Open Tilt dashboard
make tilt-dashboard

# 4. Develop with live reload!
# Edit code → Tilt rebuilds → Services restart automatically

# 5. When done, stop services and Colima
make tilt-down
colima stop
```

---

## 🐧 Linux Setup

### **Ubuntu/Debian**
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install kind (for local Kubernetes)
go install sigs.k8s.io/kind@latest

# Install Tilt
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

# Start kind cluster
kind create cluster --name usrp-dev
```

### **Red Hat/CentOS/Fedora**
```bash
# Install Docker
sudo dnf install -y docker
sudo systemctl enable --now docker
sudo usermod -aG docker $USER

# Install kubectl, kind, and tilt (same as Ubuntu)
# Follow Ubuntu instructions above
```

---

## 🪟 Windows Setup

### **Recommended: WSL2 + Docker**
```powershell
# Install WSL2
wsl --install

# Inside WSL2 Ubuntu, follow Linux setup instructions
# Then access from Windows with excellent performance
```

### **Alternative: Docker Desktop**
```powershell
# Install Docker Desktop for Windows
# Enable Kubernetes in Docker Desktop settings
# Install Tilt for Windows
```

---

## 🔧 Development Tools

### **Required Tools**
| Tool | Version | Purpose |
|------|---------|---------|
| **Go** | 1.25+ | Core development language |
| **Docker** | 24.0+ | Container runtime |
| **kubectl** | 1.28+ | Kubernetes management |
| **Tilt** | 0.33+ | Development environment orchestration |
| **just** | 1.15+ | Modern command runner (replaces make) |

### **Optional but Recommended**
| Tool | Purpose |
|------|---------|
| **FFmpeg** | Audio format conversion testing |
| **tcpdump** | Network packet analysis |
| **netcat** | Network connectivity testing |
| **jq** | JSON processing for API responses |
| **curl** | HTTP API testing |

### **Installation Verification**
```bash
# Verify all tools are properly installed
go version          # Should show Go 1.25+
docker version      # Should show Docker client/server
kubectl version     # Should show client/server versions  
tilt version        # Should show Tilt version
```

---

## 📡 Amateur Radio Specific Requirements

### **Audio Development**
For audio format conversion and analysis:
```bash
# macOS
brew install ffmpeg sox

# Ubuntu/Debian  
sudo apt install ffmpeg sox

# Red Hat/CentOS/Fedora
sudo dnf install ffmpeg sox
```

### **Network Testing Tools**
For amateur radio protocol testing:
```bash
# macOS
brew install tcpdump netcat wireshark

# Linux
sudo apt install tcpdump netcat wireshark  # Ubuntu/Debian
sudo dnf install tcpdump netcat wireshark  # Red Hat/CentOS/Fedora
```

### **Protocol Analysis**
For USRP packet inspection:
```bash
# Wireshark with custom dissectors (optional)
# Useful for detailed USRP packet analysis
```

---

## 🚀 Quick Start Verification

### **System Health Check**
```bash
# Clone the repository
git clone https://github.com/dbehnke/usrp-go.git
cd usrp-go

# Verify Go build works
go build ./cmd/audio-router/

# Verify Docker works
docker run hello-world

# Verify Kubernetes works (Colima/kind/Docker Desktop)
kubectl cluster-info

# Verify Tilt works
tilt version
```

### **Run Basic Tests**
```bash
# Run unit tests
just test

# Test basic functionality
just run-example

# Test audio router config generation
just router-config

# Start development environment (requires Kubernetes)
just dev
```

---

## ⚡ Performance Recommendations

### **macOS with Colima**
```bash
# Optimal Colima settings for USRP development
colima start \
  --cpu 4 \
  --memory 8 \
  --disk 60 \
  --kubernetes \
  --network-address
```

### **Resource Requirements by Use Case**

#### **Minimum (Basic Development)**
- **CPU**: 2 cores
- **RAM**: 4GB
- **Disk**: 20GB
- **Services**: Audio Router + 1 mock service

#### **Recommended (Full Development)**  
- **CPU**: 4 cores
- **RAM**: 8GB
- **Disk**: 60GB
- **Services**: Audio Router + All mock services + Monitoring

#### **Optimal (Heavy Development)**
- **CPU**: 8 cores
- **RAM**: 16GB
- **Disk**: 100GB
- **Services**: Multiple router instances + Load testing

---

## 🔍 Troubleshooting

### **Common macOS Issues**

#### **Colima Won't Start**
```bash
# Check if Docker Desktop is running (conflicts with Colima)
docker context ls
# Should show "colima" as current context

# If Docker Desktop is active:
docker context use colima

# Reset Colima if needed
colima delete
colima start --cpu 4 --memory 8 --kubernetes
```

#### **Kubernetes Not Working**
```bash
# Verify Kubernetes is enabled in Colima
colima status

# If not enabled:
colima stop
colima start --kubernetes

# Verify kubectl context
kubectl config current-context  # Should show "colima"
```

#### **Port Forwarding Issues**
```bash
# Check if ports are already in use
lsof -i :9090
lsof -i :10350

# Kill conflicting processes if needed
sudo lsof -ti :9090 | xargs kill -9
```

### **Common Linux Issues**

#### **Docker Permission Denied**
```bash
# Ensure user is in docker group
sudo usermod -aG docker $USER
# Log out and log back in

# Or use sudo temporarily
sudo docker version
```

#### **Kind Cluster Issues**
```bash
# Delete and recreate kind cluster
kind delete cluster --name usrp-dev
kind create cluster --name usrp-dev

# Verify cluster is running
kubectl cluster-info --context kind-usrp-dev
```

### **Getting Help**

#### **Check System Status**
```bash
# Comprehensive system check
./scripts/system-check.sh  # (if created)

# Or manual checks:
colima status              # macOS
docker version
kubectl cluster-info  
tilt doctor               # Diagnose Tilt issues
```

#### **Useful Debug Commands**
```bash
# Docker debugging
docker system info
docker system df

# Kubernetes debugging
kubectl get nodes
kubectl get pods --all-namespaces

# Tilt debugging  
tilt doctor
tilt logs --follow
```

---

## 📚 Additional Resources

### **Official Documentation**
- **[Colima GitHub](https://github.com/abiosoft/colima)**: Official Colima documentation
- **[Tilt Documentation](https://docs.tilt.dev/)**: Complete Tilt development guide
- **[kubectl Reference](https://kubernetes.io/docs/reference/kubectl/)**: Kubernetes command reference

### **Amateur Radio Resources**
- **[AllStarLink Wiki](https://wiki.allstarlink.org/)**: AllStarLink documentation
- **[USRP Protocol Spec](https://github.com/dl1hrc/svxlink/blob/svxlink-usrp/src/svxlink/svxlink/contrib/UsrpLogic/usrp_protocol.txt)**: Official USRP protocol documentation

### **Development Resources**
- **[Go Documentation](https://golang.org/doc/)**: Official Go language documentation  
- **[Docker Best Practices](https://docs.docker.com/develop/best-practices/)**: Container development guidelines

---

## 🎯 Platform-Specific Quick Start

### **macOS with Colima (Recommended)**
```bash
# One-time setup
brew install colima docker kubectl tilt
colima start --cpu 4 --memory 8 --kubernetes

# Daily development workflow
cd usrp-go
just dev               # Start development environment (Tilt)
just tilt-dashboard    # Open Tilt UI
# ... develop with live reload ...
just tilt-down         # Stop when done
colima stop           # Stop Colima to save resources
```

### **Linux with Docker + kind**
```bash
# One-time setup  
# [Install Docker, kubectl, kind, tilt as shown above]
kind create cluster --name usrp-dev

# Daily development workflow
cd usrp-go
just dev               # Start development environment  
just tilt-dashboard    # Open Tilt UI
# ... develop with live reload ...
just tilt-down         # Stop when done
```

### **Windows with WSL2**
```bash
# Run in WSL2 Ubuntu environment
# Follow Linux setup instructions
# Access from Windows browser: http://localhost:10350
```

---

**🎉 With these requirements met, you'll have an excellent development experience for the USRP Audio Router Hub platform! The combination of Colima (on macOS) + Tilt provides a lightweight, fast, and professional development environment perfect for amateur radio software development.**