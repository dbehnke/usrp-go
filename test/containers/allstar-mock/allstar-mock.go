// Mock AllStarLink Server for Integration Testing
//
// Simulates a real AllStarLink node for testing the Audio Router Hub
// Generates realistic USRP packets with various test patterns
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dbehnke/usrp-go/pkg/usrp"
)

type TestPattern string

const (
	PatternSilence    TestPattern = "silence"
	PatternSine440Hz  TestPattern = "sine_440hz"
	PatternSine1kHz   TestPattern = "sine_1khz"
	PatternWhiteNoise TestPattern = "white_noise"
	PatternDTMF       TestPattern = "dtmf_sequence"
	PatternVoice      TestPattern = "voice_sample"
	PatternSweep      TestPattern = "frequency_sweep"
)

type AllStarMock struct {
	nodeID     uint32
	callsign   string
	talkGroup  uint32
	listenPort int
	remoteAddr string
	remotePort int
	pattern    TestPattern

	// Network
	conn      *net.UDPConn
	remoteUDP *net.UDPAddr

	// Audio generation
	sampleRate  int
	frameSize   int
	sequenceNum uint32
	audioPhase  float64

	// Control
	running   bool
	pttActive bool
	mutex     sync.RWMutex

	// Statistics
	stats struct {
		packetsSent     uint64
		packetsReceived uint64
		bytesSent       uint64
		bytesReceived   uint64
		errors          uint64
		startTime       time.Time
	}
}

func NewAllStarMock(nodeID uint32, callsign string) *AllStarMock {
	return &AllStarMock{
		nodeID:     nodeID,
		callsign:   callsign,
		talkGroup:  1,
		listenPort: 34001,
		remoteAddr: "127.0.0.1",
		remotePort: 32001,
		pattern:    PatternSine440Hz,
		sampleRate: 8000,
		frameSize:  160, // 20ms at 8kHz
		running:    false,
		pttActive:  false,
	}
}

func (a *AllStarMock) Start() error {
	// Set up UDP listener
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", a.listenPort))
	if err != nil {
		return fmt.Errorf("failed to resolve listen address: %w", err)
	}

	a.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	// Set up remote address
	a.remoteUDP, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", a.remoteAddr, a.remotePort))
	if err != nil {
		return fmt.Errorf("failed to resolve remote address: %w", err)
	}

	a.running = true
	a.stats.startTime = time.Now()

	log.Printf("AllStar Mock Node %d (%s) started on port %d", a.nodeID, a.callsign, a.listenPort)
	log.Printf("Remote address: %s:%d", a.remoteAddr, a.remotePort)
	log.Printf("Test pattern: %s", a.pattern)

	// Start goroutines
	go a.receivePackets()
	go a.generateAudio()
	go a.statisticsReporter()

	return nil
}

func (a *AllStarMock) Stop() {
	a.mutex.Lock()
	a.running = false
	a.mutex.Unlock()

	if a.conn != nil {
		a.conn.Close()
	}

	log.Printf("AllStar Mock Node %d stopped", a.nodeID)
}

func (a *AllStarMock) receivePackets() {
	buffer := make([]byte, 1024)

	for a.isRunning() {
		a.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := a.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("UDP read error: %v", err)
			a.incrementErrors()
			continue
		}

		// Parse USRP packet
		if err := a.handleUSRPPacket(buffer[:n], remoteAddr); err != nil {
			log.Printf("USRP packet handling error: %v", err)
			a.incrementErrors()
		}

		a.mutex.Lock()
		a.stats.packetsReceived++
		a.stats.bytesReceived += uint64(n)
		a.mutex.Unlock()
	}
}

func (a *AllStarMock) handleUSRPPacket(data []byte, remoteAddr *net.UDPAddr) error {
	// Parse the packet to determine type
	if len(data) < 32 {
		return fmt.Errorf("packet too short")
	}

	// Check USRP magic
	if string(data[0:4]) != "USRP" {
		return fmt.Errorf("invalid USRP magic")
	}

	// Extract packet type
	packetType := (uint32(data[20]) << 24) | (uint32(data[21]) << 16) |
		(uint32(data[22]) << 8) | uint32(data[23])

	switch usrp.PacketType(packetType) {
	case usrp.USRP_TYPE_VOICE:
		msg := &usrp.VoiceMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return err
		}
		log.Printf("Received voice packet: PTT=%v, TalkGroup=%d",
			msg.Header.IsPTT(), msg.Header.TalkGroup)

	case usrp.USRP_TYPE_DTMF:
		msg := &usrp.DTMFMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return err
		}
		log.Printf("Received DTMF packet: Digit=%c", msg.Digit)

	case usrp.USRP_TYPE_PING:
		log.Printf("Received ping packet")

	default:
		log.Printf("Received unknown packet type: %d", packetType)
	}

	return nil
}

func (a *AllStarMock) generateAudio() {
	ticker := time.NewTicker(20 * time.Millisecond) // 20ms frames
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !a.isRunning() {
				return
			}

			// Generate audio frame
			audioData := a.generateAudioFrame()

			// Create USRP voice message
			voice := &usrp.VoiceMessage{
				Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, a.getNextSequence()),
			}

			// Set header fields
			voice.Header.TalkGroup = a.talkGroup
			voice.Header.SetPTT(a.isPTTActive())

			// Copy audio data
			for i := 0; i < len(voice.AudioData) && i < len(audioData); i++ {
				voice.AudioData[i] = audioData[i]
			}

			// Send packet
			if err := a.sendUSRPPacket(voice); err != nil {
				log.Printf("Failed to send voice packet: %v", err)
				a.incrementErrors()
			}

		}
	}
}

func (a *AllStarMock) generateAudioFrame() []int16 {
	audioData := make([]int16, a.frameSize)

	switch a.pattern {
	case PatternSilence:
		// Already zero-initialized

	case PatternSine440Hz:
		a.generateSineWave(audioData, 440.0)

	case PatternSine1kHz:
		a.generateSineWave(audioData, 1000.0)

	case PatternWhiteNoise:
		a.generateWhiteNoise(audioData)

	case PatternSweep:
		a.generateFrequencySweep(audioData)

	case PatternDTMF:
		a.generateDTMF(audioData)

	default:
		// Silence for unknown patterns
	}

	return audioData
}

func (a *AllStarMock) generateSineWave(audioData []int16, frequency float64) {
	amplitude := int16(8000) // -6dB from full scale

	for i := 0; i < len(audioData); i++ {
		sample := amplitude * int16(math.Sin(a.audioPhase))
		audioData[i] = sample

		a.audioPhase += 2.0 * math.Pi * frequency / float64(a.sampleRate)
		if a.audioPhase > 2.0*math.Pi {
			a.audioPhase -= 2.0 * math.Pi
		}
	}
}

func (a *AllStarMock) generateWhiteNoise(audioData []int16) {
	amplitude := int16(2000) // Lower amplitude for noise

	for i := 0; i < len(audioData); i++ {
		// Simple pseudo-random noise
		a.audioPhase = math.Mod(a.audioPhase*1103515245+12345, 4294967296)
		sample := amplitude * int16((int(a.audioPhase)%32768-16384)/16384)
		audioData[i] = sample
	}
}

func (a *AllStarMock) generateFrequencySweep(audioData []int16) {
	// Sweep from 300Hz to 3kHz over 10 seconds
	startFreq := 300.0
	endFreq := 3000.0

	elapsed := float64(time.Since(a.stats.startTime).Nanoseconds()) / 1e9
	progress := math.Mod(elapsed, 10.0) / 10.0 // Reset every 10 seconds

	currentFreq := startFreq + (endFreq-startFreq)*progress
	a.generateSineWave(audioData, currentFreq)
}

func (a *AllStarMock) generateDTMF(audioData []int16) {
	// Generate DTMF sequence: "1234567890*#"
	dtmfDigits := "1234567890*#"

	elapsed := int(time.Since(a.stats.startTime).Seconds())
	digitIndex := (elapsed / 2) % len(dtmfDigits)

	// DTMF frequencies (row, column)
	dtmfFreqs := map[rune][2]float64{
		'1': {697, 1209}, '2': {697, 1336}, '3': {697, 1477},
		'4': {770, 1209}, '5': {770, 1336}, '6': {770, 1477},
		'7': {852, 1209}, '8': {852, 1336}, '9': {852, 1477},
		'*': {941, 1209}, '0': {941, 1336}, '#': {941, 1477},
	}

	digit := rune(dtmfDigits[digitIndex])
	if freqs, ok := dtmfFreqs[digit]; ok {
		amplitude := int16(4000)

		for i := 0; i < len(audioData); i++ {
			sample1 := amplitude * int16(math.Sin(a.audioPhase)) / 2
			sample2 := amplitude * int16(math.Sin(a.audioPhase*freqs[1]/freqs[0])) / 2
			audioData[i] = sample1 + sample2

			a.audioPhase += 2.0 * math.Pi * freqs[0] / float64(a.sampleRate)
			if a.audioPhase > 2.0*math.Pi {
				a.audioPhase -= 2.0 * math.Pi
			}
		}
	}
}

func (a *AllStarMock) sendUSRPPacket(msg usrp.Message) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = a.conn.WriteToUDP(data, a.remoteUDP)
	if err != nil {
		return fmt.Errorf("failed to send UDP packet: %w", err)
	}

	a.mutex.Lock()
	a.stats.packetsSent++
	a.stats.bytesSent += uint64(len(data))
	a.mutex.Unlock()

	return nil
}

func (a *AllStarMock) statisticsReporter() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !a.isRunning() {
				return
			}
			a.printStatistics()
		}
	}
}

func (a *AllStarMock) printStatistics() {
	a.mutex.RLock()
	uptime := time.Since(a.stats.startTime)
	stats := a.stats
	a.mutex.RUnlock()

	log.Printf("=== AllStar Mock Node %d Statistics ===", a.nodeID)
	log.Printf("Uptime: %v", uptime.Round(time.Second))
	log.Printf("Packets Sent: %d", stats.packetsSent)
	log.Printf("Packets Received: %d", stats.packetsReceived)
	log.Printf("Bytes Sent: %d", stats.bytesSent)
	log.Printf("Bytes Received: %d", stats.bytesReceived)
	log.Printf("Errors: %d", stats.errors)

	if uptime.Seconds() > 0 {
		pps := float64(stats.packetsSent) / uptime.Seconds()
		log.Printf("Packets/sec: %.2f", pps)
	}
	log.Printf("=======================================")
}

// Helper methods
func (a *AllStarMock) isRunning() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.running
}

func (a *AllStarMock) isPTTActive() bool {
	// Simple PTT pattern: 3 seconds on, 2 seconds off
	elapsed := int(time.Since(a.stats.startTime).Seconds())
	cycle := elapsed % 5
	return cycle < 3
}

func (a *AllStarMock) getNextSequence() uint32 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.sequenceNum++
	return a.sequenceNum
}

func (a *AllStarMock) incrementErrors() {
	a.mutex.Lock()
	a.stats.errors++
	a.mutex.Unlock()
}

func main() {
	var (
		nodeID     = flag.Uint("node", 12345, "AllStarLink node ID")
		callsign   = flag.String("callsign", "W1AW", "Station callsign")
		listenPort = flag.Int("listen-port", 34001, "UDP listen port")
		remoteAddr = flag.String("remote-addr", "127.0.0.1", "Remote address")
		remotePort = flag.Int("remote-port", 32001, "Remote port")
		pattern    = flag.String("pattern", "sine_440hz", "Test pattern (silence, sine_440hz, sine_1khz, white_noise, dtmf_sequence, frequency_sweep)")
	)
	flag.Parse()

	mock := NewAllStarMock(uint32(*nodeID), *callsign)
	mock.listenPort = *listenPort
	mock.remoteAddr = *remoteAddr
	mock.remotePort = *remotePort
	mock.pattern = TestPattern(*pattern)

	if err := mock.Start(); err != nil {
		log.Fatalf("Failed to start mock: %v", err)
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")
	mock.Stop()
}
