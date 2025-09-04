// USRP Bridge Utility - Bridges AllStarLink nodes with destination services via FFmpeg Opus conversion
//
// Architecture: AllStarLink Node <--USRP--> USRP Bridge <--Opus--> Destination Service
//
// The bridge receives USRP packets from AllStarLink nodes, converts audio to Opus format
// using FFmpeg, and forwards to configured destination services (Discord, WhoTalkie, etc.)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbehnke/usrp-go/pkg/audio"
	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// Config holds the bridge configuration
type Config struct {
	// USRP listening configuration
	USRPListenPort int    `json:"usrp_listen_port"`
	USRPListenAddr string `json:"usrp_listen_addr"`

	// AllStarLink return configuration
	AllStarHost string `json:"allstar_host"`
	AllStarPort int    `json:"allstar_port"`

	// Destination services
	Destinations []DestinationConfig `json:"destinations"`

	// Audio conversion settings
	AudioConfig AudioConfig `json:"audio_config"`

	// Logging and monitoring
	LogLevel    string `json:"log_level"`
	MetricsPort int    `json:"metrics_port"`

	// Amateur radio settings
	StationCall string `json:"station_call"`
	TalkGroup   uint32 `json:"talk_group"`
}

// DestinationConfig defines a destination service configuration
type DestinationConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "discord", "whotalkie", "generic"
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "udp", "tcp", "websocket"
	Format   string `json:"format"`   // "opus", "ogg", "raw"
	Enabled  bool   `json:"enabled"`

	// Service-specific settings
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// AudioConfig defines audio processing settings
type AudioConfig struct {
	EnableConversion bool   `json:"enable_conversion"`
	OutputFormat     string `json:"output_format"` // "opus", "ogg"
	Bitrate          int    `json:"bitrate"`       // kbps
	SampleRate       int    `json:"sample_rate"`   // Hz
	Channels         int    `json:"channels"`
}

// Bridge represents the main USRP bridge
type Bridge struct {
	config    *Config
	converter audio.Converter

	// Network connections
	usrpConn     *net.UDPConn
	allstarConn  *net.UDPConn
	destinations map[string]*net.UDPConn

	// Metrics and monitoring
	stats *BridgeStats

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
}

// BridgeStats tracks bridge performance metrics
type BridgeStats struct {
	USRPPacketsReceived  uint64 `json:"usrp_packets_received"`
	USRPPacketsSent      uint64 `json:"usrp_packets_sent"`
	OpusPacketsGenerated uint64 `json:"opus_packets_generated"`
	OpusPacketsForwarded uint64 `json:"opus_packets_forwarded"`
	ConversionErrors     uint64 `json:"conversion_errors"`
	NetworkErrors        uint64 `json:"network_errors"`
	ActiveTransmissions  uint64 `json:"active_transmissions"`
	LastActivityTime     int64  `json:"last_activity_time"`
	BytesReceived        uint64 `json:"bytes_received"`
	BytesSent            uint64 `json:"bytes_sent"`
}

// Default configuration
func defaultConfig() *Config {
	return &Config{
		USRPListenPort: 12345,
		USRPListenAddr: "0.0.0.0",
		AllStarHost:    "127.0.0.1",
		AllStarPort:    12346,
		Destinations: []DestinationConfig{
			{
				Name:     "whotalkie",
				Type:     "whotalkie",
				Host:     "127.0.0.1",
				Port:     8080,
				Protocol: "udp",
				Format:   "opus",
				Enabled:  true,
			},
		},
		AudioConfig: AudioConfig{
			EnableConversion: true,
			OutputFormat:     "opus",
			Bitrate:          64,
			SampleRate:       8000,
			Channels:         1,
		},
		LogLevel:    "info",
		MetricsPort: 9090,
		StationCall: "N0CALL",
		TalkGroup:   0,
	}
}

func main() {
	var (
		configFile = flag.String("config", "", "Configuration file path (JSON)")
		genConfig  = flag.Bool("generate-config", false, "Generate sample configuration file")
		listenPort = flag.Int("listen-port", 12345, "USRP listen port")
		destHost   = flag.String("dest-host", "127.0.0.1", "Destination host")
		destPort   = flag.Int("dest-port", 8080, "Destination port")
		callsign   = flag.String("callsign", "N0CALL", "Amateur radio callsign")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Generate sample configuration if requested
	if *genConfig {
		generateSampleConfig()
		return
	}

	// Load configuration
	var config *Config
	if *configFile != "" {
		var err error
		config, err = loadConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		// Use command line arguments for simple configuration
		config = defaultConfig()
		config.USRPListenPort = *listenPort
		config.StationCall = *callsign
		if len(config.Destinations) > 0 {
			config.Destinations[0].Host = *destHost
			config.Destinations[0].Port = *destPort
		}
	}

	// Setup logging
	if *verbose || config.LogLevel == "debug" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	fmt.Println("🔗 USRP Bridge Utility")
	fmt.Println("======================")
	fmt.Printf("📻 Station: %s\n", config.StationCall)
	fmt.Printf("🎧 USRP Listen: %s:%d\n", config.USRPListenAddr, config.USRPListenPort)
	fmt.Printf("📡 AllStarLink: %s:%d\n", config.AllStarHost, config.AllStarPort)
	fmt.Printf("🎵 Audio: %s @ %d kbps\n", config.AudioConfig.OutputFormat, config.AudioConfig.Bitrate)

	for i, dest := range config.Destinations {
		if dest.Enabled {
			fmt.Printf("🎯 Destination %d: %s (%s://%s:%d, format: %s)\n",
				i+1, dest.Name, dest.Protocol, dest.Host, dest.Port, dest.Format)
		}
	}

	// Create and start bridge
	bridge, err := NewBridge(config)
	if err != nil {
		log.Fatalf("Failed to create bridge: %v", err)
	}

	if err := bridge.Start(); err != nil {
		log.Fatalf("Failed to start bridge: %v", err)
	}
	defer func() {
		if err := bridge.Stop(); err != nil {
			log.Printf("Error stopping bridge: %v", err)
		}
	}()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("🚀 Bridge started successfully!")
	fmt.Println("📊 Send SIGUSR1 for statistics")
	fmt.Println("Press Ctrl+C to stop...")

	// Handle signals
	signal.Notify(sigChan, syscall.SIGUSR1)

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGUSR1:
			bridge.PrintStats()
		case syscall.SIGINT, syscall.SIGTERM:
			fmt.Println("\n🛑 Shutting down bridge...")
			return
		}
	}
}

// NewBridge creates a new USRP bridge instance
func NewBridge(config *Config) (*Bridge, error) {
	ctx, cancel := context.WithCancel(context.Background())

	bridge := &Bridge{
		config:       config,
		destinations: make(map[string]*net.UDPConn),
		stats:        &BridgeStats{},
		ctx:          ctx,
		cancel:       cancel,
	}

	// Create audio converter if enabled
	if config.AudioConfig.EnableConversion {
		var err error
		switch config.AudioConfig.OutputFormat {
		case "opus":
			bridge.converter, err = audio.NewOpusConverter()
		case "ogg":
			bridge.converter, err = audio.NewOggOpusConverter()
		default:
			return nil, fmt.Errorf("unsupported audio format: %s", config.AudioConfig.OutputFormat)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create audio converter: %w", err)
		}
	}

	return bridge, nil
}

// Start initializes and starts the bridge
func (b *Bridge) Start() error {
	// Setup USRP listener
	usrpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d",
		b.config.USRPListenAddr, b.config.USRPListenPort))
	if err != nil {
		return fmt.Errorf("failed to resolve USRP address: %w", err)
	}

	b.usrpConn, err = net.ListenUDP("udp", usrpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on USRP port: %w", err)
	}

	// Setup AllStarLink connection
	allstarAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d",
		b.config.AllStarHost, b.config.AllStarPort))
	if err != nil {
		return fmt.Errorf("failed to resolve AllStarLink address: %w", err)
	}

	b.allstarConn, err = net.DialUDP("udp", nil, allstarAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to AllStarLink: %w", err)
	}

	// Setup destination connections
	for i, dest := range b.config.Destinations {
		if !dest.Enabled {
			continue
		}

		destAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dest.Host, dest.Port))
		if err != nil {
			log.Printf("Warning: Failed to resolve destination %s: %v", dest.Name, err)
			continue
		}

		conn, err := net.DialUDP("udp", nil, destAddr)
		if err != nil {
			log.Printf("Warning: Failed to connect to destination %s: %v", dest.Name, err)
			continue
		}

		b.destinations[dest.Name] = conn
		log.Printf("✅ Connected to destination: %s (%s:%d)", dest.Name, dest.Host, dest.Port)
		_ = i // Avoid unused variable
	}

	// Start processing goroutines
	go b.processUSRPPackets()

	return nil
}

// Stop gracefully shuts down the bridge
func (b *Bridge) Stop() error {
	b.cancel()

	if b.usrpConn != nil {
		b.usrpConn.Close()
	}

	if b.allstarConn != nil {
		b.allstarConn.Close()
	}

	for name, conn := range b.destinations {
		if conn != nil {
			conn.Close()
		}
		delete(b.destinations, name)
	}

	if b.converter != nil {
		b.converter.Close()
	}

	return nil
}

// processUSRPPackets handles incoming USRP packets from AllStarLink
func (b *Bridge) processUSRPPackets() {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			// Set read timeout
			if err := b.usrpConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				log.Printf("Failed to set USRP read deadline: %v", err)
				continue
			}

			n, addr, err := b.usrpConn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				b.stats.NetworkErrors++
				log.Printf("Error reading USRP packet: %v", err)
				continue
			}

			b.stats.USRPPacketsReceived++
			b.stats.BytesReceived += uint64(n)
			b.stats.LastActivityTime = time.Now().Unix()

			// Parse USRP packet
			voiceMsg := &usrp.VoiceMessage{}
			if err := voiceMsg.Unmarshal(buffer[:n]); err != nil {
				log.Printf("Failed to unmarshal USRP packet: %v", err)
				continue
			}

			// Process the packet
			if err := b.processVoicePacket(voiceMsg, addr); err != nil {
				log.Printf("Failed to process voice packet: %v", err)
				b.stats.ConversionErrors++
			}
		}
	}
}

// processVoicePacket processes a single USRP voice packet
func (b *Bridge) processVoicePacket(voiceMsg *usrp.VoiceMessage, sourceAddr *net.UDPAddr) error {
	// Update station call if configured
	if b.config.StationCall != "N0CALL" && b.config.StationCall != "" {
		// Note: In a full implementation, you might want to add TLV metadata
		// with the station callsign for amateur radio compliance
		log.Printf("Processing voice packet from station: %s", b.config.StationCall)
	}

	// Forward to destination services if audio conversion is enabled
	if b.config.AudioConfig.EnableConversion && b.converter != nil && voiceMsg.Header.IsPTT() {
		if err := b.forwardToDestinations(voiceMsg); err != nil {
			return fmt.Errorf("failed to forward to destinations: %w", err)
		}
	}

	// Echo back to AllStarLink (for testing or relay scenarios)
	if b.allstarConn != nil {
		data, err := voiceMsg.Marshal()
		if err != nil {
			return fmt.Errorf("failed to marshal voice packet: %w", err)
		}

		if _, err := b.allstarConn.Write(data); err != nil {
			b.stats.NetworkErrors++
			return fmt.Errorf("failed to send to AllStarLink: %w", err)
		}

		b.stats.USRPPacketsSent++
		b.stats.BytesSent += uint64(len(data))
	}

	return nil
}

// forwardToDestinations converts and forwards audio to destination services
func (b *Bridge) forwardToDestinations(voiceMsg *usrp.VoiceMessage) error {
	if b.converter == nil {
		return nil
	}

	// Convert USRP to target format
	audioData, err := b.converter.USRPToFormat(voiceMsg)
	if err != nil {
		return fmt.Errorf("audio conversion failed: %w", err)
	}

	if len(audioData) == 0 {
		return nil // No audio data produced
	}

	b.stats.OpusPacketsGenerated++

	// Forward to all enabled destinations
	for _, destConfig := range b.config.Destinations {
		if !destConfig.Enabled {
			continue
		}

		conn, exists := b.destinations[destConfig.Name]
		if !exists {
			continue
		}

		// Format the data based on destination requirements
		finalData := audioData

		// Add any destination-specific formatting here
		switch destConfig.Type {
		case "whotalkie":
			// WhoTalkie might expect specific packet format
			finalData = b.formatForWhoTalkie(audioData, voiceMsg)
		case "discord":
			// Discord format (handled by Discord bridge)
			finalData = audioData
		case "generic":
			// Generic Opus/Ogg format
			finalData = audioData
		}

		// Send to destination
		if _, err := conn.Write(finalData); err != nil {
			log.Printf("Failed to send to destination %s: %v", destConfig.Name, err)
			b.stats.NetworkErrors++
			continue
		}

		b.stats.OpusPacketsForwarded++
		b.stats.BytesSent += uint64(len(finalData))
	}

	return nil
}

// formatForWhoTalkie formats audio data for WhoTalkie service
func (b *Bridge) formatForWhoTalkie(audioData []byte, voiceMsg *usrp.VoiceMessage) []byte {
	// WhoTalkie might expect a specific packet format
	// This is a placeholder - adjust based on actual WhoTalkie protocol

	// For now, just return the raw Opus data
	// In a real implementation, you might wrap it in a JSON message or
	// add headers with metadata like callsign, PTT state, etc.

	return audioData
}

// PrintStats displays current bridge statistics
func (b *Bridge) PrintStats() {
	fmt.Println("\n📊 Bridge Statistics")
	fmt.Println("==================")
	fmt.Printf("USRP Packets: %d received, %d sent\n",
		b.stats.USRPPacketsReceived, b.stats.USRPPacketsSent)
	fmt.Printf("Opus Packets: %d generated, %d forwarded\n",
		b.stats.OpusPacketsGenerated, b.stats.OpusPacketsForwarded)
	fmt.Printf("Errors: %d conversion, %d network\n",
		b.stats.ConversionErrors, b.stats.NetworkErrors)
	fmt.Printf("Traffic: %d bytes received, %d bytes sent\n",
		b.stats.BytesReceived, b.stats.BytesSent)
	fmt.Printf("Last Activity: %s\n", time.Unix(b.stats.LastActivityTime, 0).Format(time.RFC3339))
	fmt.Println()
}

// Configuration file helpers

// loadConfig loads configuration from a JSON file
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// generateSampleConfig creates a sample configuration file
func generateSampleConfig() {
	config := defaultConfig()

	// Add more destination examples
	config.Destinations = append(config.Destinations,
		DestinationConfig{
			Name:     "discord-bot",
			Type:     "discord",
			Host:     "127.0.0.1",
			Port:     8081,
			Protocol: "udp",
			Format:   "opus",
			Enabled:  false,
			Settings: map[string]interface{}{
				"guild_id":   "your_guild_id",
				"channel_id": "your_channel_id",
			},
		},
		DestinationConfig{
			Name:     "backup-server",
			Type:     "generic",
			Host:     "backup.example.com",
			Port:     9999,
			Protocol: "udp",
			Format:   "ogg",
			Enabled:  false,
		})

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	filename := "usrp-bridge.json"
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}

	fmt.Printf("✅ Sample configuration written to %s\n", filename)
	fmt.Println("\nEdit the configuration file and run:")
	fmt.Printf("  ./usrp-bridge -config %s\n", filename)
}
