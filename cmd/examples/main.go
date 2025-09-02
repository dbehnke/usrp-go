// Example usage and comprehensive tests for the USRP Go library
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dbehnke/usrp-go/pkg/usrp"
)

func main() {
	fmt.Println("USRP Protocol Go Library - Example Usage")
	fmt.Println("=======================================")
	fmt.Println("Now compatible with AllStarLink and official USRP specification!")

	// Run basic protocol tests first
	if err := runProtocolTests(); err != nil {
		log.Fatalf("Protocol tests failed: %v", err)
	}
	fmt.Println("✓ All protocol tests passed")

	// Check if we should run the format demo
	if len(os.Args) > 1 && os.Args[1] == "formats" {
		runFormatDemo()
	} else {
		fmt.Println("\nRun with 'formats' argument to see all supported packet types")
		fmt.Println("Example: go run cmd/examples/main.go formats")
	}
}

// runProtocolTests runs comprehensive tests of the USRP protocol implementation
func runProtocolTests() error {
	fmt.Println("\n--- Running Protocol Compatibility Tests ---")

	// Test 1: VoiceMessage (most common)
	fmt.Print("Testing VoiceMessage (USRP_TYPE_VOICE)... ")
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1234),
	}
	voiceMsg.Header.SetPTT(true)
	
	// Fill with test audio pattern (160 samples = 20ms at 8kHz)
	for i := range voiceMsg.AudioData {
		voiceMsg.AudioData[i] = int16(i * 100) // Test pattern
	}

	data, err := voiceMsg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal voice message: %w", err)
	}

	decoded := &usrp.VoiceMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		return fmt.Errorf("failed to unmarshal voice message: %w", err)
	}

	if !decoded.Header.IsPTT() {
		return fmt.Errorf("PTT state not preserved")
	}
	fmt.Printf("✓ (%d bytes)\n", len(data))

	// Test 2: DTMFMessage
	fmt.Print("Testing DTMFMessage (USRP_TYPE_DTMF)... ")
	dtmfMsg := &usrp.DTMFMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_DTMF, 5678),
		Digit:  '5',
	}

	data, err = dtmfMsg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal DTMF message: %w", err)
	}

	decodedDTMF := &usrp.DTMFMessage{}
	if err := decodedDTMF.Unmarshal(data); err != nil {
		return fmt.Errorf("failed to unmarshal DTMF message: %w", err)
	}

	if decodedDTMF.Digit != '5' {
		return fmt.Errorf("DTMF digit mismatch")
	}
	fmt.Printf("✓ (%d bytes)\n", len(data))

	// Test 3: TLVMessage with callsign
	fmt.Print("Testing TLVMessage with callsign metadata... ")
	tlvMsg := &usrp.TLVMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_TLV, 9999),
	}
	tlvMsg.SetCallsign("W1AW")

	data, err = tlvMsg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal TLV message: %w", err)
	}

	decodedTLV := &usrp.TLVMessage{}
	if err := decodedTLV.Unmarshal(data); err != nil {
		return fmt.Errorf("failed to unmarshal TLV message: %w", err)
	}

	callsign, ok := decodedTLV.GetCallsign()
	if !ok || callsign != "W1AW" {
		return fmt.Errorf("callsign not preserved: got %s", callsign)
	}
	fmt.Printf("✓ (%d bytes)\n", len(data))

	// Test 4: μ-law voice
	fmt.Print("Testing VoiceULawMessage (USRP_TYPE_VOICE_ULAW)... ")
	ulawMsg := &usrp.VoiceULawMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE_ULAW, 1111),
	}
	
	// Fill with μ-law test pattern
	for i := range ulawMsg.AudioData {
		ulawMsg.AudioData[i] = byte(i % 256)
	}

	data, err = ulawMsg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal μ-law voice: %w", err)
	}

	decodedULaw := &usrp.VoiceULawMessage{}
	if err := decodedULaw.Unmarshal(data); err != nil {
		return fmt.Errorf("failed to unmarshal μ-law voice: %w", err)
	}
	fmt.Printf("✓ (%d bytes)\n", len(data))

	// Test 5: Ping message
	fmt.Print("Testing PingMessage (USRP_TYPE_PING)... ")
	pingMsg := &usrp.PingMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_PING, 7777),
	}

	data, err = pingMsg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal ping: %w", err)
	}

	decodedPing := &usrp.PingMessage{}
	if err := decodedPing.Unmarshal(data); err != nil {
		return fmt.Errorf("failed to unmarshal ping: %w", err)
	}

	// Should be exactly header size
	if len(data) != usrp.HeaderSize {
		return fmt.Errorf("ping size mismatch: got %d, want %d", len(data), usrp.HeaderSize)
	}
	fmt.Printf("✓ (%d bytes)\n", len(data))

	return nil
}

// runFormatDemo shows all supported packet formats
func runFormatDemo() {
	fmt.Println("\n--- USRP Packet Format Demonstration ---")
	
	packetTypes := []struct {
		name        string
		packetType  usrp.PacketType
		description string
	}{
		{"VOICE", usrp.USRP_TYPE_VOICE, "16-bit PCM audio, 160 samples (20ms at 8kHz)"},
		{"DTMF", usrp.USRP_TYPE_DTMF, "DTMF tone signaling ('0'-'9', 'A'-'D', '*', '#')"},
		{"TEXT", usrp.USRP_TYPE_TEXT, "Text messages and metadata"},
		{"PING", usrp.USRP_TYPE_PING, "Keepalive/heartbeat packets"},
		{"TLV", usrp.USRP_TYPE_TLV, "Type-Length-Value metadata (callsigns, etc.)"},
		{"VOICE_ADPCM", usrp.USRP_TYPE_VOICE_ADPCM, "ADPCM compressed audio"},
		{"VOICE_ULAW", usrp.USRP_TYPE_VOICE_ULAW, "μ-law compressed audio (160 samples)"},
	}

	fmt.Printf("%-12s %-5s %s\n", "TYPE", "ID", "DESCRIPTION")
	fmt.Printf("%-12s %-5s %s\n", "----", "--", "-----------")
	
	for _, pt := range packetTypes {
		fmt.Printf("%-12s %-5d %s\n", pt.name, pt.packetType, pt.description)
	}

	fmt.Println("\n--- Header Structure (32 bytes, AllStarLink compatible) ---")
	fmt.Println("Offset | Size | Field     | Description")
	fmt.Println("-------|------|-----------|----------------------------------")
	fmt.Println("0-3    | 4    | Eye       | Magic string \"USRP\"")
	fmt.Println("4-7    | 4    | Seq       | Sequence counter (network order)")
	fmt.Println("8-11   | 4    | Memory    | Memory ID (usually 0)")
	fmt.Println("12-15  | 4    | Keyup     | PTT state (1=ON, 0=OFF)")
	fmt.Println("16-19  | 4    | TalkGroup | Trunk talk group ID")
	fmt.Println("20-23  | 4    | Type      | Packet type (see above)")
	fmt.Println("24-27  | 4    | MpxID     | Multiplex ID (future use)")
	fmt.Println("28-31  | 4    | Reserved  | Reserved for future use")

	fmt.Println("\n--- Audio Formats ---")
	fmt.Println("VOICE:      Signed 16-bit little-endian PCM")
	fmt.Println("VOICE_ULAW: μ-law compressed (G.711)")
	fmt.Println("VOICE_ADPCM: ADPCM compressed (variable length)")

	fmt.Println("\n--- TLV Tags ---")
	fmt.Printf("TLV_TAG_SET_INFO (0x%02X): Station callsign and metadata\n", usrp.TLV_TAG_SET_INFO)
	fmt.Printf("TLV_TAG_AMBE     (0x%02X): AMBE vocoder data\n", usrp.TLV_TAG_AMBE)
	fmt.Printf("TLV_TAG_DTMF     (0x%02X): DTMF tone information\n", usrp.TLV_TAG_DTMF)

	fmt.Println("\n--- Compatibility Notes ---")
	fmt.Println("✓ Compatible with AllStarLink chan_usrp.c")
	fmt.Println("✓ Network byte order (big-endian) for header fields")
	fmt.Println("✓ Little-endian for 16-bit audio samples")
	fmt.Println("✓ Fixed 32-byte header size")
	fmt.Println("✓ Standard 160-sample voice frames (20ms)")
}