// Discord bridge example - connects amateur radio USRP to Discord voice channels
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dbehnke/usrp-go/pkg/discord"
	"github.com/dbehnke/usrp-go/pkg/usrp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Discord Bridge Demo - Amateur Radio <-> Discord Voice")
		fmt.Println("====================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/examples/discord_bridge.go bridge     # Run USRP-Discord bridge")
		fmt.Println("  go run cmd/examples/discord_bridge.go test       # Test Discord bot connection")
		fmt.Println("  go run cmd/examples/discord_bridge.go server     # USRP packet server (for testing)")
		fmt.Println()
		fmt.Println("Environment Variables:")
		fmt.Println("  DISCORD_TOKEN     - Discord bot token (required)")
		fmt.Println("  DISCORD_GUILD     - Discord server (guild) ID")
		fmt.Println("  DISCORD_CHANNEL   - Discord voice channel ID")
		fmt.Println("  AMATEUR_CALLSIGN  - Amateur radio callsign")
		fmt.Println()
		fmt.Println("Requirements:")
		fmt.Println("  - Discord bot with voice permissions")
		fmt.Println("  - FFmpeg with libopus support")
		fmt.Println("  - Amateur radio license for USRP operation")
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "test":
		runDiscordTest()
	case "bridge":
		runDiscordBridge()
	case "server":
		runUSRPServer()
	default:
		log.Fatalf("Unknown mode: %s", mode)
	}
}

// runDiscordTest tests Discord bot connection
func runDiscordTest() {
	fmt.Println("🎮 Discord Bot Connection Test")
	fmt.Println("=============================")

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	// Create Discord bot configuration
	config := discord.DefaultBridgeConfig()
	config.DiscordToken = token
	config.DiscordGuild = os.Getenv("DISCORD_GUILD")
	config.DiscordChannel = os.Getenv("DISCORD_CHANNEL")

	fmt.Printf("Token: %s...\n", token[:10])
	fmt.Printf("Guild: %s\n", config.DiscordGuild)
	fmt.Printf("Channel: %s\n", config.DiscordChannel)

	// Create and test bot
	botConfig := discord.DefaultBotConfig()
	botConfig.Token = token
	botConfig.GuildID = config.DiscordGuild
	botConfig.ChannelID = config.DiscordChannel

	bot, err := discord.NewBot(botConfig)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := bot.Start(ctx); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	defer bot.Stop()

	fmt.Println("✅ Bot connected successfully!")

	// Test voice connection if guild and channel specified
	if config.DiscordGuild != "" && config.DiscordChannel != "" {
		fmt.Printf("🔊 Attempting to join voice channel...\n")
		if err := bot.JoinVoiceChannel(config.DiscordGuild, config.DiscordChannel); err != nil {
			log.Printf("❌ Could not join voice channel: %v", err)
		} else {
			fmt.Println("✅ Successfully joined voice channel!")
			time.Sleep(3 * time.Second)
			bot.LeaveVoiceChannel()
			fmt.Println("👋 Left voice channel")
		}
	}

	fmt.Println("🎉 Discord test completed successfully!")
}

// runDiscordBridge runs the main USRP-Discord bridge
func runDiscordBridge() {
	fmt.Println("📻 Amateur Radio Discord Bridge")
	fmt.Println("===============================")

	// Get configuration from environment
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	config := discord.DefaultBridgeConfig()
	config.DiscordToken = token
	config.DiscordGuild = os.Getenv("DISCORD_GUILD")
	config.DiscordChannel = os.Getenv("DISCORD_CHANNEL")
	config.CallSign = os.Getenv("AMATEUR_CALLSIGN")

	if config.CallSign == "" {
		config.CallSign = "N0CALL"
		fmt.Printf("⚠️  Using default callsign: %s (set AMATEUR_CALLSIGN)\n", config.CallSign)
	}

	fmt.Printf("🎮 Discord Guild: %s\n", config.DiscordGuild)
	fmt.Printf("🔊 Discord Channel: %s\n", config.DiscordChannel)
	fmt.Printf("📻 Amateur Callsign: %s\n", config.CallSign)

	// Create bridge
	bridge, err := discord.NewBridge(config)
	if err != nil {
		log.Fatalf("Failed to create bridge: %v", err)
	}

	// Start bridge
	if err := bridge.Start(); err != nil {
		log.Fatalf("Failed to start bridge: %v", err)
	}
	defer bridge.Stop()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup UDP listener for USRP packets
	usrpAddr, err := net.ResolveUDPAddr("udp", ":12345")
	if err != nil {
		log.Fatalf("Failed to resolve USRP address: %v", err)
	}

	usrpConn, err := net.ListenUDP("udp", usrpAddr)
	if err != nil {
		log.Fatalf("Failed to listen for USRP packets: %v", err)
	}
	defer usrpConn.Close()

	fmt.Printf("📡 Listening for USRP packets on %s\n", usrpAddr)
	fmt.Println("🎯 Send USRP voice packets to start bridging!")
	fmt.Println("Press Ctrl+C to stop...")

	// Main bridge loop
	go func() {
		buffer := make([]byte, 1024)
		packetCount := 0

		for {
			// Set read timeout
			usrpConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

			n, _, err := usrpConn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Error reading USRP packet: %v", err)
				continue
			}

			// Parse USRP packet
			voiceMsg := &usrp.VoiceMessage{}
			if err := voiceMsg.Unmarshal(buffer[:n]); err != nil {
				log.Printf("Failed to unmarshal USRP packet: %v", err)
				continue
			}

			packetCount++
			if voiceMsg.Header.IsPTT() {
				fmt.Printf("📡 RX Packet %d: USRP voice, PTT ON, seq=%d\n",
					packetCount, voiceMsg.Header.Seq)

				// Send to Discord bridge
				if err := bridge.SendUSRPPacket(voiceMsg); err != nil {
					log.Printf("Failed to send to Discord: %v", err)
				} else {
					fmt.Printf("🎮 → Discord: Sent voice packet\n")
				}
			}
		}
	}()

	// Monitor bridge status
	statusTicker := time.NewTicker(10 * time.Second)
	defer statusTicker.Stop()

	go func() {
		for {
			select {
			case <-statusTicker.C:
				if bridge.IsRunning() {
					status := "Disconnected"
					if bridge.IsDiscordConnected() {
						status = "Connected"
					}
					fmt.Printf("🔄 Bridge Status: Running, Discord: %s\n", status)
				}
			}
		}
	}()

	// Handle Discord to USRP packets
	go func() {
		for {
			if packet, ok := bridge.GetUSRPPacket(); ok {
				fmt.Printf("🎮 RX Discord: Converting to USRP, seq=%d\n", packet.Header.Seq)

				// In a real implementation, you would send this via UDP
				// to your amateur radio system
				data, err := packet.Marshal()
				if err != nil {
					log.Printf("Failed to marshal USRP packet: %v", err)
					continue
				}

				fmt.Printf("📡 → Amateur Radio: %d bytes (would send via UDP)\n", len(data))
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutting down bridge...")
}

// runUSRPServer generates test USRP packets for testing
func runUSRPServer() {
	fmt.Println("📡 USRP Test Packet Server")
	fmt.Println("=========================")

	// Connect to bridge
	conn, err := net.Dial("udp", "127.0.0.1:12345")
	if err != nil {
		log.Fatalf("Failed to connect to bridge: %v", err)
	}
	defer conn.Close()

	fmt.Println("📻 Generating test USRP voice packets...")
	fmt.Println("Press Ctrl+C to stop")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Generate test packets
	ticker := time.NewTicker(20 * time.Millisecond) // 20ms intervals
	defer ticker.Stop()

	seq := uint32(1)
	transmission := 0

	for {
		select {
		case <-sigChan:
			return

		case <-ticker.C:
			// Simulate PTT transmission (5 seconds on, 3 seconds off)
			pttActive := (time.Now().Unix() % 8) < 5

			voiceMsg := &usrp.VoiceMessage{
				Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, seq),
			}
			voiceMsg.Header.SetPTT(pttActive)
			voiceMsg.Header.TalkGroup = 12345

			if pttActive {
				// Generate test audio (simple pattern)
				for i := range voiceMsg.AudioData {
					// Generate a test tone pattern
					voiceMsg.AudioData[i] = int16((seq*100 + uint32(i)) % 10000)
				}

				if seq%50 == 1 { // Print every 1 second
					transmission++
					fmt.Printf("📡 TX %d: Sending voice packets (seq %d)\n", transmission, seq)
				}
			} else {
				// Fill with silence
				for i := range voiceMsg.AudioData {
					voiceMsg.AudioData[i] = 0
				}

				if seq%50 == 1 {
					fmt.Printf("⏸️  Silence period (seq %d)\n", seq)
				}
			}

			// Send packet
			data, err := voiceMsg.Marshal()
			if err != nil {
				log.Printf("Failed to marshal packet: %v", err)
				continue
			}

			if _, err := conn.Write(data); err != nil {
				log.Printf("Failed to send packet: %v", err)
				continue
			}

			seq++
		}
	}
}