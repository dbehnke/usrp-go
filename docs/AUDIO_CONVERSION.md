# Audio Conversion with FFmpeg

This document explains how to use FFmpeg to convert audio data between USRP packets and other formats like Opus/Ogg for streaming applications.

## Overview

The USRP protocol uses **signed 16-bit little-endian PCM** at **8kHz mono** with **160 samples per packet** (20ms frames). This is perfect for voice applications but you may want to convert to compressed formats for:

- **Bandwidth efficiency**: Opus/Ogg provides much better compression than raw PCM
- **Internet streaming**: Standard formats work with existing streaming infrastructure  
- **Storage**: Compressed formats reduce file sizes significantly
- **Compatibility**: Integration with WebRTC, SIP, or other VoIP systems

## Key Features

✅ **Real-time conversion** between USRP and Opus/Ogg formats  
✅ **Streaming support** for non-continuous audio (PTT-based)  
✅ **Bidirectional** conversion (USRP ↔ Opus/Ogg)  
✅ **Frame-accurate** processing with 20ms alignment  
✅ **Thread-safe** with concurrent processing  
✅ **Error handling** for network timeouts and format issues  

## Requirements

- **FFmpeg** with `libopus` support
- Go 1.19+ 

### Installing FFmpeg with Opus Support

```bash
# macOS (Homebrew)
brew install ffmpeg

# Ubuntu/Debian
sudo apt update
sudo apt install ffmpeg libopus-dev

# CentOS/RHEL
sudo yum install ffmpeg opus-devel

# Windows
# Download from https://ffmpeg.org/download.html
# Make sure it includes libopus
```

Verify FFmpeg has Opus support:
```bash
ffmpeg -encoders | grep opus
```

Should show:
```
 A..... libopus              libopus Opus
```

## Basic Usage

### Simple Conversion Example

```go
package main

import (
    "fmt"
    "github.com/dbehnke/usrp-go/pkg/audio"
    "github.com/dbehnke/usrp-go/pkg/usrp"
)

func main() {
    // Create Opus converter
    converter, err := audio.NewOpusConverter()
    if err != nil {
        panic(err)
    }
    defer converter.Close()

    // Create USRP voice message
    voiceMsg := &usrp.VoiceMessage{
        Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1234),
    }
    
    // Fill with your audio data (160 samples)
    for i := range voiceMsg.AudioData {
        voiceMsg.AudioData[i] = int16(i * 100) // Your audio here
    }

    // Convert USRP -> Opus
    opusData, err := converter.USRPToFormat(voiceMsg)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Converted to %d bytes of Opus\n", len(opusData))

    // Convert back: Opus -> USRP  
    usrpMessages, err := converter.FormatToUSRP(opusData)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Converted back to %d USRP messages\n", len(usrpMessages))
}
```

### Streaming Audio Bridge

```go
package main

import (
    "github.com/dbehnke/usrp-go/pkg/audio" 
    "github.com/dbehnke/usrp-go/pkg/usrp"
)

func main() {
    // Create converter and bridge
    converter, _ := audio.NewOpusConverter()
    bridge := audio.NewAudioBridge(converter)
    defer bridge.Stop()

    bridge.Start()

    // Stream USRP packets through bridge
    for {
        // Receive USRP packet from network...
        voiceMsg := receiveUSRPPacket()
        
        // Send to bridge for conversion
        bridge.USRPIn <- voiceMsg
        
        // Get converted Opus data
        select {
        case opusData := <-bridge.USRPToChan:
            // Send Opus data over internet...
            sendOpusData(opusData)
        default:
            // No data yet
        }
    }
}
```

## Audio Formats Supported

### USRP Format (Input)
- **Format**: Signed 16-bit little-endian PCM
- **Sample Rate**: 8000 Hz  
- **Channels**: 1 (mono)
- **Frame Size**: 160 samples (20ms)
- **Bandwidth**: ~128 kbps uncompressed

### Opus (Output) 
- **Format**: Opus compressed audio
- **Sample Rate**: 8000 Hz (matches USRP)
- **Channels**: 1 (mono)
- **Bitrate**: 64 kbps (configurable)
- **Frame Size**: 20ms (matches USRP perfectly)
- **Compression**: ~50% reduction in bandwidth

### Ogg/Opus (Output)
- **Format**: Opus in Ogg container
- **Same specs as Opus** but with container overhead
- **Better for file storage** and streaming protocols

## Advanced Configuration

### Custom Converter Settings

```go
config := &audio.ConverterConfig{
    InputFormat:  "s16le",    // PCM 16-bit little-endian
    OutputFormat: "opus",     // Target format
    InputRate:    8000,       // USRP sample rate
    OutputRate:   8000,       // Keep same rate
    Channels:     1,          // Mono
    BitRate:      32,         // Lower bitrate (32 kbps)
    FrameSize:    20 * time.Millisecond, // 20ms frames
}

converter, err := audio.NewStreamingConverter(config)
```

### Error Handling & Timeouts

```go
// The converter includes timeout handling for streaming data
converter, _ := audio.NewOpusConverter()

voiceMsg := &usrp.VoiceMessage{ /* ... */ }

opusData, err := converter.USRPToFormat(voiceMsg)
if err != nil {
    if strings.Contains(err.Error(), "timeout") {
        // FFmpeg not ready or no data yet
        // This is normal for streaming applications
    } else {
        // Real error - check FFmpeg installation
        log.Printf("Conversion error: %v", err)
    }
}
```

## Integration Patterns

### 1. AllStarLink Bridge

Convert USRP from AllStarLink to Opus for internet transmission:

```go
// Receive from AllStarLink (USRP)
usrpConn, _ := net.ListenUDP("udp", &net.UDPAddr{Port: 12345})

// Send to internet client (Opus)  
opusConn, _ := net.DialUDP("udp", nil, &net.UDPAddr{
    IP: net.ParseIP("remote.server.com"), Port: 8000,
})

converter, _ := audio.NewOpusConverter()

for {
    buffer := make([]byte, 1024)
    n, _ := usrpConn.Read(buffer)
    
    voiceMsg := &usrp.VoiceMessage{}
    voiceMsg.Unmarshal(buffer[:n])
    
    opusData, _ := converter.USRPToFormat(voiceMsg) 
    opusConn.Write(opusData)
}
```

### 2. WebRTC Integration

Stream amateur radio to web browsers:

```go
// Convert USRP to Opus for WebRTC
converter, _ := audio.NewOpusConverter()

// WebRTC expects Opus in specific format
webrtcConfig := &audio.ConverterConfig{
    OutputFormat: "opus",
    BitRate:      32,  // Lower for web
    FrameSize:    20 * time.Millisecond,
}

for {
    voiceMsg := getUSRPFromRadio()
    opusData, _ := converter.USRPToFormat(voiceMsg)
    
    // Send to WebRTC peer connection
    sendToWebRTC(opusData)
}
```

### 3. Recording & Playback

Record amateur radio transmissions:

```go
// Record to Ogg file
converter, _ := audio.NewOggOpusConverter()
file, _ := os.Create("recording.ogg")

for {
    voiceMsg := getUSRPFromRadio()
    if voiceMsg.Header.IsPTT() {
        oggData, _ := converter.USRPToFormat(voiceMsg)
        file.Write(oggData)
    }
}
```

## Performance Considerations

### Latency
- **FFmpeg processing**: ~5-15ms additional latency
- **Network buffering**: Adjust based on jitter requirements
- **Frame alignment**: 20ms frames minimize latency impact

### CPU Usage  
- **Opus encoding**: ~1-3% CPU per stream on modern hardware
- **Memory**: ~10MB per converter instance  
- **Concurrent streams**: Each converter runs separate FFmpeg processes

### Bandwidth Savings

| Format | Bandwidth | Compression |
|--------|-----------|-------------|
| USRP PCM | 128 kbps | None |
| Opus 64k | 64 kbps | 50% |
| Opus 32k | 32 kbps | 75% |
| Opus 16k | 16 kbps | 87% |

## Testing

```bash
# Run conversion tests
make run-audio-test

# Test server/client bridge  
# Terminal 1:
make run-audio-server

# Terminal 2:  
make run-audio-client

# Unit tests
go test ./pkg/audio/ -v

# Benchmarks
go test -bench=. ./pkg/audio/
```

## Troubleshooting

### "FFmpeg not found"
```bash
# Check FFmpeg installation
which ffmpeg
ffmpeg -version
```

### "libopus not supported"
```bash  
# Check available encoders
ffmpeg -encoders | grep opus

# Reinstall with Opus support
brew reinstall ffmpeg  # macOS
```

### "Read timeout" errors
- Normal for streaming applications
- Indicates FFmpeg needs more data or isn't ready
- Not an error - just retry the operation

### High CPU usage
- Reduce bitrate: 64k → 32k → 16k
- Use fewer concurrent converters
- Consider hardware acceleration if available

### Poor audio quality
- Increase bitrate: 32k → 64k
- Check input audio quality (USRP source)
- Verify sample rate matches (8000 Hz)

## Example Applications

The repository includes complete examples:

- **`cmd/examples/audio_bridge.go`**: Full bidirectional converter
- **`pkg/audio/converter_test.go`**: Unit tests and benchmarks  
- **Server/client demos**: Real-time streaming over UDP

## Future Enhancements

Potential improvements:

- **Hardware acceleration**: GPU-based encoding/decoding
- **Multiple formats**: MP3, AAC, G.722 support
- **Adaptive bitrate**: Dynamic quality based on network conditions
- **WebSocket streaming**: Direct browser integration
- **SIP integration**: Direct VoIP connectivity

## Amateur Radio Applications

Perfect for:

- **Internet linking**: Connect distant repeaters over internet
- **Emergency communications**: Backup paths using internet infrastructure  
- **Digital modes**: Bridge analog FM to digital systems
- **Recording systems**: Efficient storage of transmissions
- **Web monitoring**: Listen to repeaters via web browser
- **Mobile apps**: Stream amateur radio to smartphones

This audio conversion system enables amateur radio systems to leverage modern internet infrastructure while maintaining compatibility with traditional USRP-based networks.

---

**73!** 📻 Happy experimenting with software-defined radio!