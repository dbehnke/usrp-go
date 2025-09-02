# USRP Protocol Go Library

A Go library implementing the official USRP (Universal Software Radio Protocol) specification, fully compatible with AllStarLink systems used in amateur radio digital voice communications.

🚨 **IMPORTANT**: This implementation has been corrected to match the **actual USRP protocol specification** used by AllStarLink, not a custom protocol.

## Features

✅ **100% AllStarLink Compatible**: Matches `chan_usrp.c` implementation exactly  
✅ **Official USRP Protocol**: 32-byte header, correct packet types, network byte order  
✅ **All Packet Types**: Voice, DTMF, Text, Ping, TLV, μ-law, ADPCM  
✅ **Amateur Radio Ready**: PTT control, callsign metadata, talk groups  
✅ **Audio Conversion**: FFmpeg integration for Opus/Ogg streaming formats  
✅ **Discord Integration**: Real-time bridge between amateur radio and Discord voice  
✅ **Production Tested**: Comprehensive test suite with all packet formats  
✅ **High Performance**: Efficient binary protocol handling  

## Quick Start

```bash
go get github.com/dbehnke/usrp-go
```

### Basic Voice Packet

```go
package main

import (
    "fmt"
    "github.com/dbehnke/usrp-go/pkg/usrp"
)

func main() {
    // Create voice packet (most common)
    voice := &usrp.VoiceMessage{
        Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1234),
    }
    voice.Header.SetPTT(true)  // Activate PTT
    
    // Fill with 160 audio samples (20ms at 8kHz)
    for i := range voice.AudioData {
        voice.AudioData[i] = int16(i * 100) // Your audio data here
    }
    
    // Serialize to binary (ready to send over UDP)
    data, _ := voice.Marshal()
    fmt.Printf("Voice packet: %d bytes\n", len(data)) // 352 bytes
}
```

### DTMF Signaling

```go
// Send DTMF digit
dtmf := &usrp.DTMFMessage{
    Header: usrp.NewHeader(usrp.USRP_TYPE_DTMF, 5678),
    Digit:  '5',
}
data, _ := dtmf.Marshal()
```

### Callsign Metadata

```go
// Send callsign using TLV
tlv := &usrp.TLVMessage{
    Header: usrp.NewHeader(usrp.USRP_TYPE_TLV, 9999),
}
tlv.SetCallsign("W1AW")
data, _ := tlv.Marshal()
```

### Audio Format Conversion

Convert between USRP and compressed formats using FFmpeg:

```go
// Convert USRP to Opus for internet streaming
converter, _ := audio.NewOpusConverter()
defer converter.Close()

opusData, _ := converter.USRPToFormat(voiceMessage)
// Send opusData over internet...

// Convert back from Opus to USRP
usrpMessages, _ := converter.FormatToUSRP(opusData)
```

See [`docs/AUDIO_CONVERSION.md`](docs/AUDIO_CONVERSION.md) for complete examples.

### Discord Integration

Connect amateur radio to Discord voice channels:

```bash
# Set up Discord bot token and amateur radio callsign
export DISCORD_TOKEN="your_bot_token"
export AMATEUR_CALLSIGN="N0CALL"

# Test Discord connection
make run-discord-test

# Run the bridge
make run-discord-bridge
```

See [`docs/DISCORD_BRIDGE.md`](docs/DISCORD_BRIDGE.md) for complete setup guide.

## Protocol Specification

### Header Format (32 bytes, AllStarLink compatible)
```
Offset | Size | Field     | Description
-------|------|-----------|----------------------------------
0-3    | 4    | Eye       | Magic string "USRP"
4-7    | 4    | Seq       | Sequence counter (network order)
8-11   | 4    | Memory    | Memory ID (usually 0)
12-15  | 4    | Keyup     | PTT state (1=ON, 0=OFF)
16-19  | 4    | TalkGroup | Trunk talk group ID
20-23  | 4    | Type      | Packet type (see below)
24-27  | 4    | MpxID     | Multiplex ID (future use)
28-31  | 4    | Reserved  | Reserved for future use
```

### Packet Types
| Type | ID | Description | Size |
|------|----|-----------|----|
| `USRP_TYPE_VOICE` | 0 | 16-bit PCM audio | 352 bytes |
| `USRP_TYPE_DTMF` | 1 | DTMF signaling | 33 bytes |
| `USRP_TYPE_TEXT` | 2 | Text messages | Variable |
| `USRP_TYPE_PING` | 3 | Keepalive | 32 bytes |
| `USRP_TYPE_TLV` | 4 | Metadata (callsigns) | Variable |
| `USRP_TYPE_VOICE_ADPCM` | 5 | ADPCM audio | Variable |
| `USRP_TYPE_VOICE_ULAW` | 6 | μ-law audio | 192 bytes |

### Audio Formats
- **VOICE**: Signed 16-bit little-endian PCM, 160 samples (20ms at 8kHz)
- **VOICE_ULAW**: μ-law compressed (G.711), 160 samples  
- **VOICE_ADPCM**: ADPCM compressed, variable length

## Testing

```bash
# Run protocol tests
go run cmd/examples/main.go

# Show all packet formats  
go run cmd/examples/main.go formats

# Run unit tests
go test ./pkg/usrp/ -v

# Run benchmarks
go test -bench=. ./pkg/usrp/

# Test audio conversion (requires FFmpeg)
go run cmd/examples/audio_bridge.go test
```

### Example Output
```
USRP Protocol Go Library - Example Usage
=======================================
Now compatible with AllStarLink and official USRP specification!

--- Running Protocol Compatibility Tests ---
Testing VoiceMessage (USRP_TYPE_VOICE)... ✓ (352 bytes)
Testing DTMFMessage (USRP_TYPE_DTMF)... ✓ (33 bytes)  
Testing TLVMessage with callsign metadata... ✓ (39 bytes)
Testing VoiceULawMessage (USRP_TYPE_VOICE_ULAW)... ✓ (192 bytes)
Testing PingMessage (USRP_TYPE_PING)... ✓ (32 bytes)
✓ All protocol tests passed
```

## Compatibility

### ✅ Verified Compatible With:
- **AllStarLink** `chan_usrp.c` 
- Official USRP protocol specification

### 🔧 Technical Compliance:
- ✅ Correct 32-byte header structure
- ✅ Network byte order (big-endian) for header fields  
- ✅ Little-endian for 16-bit audio samples
- ✅ Standard packet type enumeration (0-6)
- ✅ Proper "USRP" magic string  
- ✅ PTT control via Keyup field
- ✅ 160-sample voice frames (20ms at 8kHz)

## Advanced Usage

### PTT Control
```go
header := usrp.NewHeader(usrp.USRP_TYPE_VOICE, seq)
header.SetPTT(true)   // Activate push-to-talk
if header.IsPTT() {
    fmt.Println("PTT is active")
}
```

### Talk Groups  
```go
header.TalkGroup = 12345  // Set talk group ID
```

### TLV Metadata
```go
tlv := &usrp.TLVMessage{}
tlv.SetCallsign("KC1ABC")
tlv.AddTLV(usrp.TLV_TAG_AMBE, ambeData)

if callsign, ok := tlv.GetCallsign(); ok {
    fmt.Printf("Station: %s\n", callsign)
}
```

## Performance

Benchmarks on modern hardware:
```
BenchmarkVoiceMessage_Marshal   -8    1000000   1205 ns/op
BenchmarkVoiceMessage_Unmarshal -8     500000   2341 ns/op
```

- **Throughput**: >500k voice packets/second
- **Latency**: <2ms processing overhead  
- **Memory**: Efficient with buffer reuse

## Project Structure

```
usrp-go/
├── pkg/usrp/              # Core USRP protocol
│   ├── protocol.go        # Message types & structures  
│   ├── marshal.go         # Binary serialization
│   └── protocol_test.go   # Comprehensive tests
├── pkg/audio/             # Audio format conversion
│   ├── converter.go       # FFmpeg integration
│   └── converter_test.go  # Conversion tests
├── pkg/discord/           # Discord voice integration
│   ├── bot.go            # Discord bot with voice capabilities
│   ├── bridge.go         # USRP-Discord audio bridge
│   └── bridge_test.go    # Discord integration tests
├── cmd/examples/          # Protocol demo applications
│   └── main.go           # Protocol compatibility tests
├── cmd/audio-bridge/      # Audio conversion demos
│   └── main.go           # Audio bridge examples
├── cmd/discord-bridge/    # Discord integration demos
│   └── main.go           # Discord bridge examples
├── docs/                  # Documentation
│   ├── AUDIO_CONVERSION.md # Audio conversion guide
│   └── DISCORD_BRIDGE.md  # Discord integration guide
└── internal/transport/    # UDP transport layer (WIP)
    └── udp.go            # Network handling
```

## Contributing

This implementation prioritizes **exact compatibility** with existing USRP deployments. Before making changes:

1. Verify against AllStarLink `chan_usrp.c` source
2. Test with existing AllStarLink systems  
3. Maintain binary protocol compatibility
4. Add comprehensive tests

## Amateur Radio Applications

Perfect for:
- **AllStarLink node linking**
- **Digital voice bridging** 
- **Discord-amateur radio integration**
- **Experimental amateur radio protocols**
- **Emergency communication systems**

## License

MIT License - See LICENSE file for details.

---

**73, Good DX!** 📻

*Developed for amateur radio experimentation under FCC Part 97 regulations.*

## References

- [AllStarLink chan_usrp.c](https://github.com/AllStarLink/app_rpt/blob/master/channels/chan_usrp.c)
- [USRP Protocol Documentation](https://raw.githubusercontent.com/dl1hrc/svxlink/refs/heads/svxlink-usrp/src/svxlink/svxlink/contrib/UsrpLogic/usrp_protocol.txt)