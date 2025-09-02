# Discord Bridge for Amateur Radio

This document explains how to use the Discord bridge to connect amateur radio USRP systems with Discord voice channels, enabling real-time voice communication between amateur radio operators and Discord users.

## Overview

The Discord bridge provides **bidirectional audio conversion** between:
- **Amateur Radio**: USRP protocol packets (8kHz mono PCM)
- **Discord Voice**: Discord voice channels (48kHz stereo Opus)

This enables amateur radio repeaters and networks to extend their reach through Discord voice channels while maintaining proper amateur radio protocols and identification.

## Key Features

✅ **Real-time Voice Bridge**: Live audio between amateur radio and Discord  
✅ **Automatic Audio Conversion**: 8kHz ↔ 48kHz resampling with format conversion  
✅ **PTT Integration**: Push-to-talk control from amateur radio to Discord  
✅ **Voice Activity Detection**: Automatic transmission triggering from Discord  
✅ **Amateur Radio Compliant**: Proper USRP packet formatting and callsign handling  
✅ **High Performance**: Low-latency audio processing with efficient buffering  

## Requirements

### Software Requirements
- **Go 1.19+** for building the bridge
- **FFmpeg with libopus** for audio conversion
- **Discord Bot** with voice channel permissions

### Amateur Radio Requirements
- **Amateur radio license** for USRP operation
- **USRP-compatible system** (AllStarLink, app_rpt, etc.)
- **Valid amateur radio callsign** for identification

### Discord Requirements
- **Discord Server** with voice channels
- **Bot Token** with voice permissions
- **Voice Channel Access** for the bot

## Setup

### 1. Create Discord Bot

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Go to "Bot" section and create a bot
4. Copy the bot token (keep it secret!)
5. Enable the following permissions:
   - `Connect` - Join voice channels
   - `Speak` - Send audio to voice channels
   - `Use Voice Activity` - Voice activity detection

### 2. Install Dependencies

```bash
# Install FFmpeg with Opus support
# macOS
brew install ffmpeg

# Ubuntu/Debian
sudo apt install ffmpeg libopus-dev

# Verify Opus support
ffmpeg -encoders | grep opus
```

### 3. Environment Setup

```bash
# Required: Discord bot token
export DISCORD_TOKEN="your_bot_token_here"

# Optional: Specific Discord server and channel
export DISCORD_GUILD="your_guild_id"
export DISCORD_CHANNEL="your_voice_channel_id"

# Required for amateur radio: Your callsign
export AMATEUR_CALLSIGN="N0CALL"  # Replace with your callsign
```

## Usage

### Basic Bridge Operation

```bash
# Test Discord bot connection
make run-discord-test

# Run the full bridge
make run-discord-bridge
```

### Manual Commands

```bash
# Test Discord bot (verify token and permissions)
go run cmd/discord-bridge/main.go test

# Run USRP-Discord bridge
go run cmd/discord-bridge/main.go bridge

# Run USRP test server (for testing without real amateur radio)
go run cmd/discord-bridge/main.go server
```

## Architecture

```
Amateur Radio ←→ USRP Packets ←→ Audio Bridge ←→ Discord Bot ←→ Discord Voice
```

### Data Flow

**Amateur Radio → Discord:**
1. Amateur radio system sends USRP voice packets (8kHz mono PCM)
2. Bridge receives packets via UDP
3. Audio converted from 8kHz mono to 48kHz stereo
4. Discord bot sends audio to voice channel

**Discord → Amateur Radio:**
1. Discord bot receives voice from users
2. Voice activity detection triggers transmission
3. Audio resampled from 48kHz stereo to 8kHz mono
4. Bridge creates USRP packets with proper amateur radio formatting
5. Packets sent to amateur radio system via UDP

## Configuration

### Bridge Configuration

```go
config := discord.DefaultBridgeConfig()
config.DiscordToken = "your_token"
config.DiscordGuild = "guild_id"
config.DiscordChannel = "channel_id"
config.CallSign = "N0CALL"
config.VoiceThreshold = 1000    // Voice activation threshold
config.PTTTimeout = 2 * time.Second
```

### Audio Settings

| Parameter | USRP/Amateur Radio | Discord |
|-----------|-------------------|---------|
| Sample Rate | 8000 Hz | 48000 Hz |
| Channels | 1 (mono) | 2 (stereo) |
| Format | 16-bit PCM | Opus compressed |
| Frame Size | 20ms (160 samples) | 20ms (960 samples) |

## Network Configuration

### Default Ports

- **USRP Input**: UDP port 12345 (configurable)
- **Amateur Radio Output**: UDP (configured in your amateur radio system)

### Firewall Considerations

Ensure these ports are accessible:
- UDP ports for USRP communication
- HTTPS (443) for Discord API
- WebSocket connections for Discord voice

## Discord Bot Commands

Once the bot is running in a Discord server, users can interact with it:

```
!join      - Bot joins the user's current voice channel
!leave     - Bot leaves the voice channel  
!status    - Shows connection status
```

## Amateur Radio Integration

### AllStarLink Integration

```bash
# In your AllStarLink node configuration:
# Send USRP packets to bridge
usrp_node=1999,127.0.0.1:12345,NONE

# Receive USRP packets from bridge  
# Configure your node to listen on the bridge's output port
```

### Manual USRP Packet Testing

```bash
# Generate test USRP packets
make run-discord-server

# This creates realistic amateur radio voice patterns for testing
```

## Legal and Operational Considerations

### Amateur Radio Compliance

⚠️ **Important**: This bridge is for **amateur radio use only**

- **Valid License Required**: You must hold a valid amateur radio license
- **Proper Identification**: All transmissions must include proper station identification
- **Band Plan Compliance**: Use only authorized amateur radio frequencies
- **Third Party Traffic**: Follow your country's third-party traffic regulations

### Discord Terms of Service

- Ensure Discord use complies with their Terms of Service
- Consider Discord's community guidelines for voice channels
- Be respectful of Discord users who may not be familiar with amateur radio procedures

## Troubleshooting

### Discord Connection Issues

```bash
# Test bot token
make run-discord-test

# Check bot permissions in Discord server
# Ensure bot has "Connect" and "Speak" permissions in voice channels
```

### Audio Quality Issues

```bash
# Check FFmpeg installation
ffmpeg -version
ffmpeg -encoders | grep opus

# Test audio conversion separately
make run-audio-test
```

### Network Issues

```bash
# Test USRP packet reception
sudo netstat -ulnp | grep :12345

# Check firewall settings
# Ensure UDP ports are not blocked
```

### Common Error Messages

**"Discord token is required"**
- Set the DISCORD_TOKEN environment variable
- Verify the token is correct

**"FFmpeg not available"**
- Install FFmpeg with Opus support
- Check PATH includes ffmpeg binary

**"Voice connection not ready"**
- Ensure bot has voice channel permissions
- Check if the voice channel exists and is accessible

## Performance Considerations

### Latency
- **Typical latency**: 50-200ms end-to-end
- **Discord voice**: ~40-80ms
- **Audio processing**: ~10-30ms
- **Network**: Variable based on connection

### CPU Usage
- **Bridge process**: ~5-10% on modern hardware
- **FFmpeg conversion**: ~2-5% per active stream
- **Discord bot**: ~1-3%

### Memory Usage
- **Total footprint**: ~50-100MB
- **Audio buffers**: ~10MB
- **Discord connection**: ~20-40MB

## Example Scenarios

### 1. Repeater Extension

Connect a local amateur radio repeater to Discord:

```bash
# Amateur Radio Repeater ←→ USRP Bridge ←→ Discord Voice Channel
export AMATEUR_CALLSIGN="W1AW"
export DISCORD_GUILD="your_club_server"  
export DISCORD_CHANNEL="repeater_link"

make run-discord-bridge
```

### 2. Net Control Station

Use Discord for net control and amateur radio for participants:

```bash
# Net Control (Discord) ←→ Bridge ←→ Amateur Radio Net
export AMATEUR_CALLSIGN="KC1NCS"
make run-discord-bridge
```

### 3. Emergency Communications

Backup communications during emergencies:

```bash
# Emergency Responders (Discord) ←→ Bridge ←→ Amateur Radio Emergency Net
export AMATEUR_CALLSIGN="EM1RGY"
make run-discord-bridge
```

## Development

### Testing the Bridge

```bash
# Run complete test suite
go test ./pkg/discord/ -v

# Test with real Discord connection
make run-discord-test

# Test with simulated amateur radio traffic
make run-discord-server
```

### Extending Functionality

The bridge is designed to be extensible:

- **Custom audio processing**: Modify resampling algorithms
- **Additional protocols**: Add support for other amateur radio protocols
- **Enhanced PTT control**: Implement hardware PTT interfaces
- **Logging and monitoring**: Add comprehensive logging for amateur radio compliance

## Contributing

This is amateur radio software - contributions welcome!

1. Ensure changes maintain amateur radio compliance
2. Test with real amateur radio systems when possible
3. Document any changes that affect legal compliance
4. Follow Go best practices and include tests

## License

MIT License - See LICENSE file for details.

This software is provided for **amateur radio experimentation** under FCC Part 97 regulations and similar international amateur radio regulations.

---

**73, Good DX!** 📻🎮

*"Bridging the gap between amateur radio and modern communications"*

## References

- [FCC Part 97 - Amateur Radio Service](https://www.ecfr.gov/current/title-47/chapter-I/subchapter-D/part-97)
- [Discord Developer Documentation](https://discord.com/developers/docs)
- [AllStarLink Documentation](https://wiki.allstarlink.org/)
- [USRP Protocol Specification](https://github.com/dbehnke/usrp-go)