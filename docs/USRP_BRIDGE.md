# USRP Bridge Utility

The USRP Bridge is a powerful utility that connects AllStarLink nodes with modern destination services through FFmpeg Opus conversion. It acts as a transparent bridge, converting amateur radio USRP packets to compressed audio formats for internet transmission.

## Architecture

```
AllStarLink Node <--USRP--> USRP Bridge <--Opus/Ogg--> Destination Service
                             │                          │
                             ├── WhoTalkie              │
                             ├── Discord Bot            │
                             └── Custom Services        │
```

The bridge receives USRP packets from AllStarLink nodes, converts the audio to Opus/Ogg format using FFmpeg, and forwards the compressed audio to configured destination services while maintaining full amateur radio protocol compliance.

## Key Features

✅ **Multi-Destination Support**: Forward audio to multiple services simultaneously  
✅ **FFmpeg Integration**: High-quality Opus/Ogg audio conversion  
✅ **AllStarLink Compatible**: Full USRP protocol compliance  
✅ **Configurable Routing**: JSON-based configuration for complex setups  
✅ **Real-time Monitoring**: Built-in statistics and performance metrics  
✅ **Amateur Radio Compliant**: Proper callsign handling and PTT control  
✅ **Production Ready**: Robust error handling and graceful degradation  

## Installation

### Requirements

- **Go 1.19+** for building the bridge
- **FFmpeg with libopus** for audio conversion
- **AllStarLink node** or USRP-compatible system
- **Amateur radio license** for legal operation

### Building

```bash
# Build the USRP bridge binary
make build-usrp-bridge

# Or build manually
go build -o bin/usrp-bridge cmd/usrp-bridge/main.go
```

### FFmpeg Installation

```bash
# macOS (Homebrew)
brew install ffmpeg

# Ubuntu/Debian  
sudo apt install ffmpeg libopus-dev

# CentOS/RHEL
sudo yum install ffmpeg opus-devel

# Verify Opus support
ffmpeg -encoders | grep opus
```

## Quick Start

### Simple Usage (Command Line)

```bash
# Run with default settings (listen on :12345, forward to 127.0.0.1:8080)
./bin/usrp-bridge

# Specify custom ports and destination
./bin/usrp-bridge -listen-port 12345 -dest-host 192.168.1.100 -dest-port 8080 -callsign W1AW

# Enable verbose logging
./bin/usrp-bridge -verbose -callsign KC1ABC
```

### Configuration File Usage

```bash
# Generate sample configuration
make run-usrp-bridge-config
# or: ./bin/usrp-bridge -generate-config

# Edit the generated usrp-bridge.json file
vim usrp-bridge.json

# Run with configuration file
./bin/usrp-bridge -config usrp-bridge.json
```

## Configuration

### Configuration File Format

```json
{
  "usrp_listen_port": 12345,
  "usrp_listen_addr": "0.0.0.0",
  "allstar_host": "127.0.0.1",
  "allstar_port": 12346,
  "destinations": [
    {
      "name": "whotalkie",
      "type": "whotalkie",
      "host": "127.0.0.1",
      "port": 8080,
      "protocol": "udp",
      "format": "opus",
      "enabled": true
    },
    {
      "name": "discord-bot",
      "type": "discord",
      "host": "127.0.0.1", 
      "port": 8081,
      "protocol": "udp",
      "format": "opus",
      "enabled": false,
      "settings": {
        "guild_id": "your_guild_id",
        "channel_id": "your_channel_id"
      }
    }
  ],
  "audio_config": {
    "enable_conversion": true,
    "output_format": "opus",
    "bitrate": 64,
    "sample_rate": 8000,
    "channels": 1
  },
  "log_level": "info",
  "metrics_port": 9090,
  "station_call": "N0CALL",
  "talk_group": 0
}
```

### Configuration Parameters

#### Network Settings
- **`usrp_listen_port`**: Port to listen for USRP packets (default: 12345)
- **`usrp_listen_addr`**: Listen address (default: "0.0.0.0")
- **`allstar_host`**: AllStarLink return address (default: "127.0.0.1")
- **`allstar_port`**: AllStarLink return port (default: 12346)

#### Destination Services
- **`name`**: Unique identifier for the destination
- **`type`**: Service type ("whotalkie", "discord", "generic")
- **`host`**: Destination host address
- **`port`**: Destination port number
- **`protocol`**: Transport protocol ("udp", "tcp", "websocket")
- **`format`**: Audio format ("opus", "ogg", "raw")
- **`enabled`**: Enable/disable this destination
- **`settings`**: Service-specific configuration parameters

#### Audio Processing
- **`enable_conversion`**: Enable FFmpeg audio conversion
- **`output_format`**: Target audio format ("opus", "ogg")
- **`bitrate`**: Audio bitrate in kbps (default: 64)
- **`sample_rate`**: Output sample rate in Hz (default: 8000)
- **`channels`**: Number of audio channels (default: 1)

#### Amateur Radio Settings
- **`station_call`**: Your amateur radio callsign
- **`talk_group`**: USRP talk group ID (default: 0)

## Usage Examples

### WhoTalkie Integration

Connect AllStarLink to WhoTalkie service:

```json
{
  "usrp_listen_port": 12345,
  "allstar_host": "your-allstar-node.local",
  "allstar_port": 2001,
  "destinations": [
    {
      "name": "whotalkie-main",
      "type": "whotalkie",
      "host": "whotalkie.example.com",
      "port": 8080,
      "protocol": "udp", 
      "format": "opus",
      "enabled": true,
      "settings": {
        "room": "main-room",
        "user": "AllStarLink-Bridge"
      }
    }
  ],
  "station_call": "W1AW",
  "audio_config": {
    "enable_conversion": true,
    "output_format": "opus",
    "bitrate": 32
  }
}
```

### Multi-Destination Setup

Forward audio to multiple services:

```json
{
  "destinations": [
    {
      "name": "whotalkie-primary",
      "type": "whotalkie",
      "host": "primary.whotalkie.com",
      "port": 8080,
      "enabled": true
    },
    {
      "name": "whotalkie-backup",
      "type": "whotalkie", 
      "host": "backup.whotalkie.com",
      "port": 8080,
      "enabled": true
    },
    {
      "name": "discord-emergency",
      "type": "discord",
      "host": "127.0.0.1",
      "port": 8081,
      "enabled": false
    }
  ]
}
```

### High-Quality Audio Setup

For high-quality audio applications:

```json
{
  "audio_config": {
    "enable_conversion": true,
    "output_format": "ogg",
    "bitrate": 128,
    "sample_rate": 8000,
    "channels": 1
  }
}
```

## AllStarLink Integration

### Node Configuration

Configure your AllStarLink node to send USRP packets to the bridge:

```ini
; In your rpt.conf
[node-number]
duplex = 0
rxchannel = usrp/127.0.0.1:12345,usrp

; Enable USRP transmit
[usrp]
; Configuration for your USRP bridge
call = W1AW
context = radio-bridge
```

### Network Setup

```bash
# AllStarLink node sends to bridge
# Configure your node to send USRP packets to bridge IP:12345

# Bridge receives USRP packets and converts to Opus
# Bridge forwards Opus to destination services

# Example network flow:
# AllStarLink Node (192.168.1.100:2001) -> Bridge (192.168.1.200:12345) -> WhoTalkie (whotalkie.com:8080)
```

## Monitoring and Statistics

### Real-time Statistics

```bash
# Send SIGUSR1 to display statistics
kill -USR1 $(pgrep usrp-bridge)

# Or if running in foreground, press Ctrl+\ on some systems
```

### Statistics Output

```
📊 Bridge Statistics
==================
USRP Packets: 1234 received, 1230 sent
Opus Packets: 890 generated, 888 forwarded
Errors: 2 conversion, 1 network
Traffic: 445632 bytes received, 89126 bytes sent
Last Activity: 2025-01-15T10:30:45Z
```

### Performance Monitoring

The bridge provides metrics on port 9090 (configurable):

```bash
# Check bridge health
curl http://localhost:9090/stats

# Monitor with tools like Prometheus/Grafana
```

## Service Integration

### WhoTalkie Protocol

The bridge formats audio data for WhoTalkie compatibility:

- Opus audio at configured bitrate
- UDP transport protocol
- Metadata preservation for PTT state and callsign

### Discord Integration

For Discord bot integration:

```json
{
  "name": "discord-bot",
  "type": "discord",
  "settings": {
    "guild_id": "123456789012345678",
    "channel_id": "987654321098765432",
    "bot_token": "stored_separately"
  }
}
```

### Generic Services

For custom services:

```json
{
  "name": "custom-service",
  "type": "generic",
  "protocol": "udp",
  "format": "opus",
  "settings": {
    "custom_header": true,
    "packet_size": 1024
  }
}
```

## Troubleshooting

### Common Issues

**"Failed to create audio converter"**
```bash
# Check FFmpeg installation
ffmpeg -version
ffmpeg -encoders | grep opus

# Install FFmpeg with Opus support
brew install ffmpeg  # macOS
sudo apt install ffmpeg libopus-dev  # Ubuntu
```

**"Failed to listen on USRP port"**
```bash
# Check if port is already in use
sudo netstat -ulnp | grep :12345

# Check permissions for low ports (<1024)
sudo ./usrp-bridge -listen-port 1234
```

**"Failed to connect to destination"**
```bash
# Test network connectivity
nc -u destination-host 8080

# Check firewall settings
# Ensure destination service is running and accessible
```

### Performance Issues

**High CPU Usage**
- Reduce audio bitrate (64 -> 32 kbps)
- Disable unnecessary destinations
- Check FFmpeg processes with `ps aux | grep ffmpeg`

**High Latency**
- Minimize network hops to destination services
- Use faster network connections
- Consider local destination services

**Packet Loss**
- Check network quality with `ping` and `traceroute`
- Increase buffer sizes if available
- Monitor with `iftop` or similar tools

## Security Considerations

### Network Security

- **Firewall Configuration**: Only allow necessary ports
- **Access Control**: Limit source IPs for USRP packets
- **Encryption**: Use VPN for internet transmission when possible

### Amateur Radio Compliance

- **Station Identification**: Ensure proper callsign in configuration
- **License Verification**: Valid amateur radio license required
- **Band Plan Compliance**: Use only authorized frequencies
- **Third Party Traffic**: Follow local amateur radio regulations

## Development and Customization

### Custom Destination Types

Add support for new services by modifying the `formatFor*` functions:

```go
func (b *Bridge) formatForCustomService(audioData []byte, voiceMsg *usrp.VoiceMessage) []byte {
    // Custom formatting logic
    return formattedData
}
```

### Extending Configuration

Add new configuration parameters:

```go
type DestinationConfig struct {
    // Existing fields...
    CustomField string `json:"custom_field,omitempty"`
}
```

### Plugin Architecture

The bridge can be extended with a plugin system for custom audio processing or destination protocols.

## Production Deployment

### Systemd Service

```ini
[Unit]
Description=USRP Bridge Utility
After=network.target
Wants=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/usrp-bridge -config /etc/usrp-bridge/config.json
Restart=always
RestartSec=5
User=usrp-bridge
Group=usrp-bridge

[Install]
WantedBy=multi-user.target
```

### Docker Deployment

```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o usrp-bridge cmd/usrp-bridge/main.go

FROM alpine:latest
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/usrp-bridge /usr/local/bin/
EXPOSE 12345/udp
CMD ["usrp-bridge", "-config", "/etc/usrp-bridge/config.json"]
```

### High Availability

For critical applications:

- Run multiple bridge instances with different destination priorities
- Use load balancers for incoming USRP traffic
- Implement health checks and automatic failover
- Monitor with tools like Nagios or Zabbix

## Legal and Regulatory

### Amateur Radio Compliance

⚠️ **Important Legal Requirements**

- **Valid License**: Amateur radio license required for USRP operation
- **Proper Identification**: Station callsign must be configured correctly
- **Band Plan Compliance**: Use only authorized amateur frequencies
- **Third Party Traffic**: Follow regulations for internet-linked communications
- **Power Limits**: Ensure compliance with power restrictions
- **Spurious Emissions**: Use proper filtering and clean signals

### International Considerations

- Different countries have varying amateur radio regulations
- Internet linking may have specific requirements
- Check with local amateur radio authorities
- Some regions restrict third-party traffic or encryption

## Contributing

Contributions are welcome! Please:

1. Maintain amateur radio compliance in all changes
2. Test with real AllStarLink systems when possible
3. Follow Go best practices and include tests
4. Document any configuration changes
5. Consider backward compatibility

## License

MIT License - See LICENSE file for details.

This software is provided for **amateur radio experimentation** under Part 97 regulations and similar international amateur radio regulations.

---

**73, Good DX!** 📻🌐

*"Bridging amateur radio to the modern internet while preserving the spirit of experimentation"*

## References

- [AllStarLink Documentation](https://wiki.allstarlink.org/)
- [WhoTalkie Project](https://github.com/dbehnke/whotalkie)
- [FFmpeg Documentation](https://ffmpeg.org/documentation.html)
- [USRP Protocol Specification](https://github.com/dbehnke/usrp-go)