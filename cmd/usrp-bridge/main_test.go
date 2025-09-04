package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestConfigGeneration tests configuration file generation and loading
func TestConfigGeneration(t *testing.T) {
	// Test default configuration
	config := defaultConfig()

	if config.USRPListenPort != 12345 {
		t.Errorf("Expected default USRP listen port 12345, got %d", config.USRPListenPort)
	}

	if config.StationCall != "N0CALL" {
		t.Errorf("Expected default station call N0CALL, got %s", config.StationCall)
	}

	if !config.AudioConfig.EnableConversion {
		t.Error("Expected audio conversion to be enabled by default")
	}

	if config.AudioConfig.OutputFormat != "opus" {
		t.Errorf("Expected default audio format opus, got %s", config.AudioConfig.OutputFormat)
	}

	if len(config.Destinations) == 0 {
		t.Error("Expected at least one destination in default config")
	}

	// Test that whotalkie destination is enabled by default
	foundWhoTalkie := false
	for _, dest := range config.Destinations {
		if dest.Name == "whotalkie" && dest.Enabled {
			foundWhoTalkie = true
			break
		}
	}

	if !foundWhoTalkie {
		t.Error("Expected whotalkie destination to be enabled by default")
	}
}

// TestConfigSerialization tests JSON marshaling/unmarshaling
func TestConfigSerialization(t *testing.T) {
	originalConfig := defaultConfig()
	originalConfig.StationCall = "W1AW"
	originalConfig.USRPListenPort = 54321

	// Marshal to JSON
	data, err := json.MarshalIndent(originalConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back
	var loadedConfig Config
	if err := json.Unmarshal(data, &loadedConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify key fields
	if loadedConfig.StationCall != "W1AW" {
		t.Errorf("Expected station call W1AW, got %s", loadedConfig.StationCall)
	}

	if loadedConfig.USRPListenPort != 54321 {
		t.Errorf("Expected USRP listen port 54321, got %d", loadedConfig.USRPListenPort)
	}

	if len(loadedConfig.Destinations) != len(originalConfig.Destinations) {
		t.Errorf("Expected %d destinations, got %d",
			len(originalConfig.Destinations), len(loadedConfig.Destinations))
	}
}

// TestConfigFileOperations tests config file load/save
func TestConfigFileOperations(t *testing.T) {
	testFile := "test-config.json"
	defer os.Remove(testFile) // Cleanup

	// Create test config
	config := defaultConfig()
	config.StationCall = "KC1TEST"
	config.USRPListenPort = 9999

	// Save to file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load from file
	loadedConfig, err := loadConfig(testFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify
	if loadedConfig.StationCall != "KC1TEST" {
		t.Errorf("Expected station call KC1TEST, got %s", loadedConfig.StationCall)
	}

	if loadedConfig.USRPListenPort != 9999 {
		t.Errorf("Expected USRP listen port 9999, got %d", loadedConfig.USRPListenPort)
	}
}

// TestBridgeStatsInitialization tests bridge statistics initialization
func TestBridgeStatsInitialization(t *testing.T) {
	stats := &BridgeStats{}

	if stats.USRPPacketsReceived != 0 {
		t.Error("Expected initial USRP packets received to be 0")
	}

	if stats.LastActivityTime != 0 {
		t.Error("Expected initial last activity time to be 0")
	}

	// Test updating stats
	stats.USRPPacketsReceived++
	stats.LastActivityTime = time.Now().Unix()

	if stats.USRPPacketsReceived != 1 {
		t.Error("Expected USRP packets received to be 1 after increment")
	}

	if stats.LastActivityTime == 0 {
		t.Error("Expected last activity time to be updated")
	}
}

// TestDestinationConfig tests destination configuration validation
func TestDestinationConfig(t *testing.T) {
	// Test WhoTalkie destination
	whoTalkieConfig := DestinationConfig{
		Name:     "whotalkie",
		Type:     "whotalkie",
		Host:     "127.0.0.1",
		Port:     8080,
		Protocol: "udp",
		Format:   "opus",
		Enabled:  true,
	}

	if whoTalkieConfig.Name != "whotalkie" {
		t.Error("WhoTalkie destination name mismatch")
	}

	if whoTalkieConfig.Type != "whotalkie" {
		t.Error("WhoTalkie destination type mismatch")
	}

	if whoTalkieConfig.Format != "opus" {
		t.Error("WhoTalkie destination should use opus format")
	}

	// Test Discord destination
	discordConfig := DestinationConfig{
		Name:     "discord-bot",
		Type:     "discord",
		Host:     "127.0.0.1",
		Port:     8081,
		Protocol: "udp",
		Format:   "opus",
		Enabled:  false,
		Settings: map[string]interface{}{
			"guild_id":   "123456789",
			"channel_id": "987654321",
		},
	}

	if discordConfig.Type != "discord" {
		t.Error("Discord destination type mismatch")
	}

	if discordConfig.Settings["guild_id"] != "123456789" {
		t.Error("Discord guild_id setting mismatch")
	}

	// Test Generic destination
	genericConfig := DestinationConfig{
		Name:     "generic",
		Type:     "generic",
		Host:     "example.com",
		Port:     9999,
		Protocol: "udp",
		Format:   "ogg",
		Enabled:  false,
	}

	if genericConfig.Format != "ogg" {
		t.Error("Generic destination format mismatch")
	}
}

// TestAudioConfig tests audio configuration validation
func TestAudioConfig(t *testing.T) {
	audioConfig := AudioConfig{
		EnableConversion: true,
		OutputFormat:     "opus",
		Bitrate:          64,
		SampleRate:       8000,
		Channels:         1,
	}

	if !audioConfig.EnableConversion {
		t.Error("Expected audio conversion to be enabled")
	}

	if audioConfig.OutputFormat != "opus" {
		t.Error("Expected opus output format")
	}

	if audioConfig.Bitrate != 64 {
		t.Errorf("Expected bitrate 64, got %d", audioConfig.Bitrate)
	}

	if audioConfig.SampleRate != 8000 {
		t.Errorf("Expected sample rate 8000, got %d", audioConfig.SampleRate)
	}

	if audioConfig.Channels != 1 {
		t.Errorf("Expected 1 channel (mono), got %d", audioConfig.Channels)
	}
}

// BenchmarkConfigMarshaling benchmarks configuration marshaling performance
func BenchmarkConfigMarshaling(b *testing.B) {
	config := defaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfigUnmarshaling benchmarks configuration unmarshaling performance
func BenchmarkConfigUnmarshaling(b *testing.B) {
	config := defaultConfig()
	data, err := json.Marshal(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var loadedConfig Config
		err := json.Unmarshal(data, &loadedConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}
