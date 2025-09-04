// Audio bridge example showing real-time conversion between USRP and Opus/Ogg
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

	"github.com/dbehnke/usrp-go/pkg/audio"
	"github.com/dbehnke/usrp-go/pkg/usrp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Audio Bridge Demo - USRP <-> Opus/Ogg Conversion")
		fmt.Println("==============================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/examples/audio_bridge.go server     # Run as server (receives USRP, sends Opus)")
		fmt.Println("  go run cmd/examples/audio_bridge.go client     # Run as client (receives Opus, sends USRP)")
		fmt.Println("  go run cmd/examples/audio_bridge.go test       # Run conversion test")
		fmt.Println()
		fmt.Println("Requirements:")
		fmt.Println("  - FFmpeg with libopus support")
		fmt.Println("  - For server/client: run both in separate terminals")
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "test":
		runConversionTest()
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		log.Fatalf("Unknown mode: %s", mode)
	}
}

// runConversionTest demonstrates the conversion functionality
func runConversionTest() {
	fmt.Println("🎵 USRP Audio Conversion Test")
	fmt.Println("============================")

	// Test Opus conversion
	fmt.Println("\n--- Testing Opus Conversion ---")
	if err := testOpusConversion(); err != nil {
		log.Printf("❌ Opus test failed: %v", err)
	} else {
		fmt.Println("✅ Opus conversion test passed!")
	}

	// Test Ogg/Opus conversion
	fmt.Println("\n--- Testing Ogg/Opus Conversion ---")
	if err := testOggConversion(); err != nil {
		log.Printf("❌ Ogg test failed: %v", err)
	} else {
		fmt.Println("✅ Ogg/Opus conversion test passed!")
	}

	// Test streaming bridge
	fmt.Println("\n--- Testing Streaming Bridge ---")
	if err := testStreamingBridge(); err != nil {
		log.Printf("❌ Bridge test failed: %v", err)
	} else {
		fmt.Println("✅ Streaming bridge test passed!")
	}
}

func testOpusConversion() error {
	converter, err := audio.NewOpusConverter()
	if err != nil {
		return fmt.Errorf("failed to create Opus converter: %w", err)
	}
	defer converter.Close()

	// Create test USRP voice message with 440Hz tone
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 1234),
	}
	
	// Generate test pattern (20ms at 8kHz = 160 samples)
	for i := range voiceMsg.AudioData {
		// Simple linear ramp for testing (could be sine wave with math.Sin)
		voiceMsg.AudioData[i] = int16(i * 100) // Linear ramp for testing
	}

	fmt.Printf("📡 Original USRP: %d samples, PTT=%v\n", len(voiceMsg.AudioData), voiceMsg.Header.IsPTT())

	// Convert to Opus
	opusData, err := converter.USRPToFormat(voiceMsg)
	if err != nil {
		return fmt.Errorf("USRP to Opus failed: %w", err)
	}
	fmt.Printf("🎵 Converted to Opus: %d bytes (compression: %.1f%%)\n", 
		len(opusData), float64(len(opusData))*100.0/float64(len(voiceMsg.AudioData)*2))

	// Convert back to USRP
	usrpMessages, err := converter.FormatToUSRP(opusData)
	if err != nil {
		return fmt.Errorf("Opus to USRP failed: %w", err)
	}
	fmt.Printf("📡 Converted back: %d USRP messages\n", len(usrpMessages))

	return nil
}

func testOggConversion() error {
	converter, err := audio.NewOggOpusConverter()
	if err != nil {
		return fmt.Errorf("failed to create Ogg converter: %w", err)
	}
	defer converter.Close()

	// Create test voice message
	voiceMsg := &usrp.VoiceMessage{
		Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, 5678),
	}
	
	// Fill with test pattern
	for i := range voiceMsg.AudioData {
		voiceMsg.AudioData[i] = int16(1000 * (i % 20)) // Pattern for testing
	}

	fmt.Printf("📡 Original USRP: %d samples\n", len(voiceMsg.AudioData))

	// Convert to Ogg/Opus
	oggData, err := converter.USRPToFormat(voiceMsg)
	if err != nil {
		return fmt.Errorf("USRP to Ogg failed: %w", err)
	}
	fmt.Printf("🗂️  Converted to Ogg/Opus: %d bytes\n", len(oggData))

	return nil
}

func testStreamingBridge() error {
	converter, err := audio.NewOpusConverter()
	if err != nil {
		return fmt.Errorf("failed to create converter: %w", err)
	}

	bridge := audio.NewAudioBridge(converter)
	defer func() {
		if err := bridge.Stop(); err != nil {
			log.Printf("Error stopping bridge: %v", err)
		}
	}()

	if err := bridge.Start(); err != nil {
		return fmt.Errorf("failed to start bridge: %w", err)
	}

	fmt.Println("🌉 Testing streaming bridge...")

	// Create test transmission (5 packets)
	for i := 0; i < 5; i++ {
		voiceMsg := &usrp.VoiceMessage{
			Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, uint32(i+1)),
		}
		voiceMsg.Header.SetPTT(i < 4) // PTT on for first 4, off for last

		// Fill with test data
		for j := range voiceMsg.AudioData {
			voiceMsg.AudioData[j] = int16((i*1000 + j) % 30000)
		}

		// Send through bridge
		bridge.USRPIn <- voiceMsg

		// Read converted data
		select {
		case opusData := <-bridge.USRPToChan:
			fmt.Printf("  Packet %d: %d samples -> %d bytes Opus\n", i+1, len(voiceMsg.AudioData), len(opusData))
		case <-time.After(500 * time.Millisecond):
			fmt.Printf("  Packet %d: timeout\n", i+1)
		}
	}

	return nil
}

// runServer receives USRP packets and streams Opus data
func runServer() {
	fmt.Println("🖥️  Starting USRP -> Opus Server")
	fmt.Println("================================")

	// Create Opus converter
	converter, err := audio.NewOpusConverter()
	if err != nil {
		log.Fatalf("Failed to create Opus converter: %v", err)
	}
	defer converter.Close()

	// Listen for USRP packets
	usrpAddr, err := net.ResolveUDPAddr("udp", ":12345")
	if err != nil {
		log.Fatalf("Failed to resolve USRP address: %v", err)
	}

	usrpConn, err := net.ListenUDP("udp", usrpAddr)
	if err != nil {
		log.Fatalf("Failed to listen for USRP: %v", err)
	}
	defer usrpConn.Close()

	// Connect to Opus client
	opusAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:12346")
	if err != nil {
		log.Fatalf("Failed to resolve Opus address: %v", err)
	}

	opusConn, err := net.DialUDP("udp", nil, opusAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Opus client: %v", err)
	}
	defer opusConn.Close()

	fmt.Printf("📡 Listening for USRP packets on %s\n", usrpAddr)
	fmt.Printf("🎵 Sending Opus data to %s\n", opusAddr)
	fmt.Println("Press Ctrl+C to stop...")

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Stopping server...")
		cancel()
	}()

	buffer := make([]byte, 1024)
	packetCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set read timeout
			if err := usrpConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				log.Printf("Failed to set USRP read deadline: %v", err)
				continue
			}

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
			fmt.Printf("📦 Packet %d: USRP %d bytes, PTT=%v\n", 
				packetCount, n, voiceMsg.Header.IsPTT())

			// Convert to Opus
			opusData, err := converter.USRPToFormat(voiceMsg)
			if err != nil {
				log.Printf("Conversion failed: %v", err)
				continue
			}

			// Send Opus data
			if len(opusData) > 0 {
				if _, err := opusConn.Write(opusData); err != nil {
					log.Printf("Failed to send Opus data: %v", err)
				} else {
					fmt.Printf("🎵 Sent %d bytes Opus\n", len(opusData))
				}
			}
		}
	}
}

// runClient receives Opus data and sends USRP packets
func runClient() {
	fmt.Println("💻 Starting Opus -> USRP Client")
	fmt.Println("===============================")

	// Create Opus converter
	converter, err := audio.NewOpusConverter()
	if err != nil {
		log.Fatalf("Failed to create Opus converter: %v", err)
	}
	defer converter.Close()

	// Listen for Opus data
	opusAddr, err := net.ResolveUDPAddr("udp", ":12346")
	if err != nil {
		log.Fatalf("Failed to resolve Opus address: %v", err)
	}

	opusConn, err := net.ListenUDP("udp", opusAddr)
	if err != nil {
		log.Fatalf("Failed to listen for Opus: %v", err)
	}
	defer opusConn.Close()

	// Connect to USRP destination
	usrpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:12347")
	if err != nil {
		log.Fatalf("Failed to resolve USRP destination: %v", err)
	}

	usrpConn, err := net.DialUDP("udp", nil, usrpAddr)
	if err != nil {
		log.Fatalf("Failed to connect to USRP destination: %v", err)
	}
	defer usrpConn.Close()

	fmt.Printf("🎵 Listening for Opus data on %s\n", opusAddr)
	fmt.Printf("📡 Sending USRP packets to %s\n", usrpAddr)
	fmt.Println("Press Ctrl+C to stop...")

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Stopping client...")
		cancel()
	}()

	buffer := make([]byte, 4096)
	packetCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set read timeout
			if err := opusConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				log.Printf("Failed to set Opus read deadline: %v", err)
				continue
			}

			n, _, err := opusConn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Error reading Opus data: %v", err)
				continue
			}

			packetCount++
			fmt.Printf("🎵 Received %d bytes Opus data\n", n)

			// Convert to USRP
			usrpMessages, err := converter.FormatToUSRP(buffer[:n])
			if err != nil {
				log.Printf("Conversion failed: %v", err)
				continue
			}

			// Send USRP messages
			for i, voiceMsg := range usrpMessages {
				data, err := voiceMsg.Marshal()
				if err != nil {
					log.Printf("Failed to marshal USRP message %d: %v", i, err)
					continue
				}

				if _, err := usrpConn.Write(data); err != nil {
					log.Printf("Failed to send USRP message %d: %v", i, err)
				} else {
					fmt.Printf("📡 Sent USRP message %d: %d bytes\n", i+1, len(data))
				}
			}
		}
	}
}