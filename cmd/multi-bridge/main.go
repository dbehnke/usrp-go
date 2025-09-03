// Multi-Service Audio Bridge - Routes audio between AllStarLink, WhoTalkie, and Discord
//
// Architecture: 
//   AllStarLink ←→ Multi-Bridge ←→ WhoTalkie
//                      ↕
//                  Discord Voice
//
// All services can communicate with each other through the central router
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dbehnke/usrp-go/pkg/audio"
	"github.com/dbehnke/usrp-go/pkg/discord"
	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// MultiServiceConfig holds configuration for the multi-service bridge
type MultiServiceConfig struct {
	// AllStarLink USRP settings
	USRPListenPort int    `json:"usrp_listen_port"`
	USRPListenAddr string `json:"usrp_listen_addr"`
	
	// WhoTalkie settings
	WhoTalkie struct {
		Enabled bool   `json:"enabled"`
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Format  string `json:"format"` // "opus", "ogg"
	} `json:"whotalkie"`
	
	// Discord settings
	Discord struct {
		Enabled   bool   `json:"enabled"`
		Token     string `json:"token"`
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
	} `json:"discord"`
	
	// Audio processing settings
	AudioConfig struct {
		EnableConversion bool   `json:"enable_conversion"`
		OutputFormat     string `json:"output_format"`
		Bitrate          int    `json:"bitrate"`
		SampleRate       int    `json:"sample_rate"`
		Channels         int    `json:"channels"`
	} `json:"audio_config"`
	
	// Amateur radio settings
	StationCall string `json:"station_call"`
	TalkGroup   uint32 `json:"talk_group"`
	
	// Routing settings
	Routing struct {
		PreventLoops     bool `json:"prevent_loops"`     // Prevent audio loops
		RouteAllToAll    bool `json:"route_all_to_all"`  // Route between all services
		USRPToServices   bool `json:"usrp_to_services"`  // AllStarLink → WhoTalkie/Discord
		ServicesToUSRP   bool `json:"services_to_usrp"`  // WhoTalkie/Discord → AllStarLink
		ServiceToService bool `json:"service_to_service"` // WhoTalkie ↔ Discord
	} `json:"routing"`
}

// AudioMessage represents audio from any source
type AudioMessage struct {
	Source    string    // "usrp", "whotalkie", "discord"
	Data      []byte    // Audio data (format varies by source)
	Format    string    // "pcm", "opus", "ogg"
	Timestamp time.Time // When received
	PTTActive bool      // PTT state
	CallSign  string    // Source callsign (if available)
	Sequence  uint32    // Sequence number
}

// MultiServiceBridge routes audio between multiple services
type MultiServiceBridge struct {
	config *MultiServiceConfig
	
	// Audio processing
	converter audio.Converter
	
	// Service integrations
	discordBridge *discord.Bridge
	whoTalkieConn interface{} // Placeholder for WhoTalkie connection
	
	// Audio routing
	audioRouter chan *AudioMessage
	activeTransmissions map[string]time.Time // Track active transmissions by source
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex
	
	// Statistics
	stats struct {
		USRPMessages      uint64
		WhoTalkieMessages uint64
		DiscordMessages   uint64
		RoutedMessages    uint64
		DroppedMessages   uint64 // Due to loops or conflicts
	}
}

// DefaultMultiServiceConfig returns default configuration
func DefaultMultiServiceConfig() *MultiServiceConfig {
	config := &MultiServiceConfig{
		USRPListenPort: 12345,
		USRPListenAddr: "0.0.0.0",
		StationCall:    "N0CALL",
		TalkGroup:      0,
	}
	
	// WhoTalkie defaults
	config.WhoTalkie.Enabled = true
	config.WhoTalkie.Host = "127.0.0.1"
	config.WhoTalkie.Port = 8080
	config.WhoTalkie.Format = "opus"
	
	// Discord defaults  
	config.Discord.Enabled = false // Requires token
	config.Discord.GuildID = ""
	config.Discord.ChannelID = ""
	
	// Audio defaults
	config.AudioConfig.EnableConversion = true
	config.AudioConfig.OutputFormat = "opus"
	config.AudioConfig.Bitrate = 64
	config.AudioConfig.SampleRate = 8000
	config.AudioConfig.Channels = 1
	
	// Routing defaults
	config.Routing.PreventLoops = true
	config.Routing.RouteAllToAll = true
	config.Routing.USRPToServices = true
	config.Routing.ServicesToUSRP = true  
	config.Routing.ServiceToService = true
	
	return config
}

func main() {
	var (
		configFile = flag.String("config", "", "Configuration file path (JSON)")
		genConfig  = flag.Bool("generate-config", false, "Generate sample configuration")
		callsign   = flag.String("callsign", "N0CALL", "Amateur radio callsign")
	)
	flag.Parse()

	if *genConfig {
		generateSampleConfig()
		return
	}

	// Load configuration
	var config *MultiServiceConfig
	if *configFile != "" {
		var err error
		config, err = loadConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		config = DefaultMultiServiceConfig()
		config.StationCall = *callsign
	}

	fmt.Println("🌐 Multi-Service Audio Bridge")
	fmt.Println("============================")
	fmt.Printf("📻 Station: %s\n", config.StationCall)
	fmt.Printf("🎧 USRP Listen: %s:%d\n", config.USRPListenAddr, config.USRPListenPort)
	
	if config.WhoTalkie.Enabled {
		fmt.Printf("🎯 WhoTalkie: %s:%d (%s)\n", config.WhoTalkie.Host, config.WhoTalkie.Port, config.WhoTalkie.Format)
	}
	
	if config.Discord.Enabled {
		fmt.Printf("🎮 Discord: Guild %s, Channel %s\n", config.Discord.GuildID, config.Discord.ChannelID)
	}
	
	fmt.Printf("🔄 Routing: All-to-All=%v, Prevent-Loops=%v\n", 
		config.Routing.RouteAllToAll, config.Routing.PreventLoops)

	// Create and start bridge
	bridge, err := NewMultiServiceBridge(config)
	if err != nil {
		log.Fatalf("Failed to create bridge: %v", err)
	}

	if err := bridge.Start(); err != nil {
		log.Fatalf("Failed to start bridge: %v", err)
	}
	defer bridge.Stop()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	fmt.Println("🚀 Multi-service bridge started!")
	fmt.Println("📊 Send SIGUSR1 for statistics")
	fmt.Println("Press Ctrl+C to stop...")

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGUSR1:
			bridge.PrintStats()
		case syscall.SIGINT, syscall.SIGTERM:
			fmt.Println("\n🛑 Shutting down multi-service bridge...")
			return
		}
	}
}

// NewMultiServiceBridge creates a new multi-service bridge
func NewMultiServiceBridge(config *MultiServiceConfig) (*MultiServiceBridge, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	bridge := &MultiServiceBridge{
		config:              config,
		audioRouter:         make(chan *AudioMessage, 1000), // Large buffer for routing
		activeTransmissions: make(map[string]time.Time),
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Create audio converter
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

	// Initialize Discord if enabled
	if config.Discord.Enabled {
		discordConfig := discord.DefaultBridgeConfig()
		discordConfig.DiscordToken = config.Discord.Token
		discordConfig.DiscordGuild = config.Discord.GuildID
		discordConfig.DiscordChannel = config.Discord.ChannelID
		discordConfig.CallSign = config.StationCall

		var err error
		bridge.discordBridge, err = discord.NewBridge(discordConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Discord bridge: %w", err)
		}
	}

	return bridge, nil
}

// Start starts the multi-service bridge
func (b *MultiServiceBridge) Start() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Start Discord bridge if enabled
	if b.discordBridge != nil {
		if err := b.discordBridge.Start(); err != nil {
			return fmt.Errorf("failed to start Discord bridge: %w", err)
		}
	}

	// Start audio router
	go b.audioRouter_worker()
	
	// Start service workers
	go b.usrpWorker()
	if b.config.WhoTalkie.Enabled {
		go b.whoTalkieWorker()
	}
	if b.discordBridge != nil {
		go b.discordWorker()
	}

	return nil
}

// Stop stops the multi-service bridge
func (b *MultiServiceBridge) Stop() error {
	b.cancel()
	
	if b.discordBridge != nil {
		b.discordBridge.Stop()
	}
	
	if b.converter != nil {
		b.converter.Close()
	}
	
	return nil
}

// audioRouter_worker routes audio between all services
func (b *MultiServiceBridge) audioRouter_worker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case msg := <-b.audioRouter:
			b.routeAudioMessage(msg)
		}
	}
}

// routeAudioMessage routes a single audio message to appropriate destinations
func (b *MultiServiceBridge) routeAudioMessage(msg *AudioMessage) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Update active transmissions
	b.activeTransmissions[msg.Source] = msg.Timestamp

	// Prevent loops - don't route back to source
	if b.config.Routing.PreventLoops {
		// Clean up old transmissions (older than 5 seconds)
		cutoff := time.Now().Add(-5 * time.Second)
		for source, timestamp := range b.activeTransmissions {
			if timestamp.Before(cutoff) {
				delete(b.activeTransmissions, source)
			}
		}
		
		// If multiple sources are active, prioritize based on rules
		if len(b.activeTransmissions) > 1 {
			// For now, drop the message to prevent conflicts
			// In production, you might implement priority rules
			b.stats.DroppedMessages++
			return
		}
	}

	// Route to destinations based on source and configuration
	destinations := b.getRoutingDestinations(msg.Source)
	
	for _, dest := range destinations {
		switch dest {
		case "usrp":
			b.sendToUSRP(msg)
		case "whotalkie":
			b.sendToWhoTalkie(msg)
		case "discord":
			b.sendToDiscord(msg)
		}
	}
	
	b.stats.RoutedMessages++
}

// getRoutingDestinations determines where to route audio based on source
func (b *MultiServiceBridge) getRoutingDestinations(source string) []string {
	var destinations []string
	
	switch source {
	case "usrp":
		if b.config.Routing.USRPToServices || b.config.Routing.RouteAllToAll {
			if b.config.WhoTalkie.Enabled {
				destinations = append(destinations, "whotalkie")
			}
			if b.config.Discord.Enabled {
				destinations = append(destinations, "discord")
			}
		}
	case "whotalkie":
		if b.config.Routing.ServicesToUSRP || b.config.Routing.RouteAllToAll {
			destinations = append(destinations, "usrp")
		}
		if b.config.Routing.ServiceToService || b.config.Routing.RouteAllToAll {
			if b.config.Discord.Enabled {
				destinations = append(destinations, "discord")
			}
		}
	case "discord":
		if b.config.Routing.ServicesToUSRP || b.config.Routing.RouteAllToAll {
			destinations = append(destinations, "usrp")
		}
		if b.config.Routing.ServiceToService || b.config.Routing.RouteAllToAll {
			if b.config.WhoTalkie.Enabled {
				destinations = append(destinations, "whotalkie")
			}
		}
	}
	
	return destinations
}

// Worker functions for each service (simplified implementations)

func (b *MultiServiceBridge) usrpWorker() {
	// Implementation for USRP packet handling
	// This would listen for USRP packets and convert to AudioMessage
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			// Placeholder - implement USRP packet reception
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (b *MultiServiceBridge) whoTalkieWorker() {
	// Implementation for WhoTalkie integration
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			// Placeholder - implement WhoTalkie packet handling
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (b *MultiServiceBridge) discordWorker() {
	// Implementation for Discord integration
	for {
		select {
		case <-b.ctx.Done():
			return
		case usrpPacket, ok := <-b.discordBridge.GetUSRPPacket():
			if !ok {
				continue
			}
			
			// Convert Discord audio to AudioMessage
			msg := &AudioMessage{
				Source:    "discord",
				Data:      nil, // Convert from usrpPacket
				Format:    "pcm",
				Timestamp: time.Now(),
				PTTActive: usrpPacket.Header.IsPTT(),
				CallSign:  b.config.StationCall,
				Sequence:  usrpPacket.Header.Seq,
			}
			
			b.audioRouter <- msg
			b.stats.DiscordMessages++
		}
	}
}

// Send functions for each destination
func (b *MultiServiceBridge) sendToUSRP(msg *AudioMessage) {
	// Convert AudioMessage to USRP packet and send
}

func (b *MultiServiceBridge) sendToWhoTalkie(msg *AudioMessage) {
	// Convert AudioMessage to WhoTalkie format and send
}

func (b *MultiServiceBridge) sendToDiscord(msg *AudioMessage) {
	// Convert AudioMessage to Discord format and send via bridge
}

// PrintStats displays current bridge statistics
func (b *MultiServiceBridge) PrintStats() {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	fmt.Println("\n📊 Multi-Service Bridge Statistics")
	fmt.Println("==================================")
	fmt.Printf("📡 USRP Messages: %d\n", b.stats.USRPMessages)
	fmt.Printf("🎯 WhoTalkie Messages: %d\n", b.stats.WhoTalkieMessages)  
	fmt.Printf("🎮 Discord Messages: %d\n", b.stats.DiscordMessages)
	fmt.Printf("🔄 Routed Messages: %d\n", b.stats.RoutedMessages)
	fmt.Printf("🚫 Dropped Messages: %d (loop prevention)\n", b.stats.DroppedMessages)
	fmt.Printf("📻 Active Transmissions: %d\n", len(b.activeTransmissions))
	
	if len(b.activeTransmissions) > 0 {
		fmt.Println("Active sources:")
		for source, timestamp := range b.activeTransmissions {
			fmt.Printf("  - %s (since %s)\n", source, timestamp.Format("15:04:05"))
		}
	}
	fmt.Println()
}

// Configuration helpers
func generateSampleConfig() {
	config := DefaultMultiServiceConfig()
	config.Discord.Token = "your_discord_token"
	config.Discord.GuildID = "your_guild_id"
	config.Discord.ChannelID = "your_channel_id"
	config.StationCall = "W1AW"

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}

	filename := "multi-bridge.json"
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}

	fmt.Printf("✅ Sample configuration written to %s\n", filename)
	fmt.Println("\nFeatures enabled in sample config:")
	fmt.Println("📡 AllStarLink USRP ←→ WhoTalkie ←→ Discord")
	fmt.Println("🔄 Full multi-directional routing")
	fmt.Println("🚫 Loop prevention enabled")
	fmt.Printf("\nEdit %s and run:\n", filename)
	fmt.Printf("  ./multi-bridge -config %s\n", filename)
}

func loadConfig(filename string) (*MultiServiceConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config MultiServiceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}