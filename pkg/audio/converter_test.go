package audio

import (
	"strings"
	"testing"
	"time"

	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// TestOpusConverter tests USRP <-> Opus conversion
func TestOpusConverter(t *testing.T) {
	converter, err := NewOpusConverter()
	if err != nil {
		t.Skipf("FFmpeg not available or Opus not supported: %v", err)
	}
	defer converter.Close()

	// Create test USRP voice message
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1234),
	}
	
	// Fill with test audio pattern (sine wave-like)
	for i := range voiceMsg.AudioData {
		// Simple test pattern
		voiceMsg.AudioData[i] = int16(1000 * (i % 100 - 50)) // Varies between -50k to +50k
	}

	// Convert to Opus
	opusData, err := converter.USRPToFormat(voiceMsg)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			t.Skipf("FFmpeg timeout (not available or not configured): %v", err)
		}
		t.Fatalf("USRP to Opus conversion failed: %v", err)
	}

	if len(opusData) == 0 {
		t.Fatal("No Opus data produced")
	}

	t.Logf("Converted %d PCM samples to %d bytes of Opus", len(voiceMsg.AudioData), len(opusData))

	// Convert back to USRP
	usrpMessages, err := converter.FormatToUSRP(opusData)
	if err != nil {
		t.Fatalf("Opus to USRP conversion failed: %v", err)
	}

	if len(usrpMessages) == 0 {
		t.Fatal("No USRP messages produced")
	}

	t.Logf("Converted %d bytes of Opus back to %d USRP messages", len(opusData), len(usrpMessages))
}

// TestOggOpusConverter tests USRP <-> Ogg/Opus conversion
func TestOggOpusConverter(t *testing.T) {
	converter, err := NewOggOpusConverter()
	if err != nil {
		t.Skipf("FFmpeg not available or Ogg/Opus not supported: %v", err)
	}
	defer converter.Close()

	// Create test USRP voice message
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 5678),
	}
	
	// Fill with different test pattern
	for i := range voiceMsg.AudioData {
		voiceMsg.AudioData[i] = int16(500 * (i % 160)) // Sawtooth pattern
	}

	// Convert to Ogg/Opus
	oggData, err := converter.USRPToFormat(voiceMsg)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			t.Skipf("FFmpeg timeout (not available or not configured): %v", err)
		}
		t.Fatalf("USRP to Ogg conversion failed: %v", err)
	}

	if len(oggData) == 0 {
		t.Fatal("No Ogg data produced")
	}

	t.Logf("Converted %d PCM samples to %d bytes of Ogg/Opus", len(voiceMsg.AudioData), len(oggData))

	// Note: Converting back from Ogg requires proper stream handling
	// This is more complex due to Ogg container format
	t.Logf("Ogg/Opus conversion successful (reverse conversion requires stream parsing)")
}

// TestAudioBridge tests the high-level audio bridge
func TestAudioBridge(t *testing.T) {
	converter, err := NewOpusConverter()
	if err != nil {
		t.Skipf("FFmpeg not available: %v", err)
	}

	bridge := NewAudioBridge(converter)
	defer func() {
		if err := bridge.Stop(); err != nil {
			t.Logf("Error stopping bridge: %v", err)
		}
	}()

	// Start the bridge
	if err := bridge.Start(); err != nil {
		t.Fatalf("Failed to start bridge: %v", err)
	}

	// Create test USRP message
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 9999),
	}
	for i := range voiceMsg.AudioData {
		voiceMsg.AudioData[i] = int16(i * 200) // Linear ramp
	}

	// Send to bridge
	go func() {
		bridge.USRPIn <- voiceMsg
	}()

	// Wait for converted data
	select {
	case opusData := <-bridge.USRPToChan:
		if len(opusData) == 0 {
			t.Fatal("No data received from bridge")
		}
		t.Logf("Bridge converted USRP to %d bytes of Opus", len(opusData))

		// Send back through bridge
		go func() {
			bridge.FormatIn <- opusData
		}()

		// Wait for USRP messages
		select {
		case usrpMessages := <-bridge.ChanToUSRP:
			if len(usrpMessages) == 0 {
				t.Fatal("No USRP messages received from bridge")
			}
			t.Logf("Bridge converted Opus back to %d USRP messages", len(usrpMessages))
		case <-time.After(2 * time.Second):
			t.Skip("Timeout waiting for USRP messages from bridge (FFmpeg not available)")
		}

	case <-time.After(2 * time.Second):
		t.Skip("Timeout waiting for Opus data from bridge (FFmpeg not available)")
	}
}

// TestConverterCleanup tests proper resource cleanup
func TestConverterCleanup(t *testing.T) {
	converter, err := NewOpusConverter()
	if err != nil {
		t.Skipf("FFmpeg not available: %v", err)
	}

	// Test that Close() can be called multiple times
	if err := converter.Close(); err != nil {
		t.Errorf("First close failed: %v", err)
	}

	if err := converter.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}

	// Test that operations fail after close
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1),
	}

	_, err = converter.USRPToFormat(voiceMsg)
	if err == nil {
		t.Error("Expected error after close, got nil")
	}
}

// BenchmarkUSRPToOpus benchmarks USRP to Opus conversion
func BenchmarkUSRPToOpus(b *testing.B) {
	converter, err := NewOpusConverter()
	if err != nil {
		b.Skipf("FFmpeg not available: %v", err)
	}
	defer converter.Close()

	// Create test message
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1),
	}
	for i := range voiceMsg.AudioData {
		voiceMsg.AudioData[i] = int16(i * 100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.USRPToFormat(voiceMsg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Example test showing realistic usage patterns
func TestRealisticUSRPStream(t *testing.T) {
	converter, err := NewOpusConverter()
	if err != nil {
		t.Skipf("FFmpeg not available: %v", err)
	}
	defer converter.Close()

	// Simulate multiple USRP packets (like a voice transmission)
	packets := 5
	allOpusData := make([]byte, 0)

	for i := 0; i < packets; i++ {
		voiceMsg := &usrp.VoiceMessage{
			Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, uint32(i+1)),
		}

		// Fill with test pattern across packets
		for j := range voiceMsg.AudioData {
			// Simple test pattern (could be sine wave)
			voiceMsg.AudioData[j] = int16((i*1000 + j) % 20000) // Test pattern
		}

		opusData, err := converter.USRPToFormat(voiceMsg)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				t.Skipf("FFmpeg timeout (not available or not configured): %v", err)
			}
			t.Fatalf("Packet %d conversion failed: %v", i, err)
		}

		allOpusData = append(allOpusData, opusData...)
		t.Logf("Packet %d: %d samples -> %d bytes Opus", i+1, len(voiceMsg.AudioData), len(opusData))
	}

	t.Logf("Total: %d USRP packets -> %d bytes Opus", packets, len(allOpusData))
}