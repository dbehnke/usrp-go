// Package discord provides Discord bot integration for USRP audio bridging
package discord

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Bot represents a Discord bot with voice capabilities
type Bot struct {
	session    *discordgo.Session
	guildID    string
	channelID  string
	voiceConn  *discordgo.VoiceConnection
	
	// Audio channels for bridging
	AudioIn    chan []byte              // PCM audio from Discord
	AudioOut   chan []byte              // PCM audio to Discord
	
	// Control channels
	stopChan   chan bool
	running    bool
	mutex      sync.Mutex
	
	// Configuration
	config     *BotConfig
}

// BotConfig holds Discord bot configuration
type BotConfig struct {
	Token       string        // Discord bot token
	GuildID     string        // Discord server (guild) ID
	ChannelID   string        // Voice channel ID to join
	SampleRate  int           // Audio sample rate (48000 for Discord)
	Channels    int           // Audio channels (2 for Discord stereo)
	FrameSize   time.Duration // Audio frame duration (20ms)
	BufferSize  int           // Audio buffer size
}

// DefaultBotConfig returns default configuration for Discord bot
func DefaultBotConfig() *BotConfig {
	return &BotConfig{
		SampleRate:  48000,                  // Discord standard
		Channels:    2,                      // Stereo
		FrameSize:   20 * time.Millisecond,  // 20ms frames
		BufferSize:  100,                    // Channel buffer size
	}
}

// NewBot creates a new Discord bot instance
func NewBot(config *BotConfig) (*Bot, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("Discord bot token is required")
	}
	
	session, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}
	
	bot := &Bot{
		session:   session,
		guildID:   config.GuildID,
		channelID: config.ChannelID,
		AudioIn:   make(chan []byte, config.BufferSize),
		AudioOut:  make(chan []byte, config.BufferSize),
		stopChan:  make(chan bool, 1),
		config:    config,
	}
	
	// Set up event handlers
	bot.setupEventHandlers()
	
	return bot, nil
}

// setupEventHandlers configures Discord event handlers
func (b *Bot) setupEventHandlers() {
	b.session.AddHandler(b.onReady)
	b.session.AddHandler(b.onVoiceStateUpdate)
	b.session.AddHandler(b.onMessageCreate)
}

// onReady handles the ready event when bot connects
func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Discord bot ready: %s#%s", event.User.Username, event.User.Discriminator)
	
	// Set bot status
	err := s.UpdateGameStatus(0, "Amateur Radio Bridge 📻")
	if err != nil {
		log.Printf("Error setting status: %v", err)
	}
}

// onVoiceStateUpdate handles voice state changes
func (b *Bot) onVoiceStateUpdate(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	// Handle voice state changes if needed
	if event.UserID == s.State.User.ID {
		log.Printf("Bot voice state changed: Channel=%s, Guild=%s", 
			event.ChannelID, event.GuildID)
	}
}

// onMessageCreate handles incoming messages (for commands)
func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	
	// Simple command handling
	switch m.Content {
	case "!join":
		if err := b.JoinVoiceChannel(m.GuildID, m.ChannelID); err != nil {
			if _, msgErr := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error joining voice: %v", err)); msgErr != nil {
				log.Printf("Failed to send error message: %v", msgErr)
			}
		} else {
			if _, msgErr := s.ChannelMessageSend(m.ChannelID, "Joined voice channel! 📻"); msgErr != nil {
				log.Printf("Failed to send success message: %v", msgErr)
			}
		}
	case "!leave":
		if err := b.LeaveVoiceChannel(); err != nil {
			if _, msgErr := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error leaving voice: %v", err)); msgErr != nil {
				log.Printf("Failed to send error message: %v", msgErr)
			}
		} else {
			if _, msgErr := s.ChannelMessageSend(m.ChannelID, "Left voice channel! 👋"); msgErr != nil {
				log.Printf("Failed to send success message: %v", msgErr)
			}
		}
	case "!status":
		status := "Disconnected"
		if b.IsConnected() {
			status = "Connected to voice channel"
		}
		if _, msgErr := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Status: %s", status)); msgErr != nil {
			log.Printf("Failed to send status message: %v", msgErr)
		}
	}
}

// Start starts the Discord bot
func (b *Bot) Start(ctx context.Context) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if b.running {
		return fmt.Errorf("bot is already running")
	}
	
	// Open Discord session
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}
	
	b.running = true
	
	// Start audio processing goroutine
	go b.audioProcessor(ctx)
	
	log.Println("Discord bot started successfully")
	return nil
}

// Stop stops the Discord bot
func (b *Bot) Stop() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if !b.running {
		return nil
	}
	
	b.running = false
	b.stopChan <- true
	
	// Leave voice channel if connected
	if b.voiceConn != nil {
		if err := b.voiceConn.Disconnect(); err != nil {
			log.Printf("Error disconnecting from voice: %v", err)
		}
		b.voiceConn = nil
	}
	
	// Close Discord session
	if err := b.session.Close(); err != nil {
		log.Printf("Error closing Discord session: %v", err)
	}
	
	log.Println("Discord bot stopped")
	return nil
}

// JoinVoiceChannel joins a Discord voice channel
func (b *Bot) JoinVoiceChannel(guildID, channelID string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	// Leave current channel if connected
	if b.voiceConn != nil {
		if err := b.voiceConn.Disconnect(); err != nil {
			log.Printf("Error disconnecting from previous voice channel: %v", err)
		}
		b.voiceConn = nil
	}
	
	// Join new channel
	voiceConn, err := b.session.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %w", err)
	}
	
	b.voiceConn = voiceConn
	b.guildID = guildID
	b.channelID = channelID
	
	// Wait for connection to be ready
	if voiceConn.Ready {
		log.Printf("Successfully joined voice channel: %s", channelID)
	} else {
		// Give it a moment to connect
		time.Sleep(2 * time.Second)
		if !voiceConn.Ready {
			return fmt.Errorf("voice connection not ready")
		}
		log.Printf("Successfully joined voice channel: %s", channelID)
	}
	
	// Start receiving audio
	go b.receiveAudio()
	
	return nil
}

// LeaveVoiceChannel leaves the current voice channel
func (b *Bot) LeaveVoiceChannel() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if b.voiceConn != nil {
		if err := b.voiceConn.Disconnect(); err != nil {
			return fmt.Errorf("failed to disconnect from voice: %w", err)
		}
		b.voiceConn = nil
		log.Println("Left voice channel")
	}
	
	return nil
}

// IsConnected returns true if bot is connected to a voice channel
func (b *Bot) IsConnected() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.voiceConn != nil && b.voiceConn.Ready
}

// SendAudio sends PCM audio to Discord voice channel
func (b *Bot) SendAudio(pcmData []byte) error {
	if !b.IsConnected() {
		return fmt.Errorf("not connected to voice channel")
	}
	
	// Send audio to Discord (non-blocking)
	select {
	case b.AudioOut <- pcmData:
		return nil
	default:
		// Drop audio if buffer is full
		return fmt.Errorf("audio buffer full, dropping frame")
	}
}

// receiveAudio handles incoming audio from Discord
func (b *Bot) receiveAudio() {
	if b.voiceConn == nil {
		return
	}
	
	// Note: Discord audio receiving is more complex in practice
	// This is a simplified version for the bridge concept
	log.Println("Audio receiver started (simplified implementation)")
	
	// In a real implementation, you would need to:
	// 1. Handle Discord's voice packets
	// 2. Decode Opus audio to PCM
	// 3. Convert sample rates appropriately
	
	for {
		select {
		case <-b.stopChan:
			return
		case <-time.After(100 * time.Millisecond):
			// Placeholder - in real implementation, process incoming voice packets
			continue
		}
	}
}

// audioProcessor handles audio streaming to Discord
func (b *Bot) audioProcessor(ctx context.Context) {
	ticker := time.NewTicker(b.config.FrameSize)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopChan:
			return
		case <-ticker.C:
			// Process outgoing audio
			select {
			case pcmData := <-b.AudioOut:
				if b.IsConnected() && len(pcmData) > 0 {
					// Convert PCM to Opus and send to Discord
					// Note: This is simplified - real implementation needs proper Opus encoding
					log.Printf("Sending %d bytes of audio to Discord", len(pcmData))
				}
			default:
				// No audio to send
			}
		}
	}
}

// GetAudioSpecs returns audio specifications for this bot
func (b *Bot) GetAudioSpecs() (sampleRate int, channels int, frameSize time.Duration) {
	return b.config.SampleRate, b.config.Channels, b.config.FrameSize
}