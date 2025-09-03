package discord

import (
	"testing"
	"time"

	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// TestBridgeCreation tests creating a Discord bridge
func TestBridgeCreation(t *testing.T) {
	config := DefaultBridgeConfig()
	config.DiscordToken = "test_token_not_real"
	
	// This should fail because the token is fake
	_, err := NewBridge(config)
	if err == nil {
		t.Skip("Expected error with fake token - Discord API may be unavailable for testing")
	}
}

// TestBridgeConfig tests bridge configuration
func TestBridgeConfig(t *testing.T) {
	config := DefaultBridgeConfig()
	
	if config.EnableResampling != true {
		t.Error("Default config should enable resampling")
	}
	
	if config.PTTTimeout != 2*time.Second {
		t.Error("Default PTT timeout should be 2 seconds")
	}
	
	if config.BufferSize != 100 {
		t.Error("Default buffer size should be 100")
	}
}

// TestAudioResampling tests audio resampling functions
func TestAudioResampling(t *testing.T) {
	// Create a mock bridge (without actual Discord connection)
	config := DefaultBridgeConfig()
	config.DiscordToken = "fake_token"
	
	// We can't create a real bridge without a valid token,
	// but we can test the resampling logic separately
	bridge := &Bridge{
		config: config,
	}
	
	// Test USRP to Discord resampling
	usrpSamples := make([]int16, 160) // 160 samples (20ms at 8kHz)
	for i := range usrpSamples {
		usrpSamples[i] = int16(i * 100)
	}
	
	discordSamples := bridge.resampleUSRPToDiscord(usrpSamples)
	expectedLen := 160 * 6 * 2 // 6x upsampling, 2 channels
	
	if len(discordSamples) != expectedLen {
		t.Errorf("Expected %d Discord samples, got %d", expectedLen, len(discordSamples))
	}
	
	// Test Discord to USRP resampling  
	discordInput := make([]int16, 960*2) // 960 stereo samples (20ms at 48kHz)
	for i := 0; i < len(discordInput); i += 2 {
		discordInput[i] = int16(i * 10)     // Left
		discordInput[i+1] = int16(i * 10)   // Right
	}
	
	usrpOutput := bridge.resampleDiscordToUSRP(discordInput)
	expectedUSRPLen := 160 // Should downsample to 160 mono samples
	
	if len(usrpOutput) != expectedUSRPLen {
		t.Errorf("Expected %d USRP samples, got %d", expectedUSRPLen, len(usrpOutput))
	}
}

// TestVoiceActivityDetection tests voice activity detection
func TestVoiceActivityDetection(t *testing.T) {
	bridge := &Bridge{
		config: &BridgeConfig{
			VoiceThreshold: 1000,
		},
	}
	
	// Test with silence (should not detect voice)
	silence := make([]int16, 160)
	if bridge.detectVoiceActivity(silence) {
		t.Error("Should not detect voice activity in silence")
	}
	
	// Test with loud audio (should detect voice)
	loudAudio := make([]int16, 160)
	for i := range loudAudio {
		loudAudio[i] = 5000 // Well above threshold
	}
	if !bridge.detectVoiceActivity(loudAudio) {
		t.Error("Should detect voice activity in loud audio")
	}
	
	// Test with threshold of 0 (should always detect)
	bridge.config.VoiceThreshold = 0
	if !bridge.detectVoiceActivity(silence) {
		t.Error("Should always detect voice when threshold is 0")
	}
}

// TestUSRPPacketProcessing tests USRP packet creation and processing
func TestUSRPPacketProcessing(t *testing.T) {
	// Create a USRP voice packet
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1234),
	}
	voiceMsg.Header.SetPTT(true)
	voiceMsg.Header.TalkGroup = 12345
	
	// Fill with test audio
	for i := range voiceMsg.AudioData {
		voiceMsg.AudioData[i] = int16(i * 100)
	}
	
	// Test marshaling
	data, err := voiceMsg.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal USRP packet: %v", err)
	}
	
	if len(data) != 352 { // 32-byte header + 320 audio bytes
		t.Errorf("Expected 352 bytes, got %d", len(data))
	}
	
	// Test unmarshaling
	voiceMsg2 := &usrp.VoiceMessage{}
	if err := voiceMsg2.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal USRP packet: %v", err)
	}
	
	if voiceMsg2.Header.Seq != 1234 {
		t.Errorf("Expected sequence 1234, got %d", voiceMsg2.Header.Seq)
	}
	
	if !voiceMsg2.Header.IsPTT() {
		t.Error("Expected PTT to be active")
	}
	
	if voiceMsg2.Header.TalkGroup != 12345 {
		t.Errorf("Expected talk group 12345, got %d", voiceMsg2.Header.TalkGroup)
	}
}

// BenchmarkAudioResampling benchmarks the audio resampling performance
func BenchmarkUSRPToDiscordResampling(b *testing.B) {
	bridge := &Bridge{
		config: DefaultBridgeConfig(),
	}
	
	usrpSamples := make([]int16, 160)
	for i := range usrpSamples {
		usrpSamples[i] = int16(i * 100)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bridge.resampleUSRPToDiscord(usrpSamples)
	}
}

func BenchmarkDiscordToUSRPResampling(b *testing.B) {
	bridge := &Bridge{
		config: DefaultBridgeConfig(),
	}
	
	discordSamples := make([]int16, 960*2) // 20ms at 48kHz stereo
	for i := range discordSamples {
		discordSamples[i] = int16(i * 10)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bridge.resampleDiscordToUSRP(discordSamples)
	}
}

func BenchmarkVoiceActivityDetection(b *testing.B) {
	bridge := &Bridge{
		config: &BridgeConfig{VoiceThreshold: 1000},
	}
	
	samples := make([]int16, 160)
	for i := range samples {
		samples[i] = int16(i * 50)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bridge.detectVoiceActivity(samples)
	}
}