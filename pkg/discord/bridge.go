// Discord-USRP audio bridge implementation
package discord

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dbehnke/usrp-go/pkg/audio"
	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// Bridge connects USRP packets with Discord voice channels
type Bridge struct {
	// Discord bot
	bot *Bot
	
	// Audio converter (USRP <-> PCM/Opus)
	converter audio.Converter
	
	// USRP channels
	USRPIn  chan *usrp.VoiceMessage   // USRP packets from amateur radio
	USRPOut chan *usrp.VoiceMessage   // USRP packets to amateur radio
	
	// Control
	running   bool
	mutex     sync.Mutex
	stopChan  chan bool
	ctx       context.Context
	cancel    context.CancelFunc
	
	// Configuration
	config *BridgeConfig
	
	// Audio resampling buffers
	discordBuffer []int16  // Buffer for Discord audio (48kHz)
	usrpBuffer    []int16  // Buffer for USRP audio (8kHz)
}

// BridgeConfig holds bridge configuration
type BridgeConfig struct {
	// Discord settings
	DiscordToken   string
	DiscordGuild   string
	DiscordChannel string
	
	// Audio settings
	EnableResampling bool          // Enable audio resampling between 8kHz and 48kHz
	PTTTimeout       time.Duration // PTT timeout for voice activation
	VoiceThreshold   int16         // Minimum audio level to trigger PTT
	
	// USRP settings
	CallSign         string        // Amateur radio callsign
	TalkGroup        uint32        // USRP talk group ID
	
	// Buffering
	BufferSize       int           // Channel buffer sizes
}

// DefaultBridgeConfig returns default bridge configuration
func DefaultBridgeConfig() *BridgeConfig {
	return &BridgeConfig{
		EnableResampling: true,
		PTTTimeout:       2 * time.Second,
		VoiceThreshold:   1000, // Adjust based on audio levels
		TalkGroup:        0,    // Default talk group
		BufferSize:       100,
	}
}

// NewBridge creates a new USRP-Discord bridge
func NewBridge(config *BridgeConfig) (*Bridge, error) {
	if config.DiscordToken == "" {
		return nil, fmt.Errorf("Discord token is required")
	}
	
	// Create Discord bot
	botConfig := DefaultBotConfig()
	botConfig.Token = config.DiscordToken
	botConfig.GuildID = config.DiscordGuild
	botConfig.ChannelID = config.DiscordChannel
	botConfig.BufferSize = config.BufferSize
	
	bot, err := NewBot(botConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord bot: %w", err)
	}
	
	// Create audio converter (USRP uses Opus for efficiency)
	converter, err := audio.NewOpusConverter()
	if err != nil {
		return nil, fmt.Errorf("failed to create audio converter: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	bridge := &Bridge{
		bot:           bot,
		converter:     converter,
		USRPIn:        make(chan *usrp.VoiceMessage, config.BufferSize),
		USRPOut:       make(chan *usrp.VoiceMessage, config.BufferSize),
		stopChan:      make(chan bool, 1),
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
		discordBuffer: make([]int16, 0, 4800), // ~100ms at 48kHz
		usrpBuffer:    make([]int16, 0, 800),  // ~100ms at 8kHz
	}
	
	return bridge, nil
}

// Start starts the USRP-Discord bridge
func (b *Bridge) Start() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if b.running {
		return fmt.Errorf("bridge is already running")
	}
	
	// Start Discord bot
	if err := b.bot.Start(b.ctx); err != nil {
		return fmt.Errorf("failed to start Discord bot: %w", err)
	}
	
	// Auto-join voice channel if specified
	if b.config.DiscordGuild != "" && b.config.DiscordChannel != "" {
		if err := b.bot.JoinVoiceChannel(b.config.DiscordGuild, b.config.DiscordChannel); err != nil {
			log.Printf("Warning: Could not auto-join voice channel: %v", err)
		}
	}
	
	b.running = true
	
	// Start bridge workers
	go b.usrpToDiscordWorker()
	go b.discordToUSRPWorker()
	
	log.Println("USRP-Discord bridge started")
	return nil
}

// Stop stops the bridge
func (b *Bridge) Stop() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if !b.running {
		return nil
	}
	
	b.running = false
	b.cancel()
	b.stopChan <- true
	
	// Stop Discord bot
	if err := b.bot.Stop(); err != nil {
		log.Printf("Error stopping Discord bot: %v", err)
	}
	
	// Stop audio converter
	if err := b.converter.Close(); err != nil {
		log.Printf("Error stopping audio converter: %v", err)
	}
	
	log.Println("USRP-Discord bridge stopped")
	return nil
}

// SendUSRPPacket sends a USRP packet to Discord
func (b *Bridge) SendUSRPPacket(packet *usrp.VoiceMessage) error {
	if !b.running {
		return fmt.Errorf("bridge is not running")
	}
	
	select {
	case b.USRPIn <- packet:
		return nil
	default:
		return fmt.Errorf("USRP input buffer full")
	}
}

// GetUSRPPacket gets a USRP packet from Discord audio
func (b *Bridge) GetUSRPPacket() (*usrp.VoiceMessage, bool) {
	select {
	case packet := <-b.USRPOut:
		return packet, true
	default:
		return nil, false
	}
}

// usrpToDiscordWorker converts USRP packets to Discord audio
func (b *Bridge) usrpToDiscordWorker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-b.stopChan:
			return
		case usrpPacket := <-b.USRPIn:
			if err := b.processUSRPToDiscord(usrpPacket); err != nil {
				log.Printf("Error processing USRP to Discord: %v", err)
			}
		}
	}
}

// discordToUSRPWorker converts Discord audio to USRP packets
func (b *Bridge) discordToUSRPWorker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-b.stopChan:
			return
		case discordAudio := <-b.bot.AudioIn:
			if err := b.processDiscordToUSRP(discordAudio); err != nil {
				log.Printf("Error processing Discord to USRP: %v", err)
			}
		}
	}
}

// processUSRPToDiscord converts USRP voice packet to Discord audio
func (b *Bridge) processUSRPToDiscord(usrpPacket *usrp.VoiceMessage) error {
	// Check if this is an active voice packet
	if !usrpPacket.Header.IsPTT() {
		return nil // Skip non-PTT packets
	}
	
	// Convert USRP audio samples to Discord format
	// USRP: 8kHz mono, 160 samples (20ms)
	// Discord: 48kHz stereo (need resampling)
	
	discordAudio := b.resampleUSRPToDiscord(usrpPacket.AudioData[:])
	
	// Send to Discord bot
	if len(discordAudio) > 0 {
		// Convert to bytes
		audioBytes := make([]byte, len(discordAudio)*2)
		for i, sample := range discordAudio {
			audioBytes[i*2] = byte(sample)
			audioBytes[i*2+1] = byte(sample >> 8)
		}
		
		if err := b.bot.SendAudio(audioBytes); err != nil {
			return fmt.Errorf("failed to send audio to Discord: %w", err)
		}
	}
	
	return nil
}

// processDiscordToUSRP converts Discord audio to USRP packets
func (b *Bridge) processDiscordToUSRP(discordAudio []byte) error {
	// Convert bytes to int16 samples
	samples := make([]int16, len(discordAudio)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(discordAudio[i*2]) | int16(discordAudio[i*2+1])<<8
	}
	
	// Add to buffer for resampling
	b.discordBuffer = append(b.discordBuffer, samples...)
	
	// Process in chunks suitable for USRP (160 samples at 8kHz)
	for len(b.discordBuffer) >= 960 { // 960 samples at 48kHz = 160 at 8kHz
		// Resample Discord audio (48kHz) to USRP (8kHz)
		usrpSamples := b.resampleDiscordToUSRP(b.discordBuffer[:960])
		b.discordBuffer = b.discordBuffer[960:]
		
		// Check if audio level is above threshold (voice activity detection)
		if b.detectVoiceActivity(usrpSamples) {
			// Create USRP voice packet
			usrpPacket := &usrp.VoiceMessage{
				Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, b.generateSequence()),
			}
			usrpPacket.Header.SetPTT(true)
			usrpPacket.Header.TalkGroup = b.config.TalkGroup
			
			// Copy audio samples
			copy(usrpPacket.AudioData[:], usrpSamples)
			
			// Send to USRP output
			select {
			case b.USRPOut <- usrpPacket:
				// Sent successfully
			default:
				log.Printf("USRP output buffer full, dropping packet")
			}
		}
	}
	
	return nil
}

// resampleUSRPToDiscord converts 8kHz mono to 48kHz stereo
func (b *Bridge) resampleUSRPToDiscord(usrpSamples []int16) []int16 {
	if !b.config.EnableResampling {
		return usrpSamples // Return as-is if resampling disabled
	}
	
	// Simple 6x upsampling (8kHz -> 48kHz) with duplication
	// Real implementation would use proper resampling algorithms
	discordSamples := make([]int16, len(usrpSamples)*6*2) // 6x rate, 2x channels
	
	for i, sample := range usrpSamples {
		// Each USRP sample becomes 6 Discord samples (stereo)
		for j := 0; j < 6; j++ {
			idx := (i*6 + j) * 2
			discordSamples[idx] = sample     // Left channel
			discordSamples[idx+1] = sample   // Right channel
		}
	}
	
	return discordSamples
}

// resampleDiscordToUSRP converts 48kHz stereo to 8kHz mono
func (b *Bridge) resampleDiscordToUSRP(discordSamples []int16) []int16 {
	if !b.config.EnableResampling {
		// Convert stereo to mono
		monoSamples := make([]int16, len(discordSamples)/2)
		for i := 0; i < len(monoSamples); i++ {
			// Mix left and right channels
			left := int32(discordSamples[i*2])
			right := int32(discordSamples[i*2+1])
			monoSamples[i] = int16((left + right) / 2)
		}
		return monoSamples
	}
	
	// Simple 6x downsampling (48kHz -> 8kHz) with decimation
	// Real implementation would use proper anti-aliasing filters
	usrpSamples := make([]int16, len(discordSamples)/12) // 6x rate, 2x channels
	
	for i := 0; i < len(usrpSamples); i++ {
		// Take every 6th sample, mix stereo to mono
		idx := i * 12
		if idx+1 < len(discordSamples) {
			left := int32(discordSamples[idx])
			right := int32(discordSamples[idx+1])
			usrpSamples[i] = int16((left + right) / 2)
		}
	}
	
	return usrpSamples
}

// detectVoiceActivity checks if audio contains voice activity
func (b *Bridge) detectVoiceActivity(samples []int16) bool {
	if b.config.VoiceThreshold == 0 {
		return true // Always transmit if threshold is 0
	}
	
	// Calculate RMS (Root Mean Square) for voice detection
	var sum int64
	for _, sample := range samples {
		sum += int64(sample) * int64(sample)
	}
	
	rms := int16(sum / int64(len(samples)))
	if rms < 0 {
		rms = -rms
	}
	
	return rms > b.config.VoiceThreshold
}

// generateSequence generates a sequence number for USRP packets
func (b *Bridge) generateSequence() uint32 {
	return uint32(time.Now().Unix())
}

// IsRunning returns true if bridge is running
func (b *Bridge) IsRunning() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.running
}

// IsDiscordConnected returns true if Discord bot is connected to voice
func (b *Bridge) IsDiscordConnected() bool {
	return b.bot.IsConnected()
}

// JoinDiscordChannel joins a Discord voice channel
func (b *Bridge) JoinDiscordChannel(guildID, channelID string) error {
	return b.bot.JoinVoiceChannel(guildID, channelID)
}

// LeaveDiscordChannel leaves the current Discord voice channel
func (b *Bridge) LeaveDiscordChannel() error {
	return b.bot.LeaveVoiceChannel()
}