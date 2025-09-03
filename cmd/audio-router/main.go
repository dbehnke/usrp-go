// Audio Router Hub - Scalable hub-and-spoke audio routing for amateur radio
//
// Architecture:
//   AllStarLink-1 ←┐
//   AllStarLink-2 ←┤
//   AllStarLink-N ←┤    ┌─→ WhoTalkie-1
//                  ├────┤   WhoTalkie-2
//   Discord-1 ←────┤    └─→ WhoTalkie-N
//   Discord-2 ←────┤
//   Discord-N ←────┘
//
// All services communicate through the central audio router hub
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
	"sync"
	"syscall"
	"time"

	"github.com/dbehnke/usrp-go/pkg/audio"
	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// ServiceType represents the type of audio service
type ServiceType string

const (
	ServiceTypeUSRP      ServiceType = "usrp"      // AllStarLink nodes
	ServiceTypeWhoTalkie ServiceType = "whotalkie" // WhoTalkie instances
	ServiceTypeDiscord   ServiceType = "discord"   // Discord bots
	ServiceTypeGeneric   ServiceType = "generic"   // Custom services
)

// ServiceInstance represents a single service instance
type ServiceInstance struct {
	ID          string            `json:"id"`
	Type        ServiceType       `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	
	// Network configuration
	Network struct {
		Protocol   string `json:"protocol"`    // "udp", "tcp"
		ListenAddr string `json:"listen_addr"` // For incoming (empty = don't listen)
		ListenPort int    `json:"listen_port"`
		RemoteAddr string `json:"remote_addr"` // For outgoing (empty = don't send)
		RemotePort int    `json:"remote_port"`
	} `json:"network"`
	
	// Audio configuration
	Audio struct {
		Format     string `json:"format"`      // "pcm", "opus", "ogg"
		SampleRate int    `json:"sample_rate"` // Hz
		Channels   int    `json:"channels"`    // 1=mono, 2=stereo
		Bitrate    int    `json:"bitrate"`     // For compressed formats
	} `json:"audio"`
	
	// Service-specific settings
	Settings map[string]interface{} `json:"settings,omitempty"`
	
	// Routing configuration
	Routing struct {
		CanSend        bool     `json:"can_send"`         // Can send audio to router
		CanReceive     bool     `json:"can_receive"`      // Can receive audio from router
		SendToTypes    []string `json:"send_to_types"`    // Which service types to send to
		ReceiveFrom    []string `json:"receive_from"`     // Which service types to receive from
		ExcludeServices []string `json:"exclude_services"` // Specific service IDs to exclude
		Priority       int      `json:"priority"`         // Higher = higher priority (0-10)
	} `json:"routing"`
}

// AudioRouterConfig holds the complete router configuration
type AudioRouterConfig struct {
	// Router settings
	Router struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ListenAddr  string `json:"listen_addr"`
		StatusPort  int    `json:"status_port"` // HTTP status/metrics port
	} `json:"router"`
	
	// Audio processing
	Audio struct {
		BufferSize       int           `json:"buffer_size"`        // Channel buffer size
		ProcessingDelay  int           `json:"processing_delay"`   // ms
		MaxConcurrentTx  int           `json:"max_concurrent_tx"`  // Max simultaneous transmissions
		TxTimeoutSeconds int           `json:"tx_timeout_seconds"` // TX timeout
		EnableConversion bool          `json:"enable_conversion"`  // Enable format conversion
		DefaultFormat    string        `json:"default_format"`     // Default audio format
	} `json:"audio"`
	
	// Routing rules
	Routing struct {
		PreventLoops        bool     `json:"prevent_loops"`         // Prevent audio loops
		EnablePriorityRules bool     `json:"enable_priority_rules"` // Use priority for conflicts
		DefaultRouting      string   `json:"default_routing"`       // "all-to-all", "hub-only", "none"
		BlockedPairs        []string `json:"blocked_pairs"`         // Service pairs to block (e.g. "discord1->usrp2")
	} `json:"routing"`
	
	// Amateur radio settings
	Amateur struct {
		StationCall       string `json:"station_call"`
		DefaultTalkGroup  uint32 `json:"default_talk_group"`
		RequireValidCall  bool   `json:"require_valid_call"`
		LogTransmissions  bool   `json:"log_transmissions"`
	} `json:"amateur"`
	
	// Service instances
	Services []ServiceInstance `json:"services"`
}

// AudioMessage represents audio flowing through the router
type AudioMessage struct {
	// Source information
	SourceID      string      `json:"source_id"`
	SourceType    ServiceType `json:"source_type"`
	SourceName    string      `json:"source_name"`
	
	// Audio data
	Data          []byte    `json:"data"`
	Format        string    `json:"format"`
	SampleRate    int       `json:"sample_rate"`
	Channels      int       `json:"channels"`
	Duration      time.Duration `json:"duration"`
	
	// Metadata
	Timestamp     time.Time `json:"timestamp"`
	SequenceNum   uint32    `json:"sequence_num"`
	PTTActive     bool      `json:"ptt_active"`
	CallSign      string    `json:"call_sign"`
	TalkGroup     uint32    `json:"talk_group"`
	
	// Routing
	RouteToTypes  []ServiceType `json:"route_to_types"`
	ExcludeIDs    []string      `json:"exclude_ids"`
	Priority      int           `json:"priority"`
}

// ServiceConnection represents an active service connection
type ServiceConnection struct {
	Instance   *ServiceInstance
	Connection net.Conn
	LastSeen   time.Time
	TxActive   bool
	RxActive   bool
	
	// Statistics
	Stats struct {
		MessagesSent     uint64
		MessagesReceived uint64
		BytesSent        uint64
		BytesReceived    uint64
		LastActivity     time.Time
		Errors           uint64
	}
}

// AudioRouter is the main hub-and-spoke audio router
type AudioRouter struct {
	config    *AudioRouterConfig
	converter audio.Converter
	
	// Service management
	services    map[string]*ServiceConnection // serviceID -> connection
	servicesMux sync.RWMutex
	
	// Audio routing
	audioHub          chan *AudioMessage
	activeTransmissions map[string]*AudioMessage // sourceID -> current transmission
	txMux             sync.RWMutex
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	
	// Statistics
	stats struct {
		TotalMessages      uint64
		RoutedMessages     uint64
		DroppedMessages    uint64
		ConversionErrors   uint64
		ActiveServices     int
		ActiveTransmissions int
		UptimeStart       time.Time
	}
	statsMux sync.RWMutex
}

func main() {
	var (
		configFile = flag.String("config", "", "Configuration file path (JSON)")
		genConfig  = flag.Bool("generate-config", false, "Generate sample configuration file")
		statusPort = flag.Int("status-port", 9090, "HTTP status/metrics port")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *genConfig {
		generateSampleConfig()
		return
	}

	// Load configuration
	var config *AudioRouterConfig
	if *configFile != "" {
		var err error
		config, err = loadConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		config = defaultConfig()
		config.Router.StatusPort = *statusPort
	}

	// Setup logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	printBanner(config)

	// Create and start router
	router, err := NewAudioRouter(config)
	if err != nil {
		log.Fatalf("Failed to create audio router: %v", err)
	}

	if err := router.Start(); err != nil {
		log.Fatalf("Failed to start audio router: %v", err)
	}
	defer router.Stop()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	fmt.Println("🚀 Audio Router Hub is running!")
	fmt.Println("📊 Send SIGUSR1 for statistics")
	fmt.Printf("🌐 Status page: http://localhost:%d/status\n", config.Router.StatusPort)
	fmt.Println("Press Ctrl+C to stop...")

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGUSR1:
			router.PrintStats()
		case syscall.SIGINT, syscall.SIGTERM:
			fmt.Println("\n🛑 Shutting down Audio Router Hub...")
			return
		}
	}
}

func printBanner(config *AudioRouterConfig) {
	fmt.Println("🎵 Audio Router Hub - Amateur Radio Voice Bridge")
	fmt.Println("==============================================")
	fmt.Printf("📻 Station: %s\n", config.Amateur.StationCall)
	fmt.Printf("🎛️  Router: %s\n", config.Router.Name)
	
	// Count services by type
	serviceCounts := make(map[ServiceType]int)
	enabledServices := 0
	
	for _, svc := range config.Services {
		serviceCounts[svc.Type]++
		if svc.Enabled {
			enabledServices++
		}
	}
	
	fmt.Printf("🔧 Services: %d total, %d enabled\n", len(config.Services), enabledServices)
	for svcType, count := range serviceCounts {
		enabled := 0
		for _, svc := range config.Services {
			if svc.Type == svcType && svc.Enabled {
				enabled++
			}
		}
		fmt.Printf("   %s: %d total (%d enabled)\n", svcType, count, enabled)
	}
	
	fmt.Printf("🔄 Routing: %s, Priority Rules: %v, Loop Prevention: %v\n", 
		config.Routing.DefaultRouting, 
		config.Routing.EnablePriorityRules,
		config.Routing.PreventLoops)
	fmt.Println()
}

// NewAudioRouter creates a new audio router hub
func NewAudioRouter(config *AudioRouterConfig) (*AudioRouter, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	router := &AudioRouter{
		config:              config,
		services:            make(map[string]*ServiceConnection),
		audioHub:            make(chan *AudioMessage, config.Audio.BufferSize),
		activeTransmissions: make(map[string]*AudioMessage),
		ctx:                 ctx,
		cancel:              cancel,
	}
	
	router.stats.UptimeStart = time.Now()

	// Create audio converter if enabled
	if config.Audio.EnableConversion {
		var err error
		switch config.Audio.DefaultFormat {
		case "opus":
			router.converter, err = audio.NewOpusConverter()
		case "ogg":
			router.converter, err = audio.NewOggOpusConverter()
		default:
			return nil, fmt.Errorf("unsupported default audio format: %s", config.Audio.DefaultFormat)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to create audio converter: %w", err)
		}
	}

	return router, nil
}

// Start starts the audio router hub
func (r *AudioRouter) Start() error {
	// Start the main audio routing hub
	go r.audioHubWorker()
	
	// Start service connections
	for i := range r.config.Services {
		service := &r.config.Services[i]
		if service.Enabled {
			if err := r.startService(service); err != nil {
				log.Printf("Warning: Failed to start service %s: %v", service.ID, err)
			}
		}
	}
	
	// Start HTTP status server
	go r.startStatusServer()
	
	// Start housekeeping
	go r.housekeepingWorker()
	
	return nil
}

// Stop stops the audio router hub
func (r *AudioRouter) Stop() error {
	r.cancel()
	
	// Stop all service connections
	r.servicesMux.Lock()
	for _, conn := range r.services {
		if conn.Connection != nil {
			conn.Connection.Close()
		}
	}
	r.servicesMux.Unlock()
	
	// Stop audio converter
	if r.converter != nil {
		r.converter.Close()
	}
	
	return nil
}

// startService starts a connection to a service
func (r *AudioRouter) startService(service *ServiceInstance) error {
	conn := &ServiceConnection{
		Instance: service,
		LastSeen: time.Now(),
	}
	
	r.servicesMux.Lock()
	r.services[service.ID] = conn
	r.servicesMux.Unlock()
	
	// Start service-specific worker
	switch service.Type {
	case ServiceTypeUSRP:
		go r.usrpServiceWorker(conn)
	case ServiceTypeWhoTalkie:
		go r.whoTalkieServiceWorker(conn)
	case ServiceTypeDiscord:
		go r.discordServiceWorker(conn)
	case ServiceTypeGeneric:
		go r.genericServiceWorker(conn)
	}
	
	log.Printf("Started service: %s (%s) - %s", service.Name, service.Type, service.Description)
	return nil
}

// audioHubWorker is the main audio routing hub
func (r *AudioRouter) audioHubWorker() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case msg := <-r.audioHub:
			r.routeAudioMessage(msg)
		}
	}
}

// routeAudioMessage routes an audio message to appropriate destinations
func (r *AudioRouter) routeAudioMessage(msg *AudioMessage) {
	r.statsMux.Lock()
	r.stats.TotalMessages++
	r.statsMux.Unlock()
	
	// Handle transmission management
	if err := r.manageTransmission(msg); err != nil {
		log.Printf("Transmission management error: %v", err)
		r.statsMux.Lock()
		r.stats.DroppedMessages++
		r.statsMux.Unlock()
		return
	}
	
	// Determine routing destinations
	destinations := r.getRoutingDestinations(msg)
	if len(destinations) == 0 {
		return // No destinations
	}
	
	// Route to each destination
	routed := 0
	for _, destService := range destinations {
		if r.sendToService(msg, destService) {
			routed++
		}
	}
	
	r.statsMux.Lock()
	if routed > 0 {
		r.stats.RoutedMessages++
	} else {
		r.stats.DroppedMessages++
	}
	r.statsMux.Unlock()
}

// manageTransmission handles transmission conflicts and timeouts
func (r *AudioRouter) manageTransmission(msg *AudioMessage) error {
	r.txMux.Lock()
	defer r.txMux.Unlock()
	
	now := time.Now()
	
	// Clean up expired transmissions
	for sourceID, activeTx := range r.activeTransmissions {
		if now.Sub(activeTx.Timestamp) > time.Duration(r.config.Audio.TxTimeoutSeconds)*time.Second {
			delete(r.activeTransmissions, sourceID)
		}
	}
	
	// Check for conflicts
	if msg.PTTActive {
		// Starting transmission
		if len(r.activeTransmissions) >= r.config.Audio.MaxConcurrentTx {
			if r.config.Routing.EnablePriorityRules {
				// Check if this message has higher priority than existing transmissions
				canPreempt := false
				for _, activeTx := range r.activeTransmissions {
					if msg.Priority > activeTx.Priority {
						canPreempt = true
						break
					}
				}
				if !canPreempt {
					return fmt.Errorf("transmission rejected: max concurrent limit reached")
				}
			} else {
				return fmt.Errorf("transmission rejected: max concurrent limit reached")
			}
		}
		
		r.activeTransmissions[msg.SourceID] = msg
	} else {
		// Ending transmission
		delete(r.activeTransmissions, msg.SourceID)
	}
	
	r.statsMux.Lock()
	r.stats.ActiveTransmissions = len(r.activeTransmissions)
	r.statsMux.Unlock()
	
	return nil
}

// getRoutingDestinations determines where to route an audio message
func (r *AudioRouter) getRoutingDestinations(msg *AudioMessage) []*ServiceConnection {
	var destinations []*ServiceConnection
	
	r.servicesMux.RLock()
	defer r.servicesMux.RUnlock()
	
	// Find source service for routing rules
	var sourceService *ServiceInstance
	if sourceConn, exists := r.services[msg.SourceID]; exists {
		sourceService = sourceConn.Instance
	}
	
	for _, conn := range r.services {
		destService := conn.Instance
		
		// Skip if destination is disabled
		if !destService.Enabled || !destService.Routing.CanReceive {
			continue
		}
		
		// Skip self
		if destService.ID == msg.SourceID {
			continue
		}
		
		// Check if explicitly excluded
		excluded := false
		for _, excludeID := range msg.ExcludeIDs {
			if destService.ID == excludeID {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		
		// Check service-level exclusions
		if sourceService != nil {
			excluded = false
			for _, excludeID := range sourceService.Routing.ExcludeServices {
				if destService.ID == excludeID {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}
		
		// Apply routing rules
		if r.shouldRoute(sourceService, destService, msg) {
			destinations = append(destinations, conn)
		}
	}
	
	return destinations
}

// shouldRoute determines if audio should be routed between two services
func (r *AudioRouter) shouldRoute(source *ServiceInstance, dest *ServiceInstance, msg *AudioMessage) bool {
	// Default routing rules
	switch r.config.Routing.DefaultRouting {
	case "all-to-all":
		return true
	case "hub-only":
		// Only route if one service is designated as hub
		return false // TODO: implement hub designation
	case "none":
		return false
	}
	
	// Check source routing rules
	if source != nil && len(source.Routing.SendToTypes) > 0 {
		found := false
		for _, allowedType := range source.Routing.SendToTypes {
			if allowedType == string(dest.Type) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check destination routing rules
	if len(dest.Routing.ReceiveFrom) > 0 {
		found := false
		for _, allowedType := range dest.Routing.ReceiveFrom {
			if source != nil && allowedType == string(source.Type) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check message-level routing
	if len(msg.RouteToTypes) > 0 {
		found := false
		for _, allowedType := range msg.RouteToTypes {
			if allowedType == dest.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

// sendToService sends an audio message to a specific service
func (r *AudioRouter) sendToService(msg *AudioMessage, destConn *ServiceConnection) bool {
	destService := destConn.Instance
	
	// Convert audio format if needed
	audioData := msg.Data
	if r.converter != nil && msg.Format != destService.Audio.Format {
		// TODO: Implement format conversion based on service requirements
		_ = audioData // Placeholder
	}
	
	// Send based on service type
	switch destService.Type {
	case ServiceTypeUSRP:
		return r.sendToUSRPService(msg, destConn)
	case ServiceTypeWhoTalkie:
		return r.sendToWhoTalkieService(msg, destConn)
	case ServiceTypeDiscord:
		return r.sendToDiscordService(msg, destConn)
	case ServiceTypeGeneric:
		return r.sendToGenericService(msg, destConn)
	}
	
	return false
}

// Service-specific worker and sender functions
func (r *AudioRouter) usrpServiceWorker(conn *ServiceConnection) {
	service := conn.Instance
	log.Printf("Starting USRP service worker for %s", service.Name)
	
	// Set up UDP listening if configured
	var listener net.PacketConn
	if service.Network.ListenAddr != "" {
		addr := fmt.Sprintf("%s:%d", service.Network.ListenAddr, service.Network.ListenPort)
		var err error
		listener, err = net.ListenPacket("udp", addr)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", addr, err)
			return
		}
		defer listener.Close()
		log.Printf("USRP service %s listening on %s", service.Name, addr)
	}
	
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			if listener != nil {
				// Read USRP packets
				buffer := make([]byte, 1024)
				listener.SetReadDeadline(time.Now().Add(1 * time.Second))
				n, remoteAddr, err := listener.ReadFrom(buffer)
				if err != nil {
					if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
						log.Printf("USRP read error: %v", err)
					}
					continue
				}
				
				// Parse USRP packet
				if err := r.handleUSRPPacket(service, buffer[:n], remoteAddr); err != nil {
					log.Printf("USRP packet handling error: %v", err)
				}
				
				conn.Stats.MessagesReceived++
				conn.Stats.BytesReceived += uint64(n)
				conn.Stats.LastActivity = time.Now()
				conn.LastSeen = time.Now()
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (r *AudioRouter) whoTalkieServiceWorker(conn *ServiceConnection) {
	service := conn.Instance
	log.Printf("Starting WhoTalkie service worker for %s", service.Name)
	
	// Set up UDP listening if configured
	var listener net.PacketConn
	if service.Network.ListenAddr != "" {
		addr := fmt.Sprintf("%s:%d", service.Network.ListenAddr, service.Network.ListenPort)
		var err error
		listener, err = net.ListenPacket("udp", addr)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", addr, err)
			return
		}
		defer listener.Close()
		log.Printf("WhoTalkie service %s listening on %s", service.Name, addr)
	}
	
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			if listener != nil {
				// Read WhoTalkie audio packets (typically Opus)
				buffer := make([]byte, 4096)
				listener.SetReadDeadline(time.Now().Add(1 * time.Second))
				n, remoteAddr, err := listener.ReadFrom(buffer)
				if err != nil {
					if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
						log.Printf("WhoTalkie read error: %v", err)
					}
					continue
				}
				
				// Handle WhoTalkie audio packet
				if err := r.handleWhoTalkiePacket(service, buffer[:n], remoteAddr); err != nil {
					log.Printf("WhoTalkie packet handling error: %v", err)
				}
				
				conn.Stats.MessagesReceived++
				conn.Stats.BytesReceived += uint64(n)
				conn.Stats.LastActivity = time.Now()
				conn.LastSeen = time.Now()
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (r *AudioRouter) discordServiceWorker(conn *ServiceConnection) {
	service := conn.Instance
	log.Printf("Starting Discord service worker for %s", service.Name)
	
	// Discord integration would require Discord bot setup
	// For now, this is a placeholder that would integrate with our Discord bridge
	// The actual implementation would use the discord bridge from pkg/discord
	
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			// Discord audio handling would go here
			// This would integrate with the DiscordBridge from pkg/discord
			time.Sleep(1 * time.Second)
			conn.LastSeen = time.Now()
		}
	}
}

func (r *AudioRouter) genericServiceWorker(conn *ServiceConnection) {
	service := conn.Instance
	log.Printf("Starting generic service worker for %s", service.Name)
	
	// Generic UDP/TCP service worker
	var listener net.Listener
	var packetListener net.PacketConn
	
	if service.Network.ListenAddr != "" {
		addr := fmt.Sprintf("%s:%d", service.Network.ListenAddr, service.Network.ListenPort)
		
		if service.Network.Protocol == "tcp" {
			var err error
			listener, err = net.Listen("tcp", addr)
			if err != nil {
				log.Printf("Failed to listen on TCP %s: %v", addr, err)
				return
			}
			defer listener.Close()
			log.Printf("Generic service %s listening on TCP %s", service.Name, addr)
			
			// Handle TCP connections
			for {
				select {
				case <-r.ctx.Done():
					return
				default:
					listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
					conn, err := listener.Accept()
					if err != nil {
						if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
							log.Printf("Generic TCP accept error: %v", err)
						}
						continue
					}
					go r.handleGenericTCPConnection(service, conn)
				}
			}
		} else {
			// UDP
			var err error
			packetListener, err = net.ListenPacket("udp", addr)
			if err != nil {
				log.Printf("Failed to listen on UDP %s: %v", addr, err)
				return
			}
			defer packetListener.Close()
			log.Printf("Generic service %s listening on UDP %s", service.Name, addr)
		}
	}
	
	// UDP packet handling loop
	if packetListener != nil {
		for {
			select {
			case <-r.ctx.Done():
				return
			default:
				buffer := make([]byte, 4096)
				packetListener.SetReadDeadline(time.Now().Add(1 * time.Second))
				n, remoteAddr, err := packetListener.ReadFrom(buffer)
				if err != nil {
					if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
						log.Printf("Generic UDP read error: %v", err)
					}
					continue
				}
				
				// Handle generic audio packet
				if err := r.handleGenericPacket(service, buffer[:n], remoteAddr); err != nil {
					log.Printf("Generic packet handling error: %v", err)
				}
			}
		}
	} else {
		// No listening configured, just maintain connection
		for {
			select {
			case <-r.ctx.Done():
				return
			default:
				time.Sleep(1 * time.Second)
				conn.LastSeen = time.Now()
			}
		}
	}
}

func (r *AudioRouter) sendToUSRPService(msg *AudioMessage, conn *ServiceConnection) bool {
	service := conn.Instance
	
	// Skip if no remote address configured
	if service.Network.RemoteAddr == "" {
		return false
	}
	
	// Convert audio to USRP format if needed
	var usrpData []byte
	if msg.Format == "pcm" {
		// Create USRP voice packet
		voice := &usrp.VoiceMessage{
			Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, msg.SequenceNum),
		}
		voice.Header.SetPTT(msg.PTTActive)
		voice.Header.TalkGroup = msg.TalkGroup
		
		// Copy audio data (assuming 16-bit PCM, 160 samples)
		if len(msg.Data) >= 320 {
			for i := 0; i < 160 && i*2+1 < len(msg.Data); i++ {
				// Convert bytes to int16
				voice.AudioData[i] = int16(msg.Data[i*2]) | int16(msg.Data[i*2+1])<<8
			}
		}
		
		var err error
		usrpData, err = voice.Marshal()
		if err != nil {
			log.Printf("Failed to marshal USRP packet: %v", err)
			return false
		}
	} else {
		// Use audio conversion if available
		if r.converter != nil {
			// Convert from source format to USRP
			// This would use the audio converter
			usrpData = msg.Data // Placeholder
		} else {
			log.Printf("Cannot convert audio format %s to USRP without converter", msg.Format)
			return false
		}
	}
	
	// Send UDP packet
	remoteAddr := fmt.Sprintf("%s:%d", service.Network.RemoteAddr, service.Network.RemotePort)
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		log.Printf("Failed to resolve USRP address %s: %v", remoteAddr, err)
		return false
	}
	
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Printf("Failed to dial USRP %s: %v", remoteAddr, err)
		return false
	}
	defer udpConn.Close()
	
	_, err = udpConn.Write(usrpData)
	if err != nil {
		log.Printf("Failed to send USRP packet: %v", err)
		return false
	}
	
	conn.Stats.MessagesSent++
	conn.Stats.BytesSent += uint64(len(usrpData))
	conn.Stats.LastActivity = time.Now()
	
	return true
}

func (r *AudioRouter) sendToWhoTalkieService(msg *AudioMessage, conn *ServiceConnection) bool {
	service := conn.Instance
	
	// Skip if no remote address configured
	if service.Network.RemoteAddr == "" {
		return false
	}
	
	// Convert audio to WhoTalkie format (typically Opus)
	var audioData []byte
	if r.converter != nil && msg.Format != service.Audio.Format {
		// Use audio converter to convert to Opus/Ogg
		// This would require the specific WhoTalkie format
		audioData = msg.Data // Placeholder
	} else {
		audioData = msg.Data
	}
	
	// Create WhoTalkie packet (simplified - would need actual WhoTalkie protocol)
	// For now, just send raw audio data
	remoteAddr := fmt.Sprintf("%s:%d", service.Network.RemoteAddr, service.Network.RemotePort)
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		log.Printf("Failed to resolve WhoTalkie address %s: %v", remoteAddr, err)
		return false
	}
	
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Printf("Failed to dial WhoTalkie %s: %v", remoteAddr, err)
		return false
	}
	defer udpConn.Close()
	
	_, err = udpConn.Write(audioData)
	if err != nil {
		log.Printf("Failed to send WhoTalkie packet: %v", err)
		return false
	}
	
	conn.Stats.MessagesSent++
	conn.Stats.BytesSent += uint64(len(audioData))
	conn.Stats.LastActivity = time.Now()
	
	return true
}

func (r *AudioRouter) sendToDiscordService(msg *AudioMessage, conn *ServiceConnection) bool {
	// Discord audio sending would integrate with our Discord bridge
	// This would require the Discord bot to be connected and in a voice channel
	// For now, this is a placeholder
	
	conn.Stats.MessagesSent++
	conn.Stats.BytesSent += uint64(len(msg.Data))
	conn.Stats.LastActivity = time.Now()
	
	// In a real implementation, this would:
	// 1. Convert audio format to 48kHz PCM for Discord
	// 2. Send to Discord voice gateway via WebSocket
	// 3. Handle Discord voice protocol specifics
	
	return true // Placeholder success
}

func (r *AudioRouter) sendToGenericService(msg *AudioMessage, conn *ServiceConnection) bool {
	service := conn.Instance
	
	// Skip if no remote address configured
	if service.Network.RemoteAddr == "" {
		return false
	}
	
	// Use audio data as-is for generic service
	audioData := msg.Data
	
	// Send based on protocol
	remoteAddr := fmt.Sprintf("%s:%d", service.Network.RemoteAddr, service.Network.RemotePort)
	
	if service.Network.Protocol == "tcp" {
		// TCP connection
		tcpAddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
		if err != nil {
			log.Printf("Failed to resolve generic TCP address %s: %v", remoteAddr, err)
			return false
		}
		
		tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			log.Printf("Failed to dial generic TCP %s: %v", remoteAddr, err)
			return false
		}
		defer tcpConn.Close()
		
		_, err = tcpConn.Write(audioData)
		if err != nil {
			log.Printf("Failed to send generic TCP packet: %v", err)
			return false
		}
	} else {
		// UDP connection
		udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
		if err != nil {
			log.Printf("Failed to resolve generic UDP address %s: %v", remoteAddr, err)
			return false
		}
		
		udpConn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			log.Printf("Failed to dial generic UDP %s: %v", remoteAddr, err)
			return false
		}
		defer udpConn.Close()
		
		_, err = udpConn.Write(audioData)
		if err != nil {
			log.Printf("Failed to send generic UDP packet: %v", err)
			return false
		}
	}
	
	conn.Stats.MessagesSent++
	conn.Stats.BytesSent += uint64(len(audioData))
	conn.Stats.LastActivity = time.Now()
	
	return true
}

// housekeepingWorker performs periodic maintenance
func (r *AudioRouter) housekeepingWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.performHousekeeping()
		}
	}
}

func (r *AudioRouter) performHousekeeping() {
	// Update active service count
	r.servicesMux.RLock()
	activeCount := 0
	for _, conn := range r.services {
		if conn.Instance.Enabled {
			activeCount++
		}
	}
	r.servicesMux.RUnlock()
	
	r.statsMux.Lock()
	r.stats.ActiveServices = activeCount
	r.statsMux.Unlock()
}

// startStatusServer starts the HTTP status/metrics server
func (r *AudioRouter) startStatusServer() {
	if r.config.Router.StatusPort == 0 {
		return
	}
	
	addr := fmt.Sprintf(":%d", r.config.Router.StatusPort)
	log.Printf("Starting HTTP status server on %s", addr)
	
	// This would implement a full HTTP server with JSON metrics
	// For now, just log that it would be running
	log.Printf("Status server would run on port %d", r.config.Router.StatusPort)
	
	// In a complete implementation, this would serve:
	// GET /status - JSON status and statistics
	// GET /services - List of services and their status  
	// GET /metrics - Prometheus-style metrics
	// GET /config - Current configuration (sanitized)
}

// PrintStats displays current router statistics
func (r *AudioRouter) PrintStats() {
	r.statsMux.RLock()
	stats := r.stats
	r.statsMux.RUnlock()
	
	uptime := time.Since(stats.UptimeStart)
	
	fmt.Println("\n📊 Audio Router Hub Statistics")
	fmt.Println("==============================")
	fmt.Printf("⏰ Uptime: %v\n", uptime.Round(time.Second))
	fmt.Printf("🔧 Active Services: %d\n", stats.ActiveServices)
	fmt.Printf("📡 Total Messages: %d\n", stats.TotalMessages)
	fmt.Printf("🔄 Routed Messages: %d\n", stats.RoutedMessages)
	fmt.Printf("🚫 Dropped Messages: %d\n", stats.DroppedMessages)
	fmt.Printf("❌ Conversion Errors: %d\n", stats.ConversionErrors)
	fmt.Printf("📻 Active Transmissions: %d\n", stats.ActiveTransmissions)
	
	if stats.TotalMessages > 0 {
		routeRate := float64(stats.RoutedMessages) / float64(stats.TotalMessages) * 100
		fmt.Printf("📈 Routing Success Rate: %.1f%%\n", routeRate)
	}
	
	// Show service details
	r.servicesMux.RLock()
	if len(r.services) > 0 {
		fmt.Println("\n🔗 Service Status:")
		for _, conn := range r.services {
			status := "🔴 Offline"
			if conn.Instance.Enabled {
				status = "🟢 Online"
			}
			fmt.Printf("  %s (%s): %s - %s\n", 
				conn.Instance.Name, 
				conn.Instance.Type,
				status,
				conn.Instance.Description)
		}
	}
	r.servicesMux.RUnlock()
	
	fmt.Println()
}

// parseUSRPPacket attempts to parse a USRP packet by checking the packet type
func parseUSRPPacket(data []byte) (usrp.Message, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("packet too short for USRP header")
	}
	
	// Check magic string "USRP"
	if string(data[0:4]) != "USRP" {
		return nil, fmt.Errorf("invalid USRP magic string")
	}
	
	// Extract packet type (bytes 20-23, network byte order)
	packetType := (uint32(data[20]) << 24) | (uint32(data[21]) << 16) | (uint32(data[22]) << 8) | uint32(data[23])
	
	switch usrp.PacketType(packetType) {
	case usrp.USRP_TYPE_VOICE:
		msg := &usrp.VoiceMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		return msg, nil
		
	case usrp.USRP_TYPE_DTMF:
		msg := &usrp.DTMFMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		return msg, nil
		
	case usrp.USRP_TYPE_VOICE_ULAW:
		msg := &usrp.VoiceULawMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		return msg, nil
		
	case usrp.USRP_TYPE_TLV:
		msg := &usrp.TLVMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		return msg, nil
		
	case usrp.USRP_TYPE_PING:
		msg := &usrp.PingMessage{}
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		return msg, nil
		
	default:
		return nil, fmt.Errorf("unsupported USRP packet type: %d", packetType)
	}
}

// Packet handling functions
func (r *AudioRouter) handleUSRPPacket(service *ServiceInstance, data []byte, remoteAddr net.Addr) error {
	// Parse USRP packet
	msg, err := parseUSRPPacket(data)
	if err != nil {
		return fmt.Errorf("failed to parse USRP packet: %w", err)
	}
	
	// Convert to AudioMessage based on USRP packet type
	var audioMsg *AudioMessage
	
	switch typedMsg := msg.(type) {
	case *usrp.VoiceMessage:
		// Convert USRP voice to AudioMessage
		audioData := make([]byte, 320) // 160 samples * 2 bytes
		for i, sample := range typedMsg.AudioData {
			if i*2+1 < len(audioData) {
				audioData[i*2] = byte(sample & 0xFF)
				audioData[i*2+1] = byte((sample >> 8) & 0xFF)
			}
		}
		
		audioMsg = &AudioMessage{
			SourceID:      service.ID,
			SourceType:    service.Type,
			SourceName:    service.Name,
			Data:          audioData,
			Format:        "pcm",
			SampleRate:    8000,
			Channels:      1,
			Timestamp:     time.Now(),
			SequenceNum:   typedMsg.Header.Seq,
			PTTActive:     typedMsg.Header.IsPTT(),
			TalkGroup:     typedMsg.Header.TalkGroup,
			Priority:      service.Routing.Priority,
		}
		
	case *usrp.DTMFMessage:
		// Handle DTMF (could be converted to audio tone or metadata)
		return nil // Skip DTMF for now
		
	default:
		return nil // Skip other packet types
	}
	
	if audioMsg != nil {
		// Send to audio hub for routing
		select {
		case r.audioHub <- audioMsg:
			return nil
		case <-time.After(100 * time.Millisecond):
			return fmt.Errorf("audio hub full, dropping packet")
		}
	}
	
	return nil
}

func (r *AudioRouter) handleWhoTalkiePacket(service *ServiceInstance, data []byte, remoteAddr net.Addr) error {
	// WhoTalkie packets are typically Opus-encoded audio
	// This is a simplified handler
	
	audioMsg := &AudioMessage{
		SourceID:      service.ID,
		SourceType:    service.Type,
		SourceName:    service.Name,
		Data:          data,
		Format:        service.Audio.Format, // "opus" typically
		SampleRate:    service.Audio.SampleRate,
		Channels:      service.Audio.Channels,
		Timestamp:     time.Now(),
		PTTActive:     true, // Assume active transmission
		Priority:      service.Routing.Priority,
	}
	
	// Send to audio hub for routing
	select {
	case r.audioHub <- audioMsg:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("audio hub full, dropping WhoTalkie packet")
	}
}

func (r *AudioRouter) handleGenericPacket(service *ServiceInstance, data []byte, remoteAddr net.Addr) error {
	// Generic packet handler - assumes raw audio data
	
	audioMsg := &AudioMessage{
		SourceID:      service.ID,
		SourceType:    service.Type,
		SourceName:    service.Name,
		Data:          data,
		Format:        service.Audio.Format,
		SampleRate:    service.Audio.SampleRate,
		Channels:      service.Audio.Channels,
		Timestamp:     time.Now(),
		PTTActive:     true, // Assume active transmission
		Priority:      service.Routing.Priority,
	}
	
	// Send to audio hub for routing
	select {
	case r.audioHub <- audioMsg:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("audio hub full, dropping generic packet")
	}
}

func (r *AudioRouter) handleGenericTCPConnection(service *ServiceInstance, conn net.Conn) {
	defer conn.Close()
	
	buffer := make([]byte, 4096)
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
					log.Printf("Generic TCP connection error: %v", err)
				}
				return
			}
			
			if err := r.handleGenericPacket(service, buffer[:n], conn.RemoteAddr()); err != nil {
				log.Printf("Generic TCP packet handling error: %v", err)
			}
		}
	}
}

// Configuration management functions
func loadConfig(filename string) (*AudioRouterConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config AudioRouterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	return &config, nil
}

func validateConfig(config *AudioRouterConfig) error {
	// Validate basic settings
	if config.Router.Name == "" {
		config.Router.Name = "Audio Router Hub"
	}
	
	if config.Audio.BufferSize <= 0 {
		config.Audio.BufferSize = 1000
	}
	
	if config.Audio.MaxConcurrentTx <= 0 {
		config.Audio.MaxConcurrentTx = 3
	}
	
	if config.Audio.TxTimeoutSeconds <= 0 {
		config.Audio.TxTimeoutSeconds = 30
	}
	
	// Validate services
	serviceIDs := make(map[string]bool)
	for i := range config.Services {
		service := &config.Services[i]
		
		// Ensure unique service IDs
		if service.ID == "" {
			service.ID = fmt.Sprintf("%s_%d", service.Type, i+1)
		}
		if serviceIDs[service.ID] {
			return fmt.Errorf("duplicate service ID: %s", service.ID)
		}
		serviceIDs[service.ID] = true
		
		// Validate service type
		switch service.Type {
		case ServiceTypeUSRP, ServiceTypeWhoTalkie, ServiceTypeDiscord, ServiceTypeGeneric:
		default:
			return fmt.Errorf("invalid service type: %s", service.Type)
		}
		
		// Set defaults for network
		if service.Network.Protocol == "" {
			service.Network.Protocol = "udp"
		}
		
		// Set defaults for audio
		if service.Audio.SampleRate <= 0 {
			service.Audio.SampleRate = 8000
		}
		if service.Audio.Channels <= 0 {
			service.Audio.Channels = 1
		}
		if service.Audio.Format == "" {
			switch service.Type {
			case ServiceTypeUSRP:
				service.Audio.Format = "pcm"
			case ServiceTypeWhoTalkie:
				service.Audio.Format = "opus"
			case ServiceTypeDiscord:
				service.Audio.Format = "pcm"
			default:
				service.Audio.Format = "pcm"
			}
		}
	}
	
	return nil
}

func defaultConfig() *AudioRouterConfig {
	return &AudioRouterConfig{
		Router: struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			ListenAddr  string `json:"listen_addr"`
			StatusPort  int    `json:"status_port"`
		}{
			Name:        "Audio Router Hub",
			Description: "Hub-and-spoke amateur radio audio router",
			ListenAddr:  "0.0.0.0",
			StatusPort:  9090,
		},
		Audio: struct {
			BufferSize       int    `json:"buffer_size"`
			ProcessingDelay  int    `json:"processing_delay"`
			MaxConcurrentTx  int    `json:"max_concurrent_tx"`
			TxTimeoutSeconds int    `json:"tx_timeout_seconds"`
			EnableConversion bool   `json:"enable_conversion"`
			DefaultFormat    string `json:"default_format"`
		}{
			BufferSize:       1000,
			ProcessingDelay:  10,
			MaxConcurrentTx:  3,
			TxTimeoutSeconds: 30,
			EnableConversion: true,
			DefaultFormat:    "opus",
		},
		Routing: struct {
			PreventLoops        bool     `json:"prevent_loops"`
			EnablePriorityRules bool     `json:"enable_priority_rules"`
			DefaultRouting      string   `json:"default_routing"`
			BlockedPairs        []string `json:"blocked_pairs"`
		}{
			PreventLoops:        true,
			EnablePriorityRules: true,
			DefaultRouting:      "all-to-all",
			BlockedPairs:        []string{},
		},
		Amateur: struct {
			StationCall       string `json:"station_call"`
			DefaultTalkGroup  uint32 `json:"default_talk_group"`
			RequireValidCall  bool   `json:"require_valid_call"`
			LogTransmissions  bool   `json:"log_transmissions"`
		}{
			StationCall:       "N0CALL",
			DefaultTalkGroup:  1,
			RequireValidCall:  false,
			LogTransmissions:  true,
		},
		Services: []ServiceInstance{
			{
				ID:          "usrp_1",
				Type:        ServiceTypeUSRP,
				Name:        "AllStarLink Node 1",
				Description: "Primary AllStarLink node",
				Enabled:     true,
				Network: struct {
					Protocol   string `json:"protocol"`
					ListenAddr string `json:"listen_addr"`
					ListenPort int    `json:"listen_port"`
					RemoteAddr string `json:"remote_addr"`
					RemotePort int    `json:"remote_port"`
				}{
					Protocol:   "udp",
					ListenAddr: "0.0.0.0",
					ListenPort: 32001,
					RemoteAddr: "127.0.0.1",
					RemotePort: 34001,
				},
				Audio: struct {
					Format     string `json:"format"`
					SampleRate int    `json:"sample_rate"`
					Channels   int    `json:"channels"`
					Bitrate    int    `json:"bitrate"`
				}{
					Format:     "pcm",
					SampleRate: 8000,
					Channels:   1,
					Bitrate:    64000,
				},
				Routing: struct {
					CanSend        bool     `json:"can_send"`
					CanReceive     bool     `json:"can_receive"`
					SendToTypes    []string `json:"send_to_types"`
					ReceiveFrom    []string `json:"receive_from"`
					ExcludeServices []string `json:"exclude_services"`
					Priority       int      `json:"priority"`
				}{
					CanSend:     true,
					CanReceive:  true,
					SendToTypes: []string{"whotalkie", "discord", "generic"},
					ReceiveFrom: []string{"whotalkie", "discord", "generic"},
					Priority:    5,
				},
			},
		},
	}
}

func generateSampleConfig() {
	config := &AudioRouterConfig{
		Router: struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			ListenAddr  string `json:"listen_addr"`
			StatusPort  int    `json:"status_port"`
		}{
			Name:        "Amateur Radio Audio Router Hub",
			Description: "Hub-and-spoke audio routing for amateur radio services",
			ListenAddr:  "0.0.0.0",
			StatusPort:  9090,
		},
		Audio: struct {
			BufferSize       int    `json:"buffer_size"`
			ProcessingDelay  int    `json:"processing_delay"`
			MaxConcurrentTx  int    `json:"max_concurrent_tx"`
			TxTimeoutSeconds int    `json:"tx_timeout_seconds"`
			EnableConversion bool   `json:"enable_conversion"`
			DefaultFormat    string `json:"default_format"`
		}{
			BufferSize:       1000,
			ProcessingDelay:  10,
			MaxConcurrentTx:  3,
			TxTimeoutSeconds: 30,
			EnableConversion: true,
			DefaultFormat:    "opus",
		},
		Routing: struct {
			PreventLoops        bool     `json:"prevent_loops"`
			EnablePriorityRules bool     `json:"enable_priority_rules"`
			DefaultRouting      string   `json:"default_routing"`
			BlockedPairs        []string `json:"blocked_pairs"`
		}{
			PreventLoops:        true,
			EnablePriorityRules: true,
			DefaultRouting:      "all-to-all",
			BlockedPairs:        []string{},
		},
		Amateur: struct {
			StationCall       string `json:"station_call"`
			DefaultTalkGroup  uint32 `json:"default_talk_group"`
			RequireValidCall  bool   `json:"require_valid_call"`
			LogTransmissions  bool   `json:"log_transmissions"`
		}{
			StationCall:       "W1AW",
			DefaultTalkGroup:  1,
			RequireValidCall:  false,
			LogTransmissions:  true,
		},
		Services: []ServiceInstance{
			{
				ID:          "allstar_1",
				Type:        ServiceTypeUSRP,
				Name:        "AllStarLink Node 12345",
				Description: "Primary AllStarLink node",
				Enabled:     true,
				Network: struct {
					Protocol   string `json:"protocol"`
					ListenAddr string `json:"listen_addr"`
					ListenPort int    `json:"listen_port"`
					RemoteAddr string `json:"remote_addr"`
					RemotePort int    `json:"remote_port"`
				}{
					Protocol:   "udp",
					ListenAddr: "0.0.0.0",
					ListenPort: 32001,
					RemoteAddr: "127.0.0.1",
					RemotePort: 34001,
				},
				Audio: struct {
					Format     string `json:"format"`
					SampleRate int    `json:"sample_rate"`
					Channels   int    `json:"channels"`
					Bitrate    int    `json:"bitrate"`
				}{
					Format:     "pcm",
					SampleRate: 8000,
					Channels:   1,
				},
				Routing: struct {
					CanSend        bool     `json:"can_send"`
					CanReceive     bool     `json:"can_receive"`
					SendToTypes    []string `json:"send_to_types"`
					ReceiveFrom    []string `json:"receive_from"`
					ExcludeServices []string `json:"exclude_services"`
					Priority       int      `json:"priority"`
				}{
					CanSend:     true,
					CanReceive:  true,
					SendToTypes: []string{"whotalkie", "discord"},
					ReceiveFrom: []string{"whotalkie", "discord"},
					Priority:    5,
				},
			},
			{
				ID:          "whotalkie_1",
				Type:        ServiceTypeWhoTalkie,
				Name:        "WhoTalkie Service 1",
				Description: "WhoTalkie internet service",
				Enabled:     true,
				Network: struct {
					Protocol   string `json:"protocol"`
					ListenAddr string `json:"listen_addr"`
					ListenPort int    `json:"listen_port"`
					RemoteAddr string `json:"remote_addr"`
					RemotePort int    `json:"remote_port"`
				}{
					Protocol:   "udp",
					ListenAddr: "0.0.0.0",
					ListenPort: 32002,
					RemoteAddr: "whotalkie.example.com",
					RemotePort: 8080,
				},
				Audio: struct {
					Format     string `json:"format"`
					SampleRate int    `json:"sample_rate"`
					Channels   int    `json:"channels"`
					Bitrate    int    `json:"bitrate"`
				}{
					Format:     "opus",
					SampleRate: 48000,
					Channels:   1,
					Bitrate:    64000,
				},
				Routing: struct {
					CanSend        bool     `json:"can_send"`
					CanReceive     bool     `json:"can_receive"`
					SendToTypes    []string `json:"send_to_types"`
					ReceiveFrom    []string `json:"receive_from"`
					ExcludeServices []string `json:"exclude_services"`
					Priority       int      `json:"priority"`
				}{
					CanSend:     true,
					CanReceive:  true,
					SendToTypes: []string{"usrp", "discord"},
					ReceiveFrom: []string{"usrp", "discord"},
					Priority:    3,
				},
			},
			{
				ID:          "discord_1",
				Type:        ServiceTypeDiscord,
				Name:        "Discord Bridge Bot",
				Description: "Discord voice channel bridge",
				Enabled:     false,
				Settings: map[string]interface{}{
					"bot_token":   "YOUR_DISCORD_BOT_TOKEN",
					"guild_id":    "123456789",
					"channel_id":  "987654321",
					"callsign":    "W1AW",
				},
				Audio: struct {
					Format     string `json:"format"`
					SampleRate int    `json:"sample_rate"`
					Channels   int    `json:"channels"`
					Bitrate    int    `json:"bitrate"`
				}{
					Format:     "pcm",
					SampleRate: 48000,
					Channels:   2,
					Bitrate:    128000,
				},
				Routing: struct {
					CanSend        bool     `json:"can_send"`
					CanReceive     bool     `json:"can_receive"`
					SendToTypes    []string `json:"send_to_types"`
					ReceiveFrom    []string `json:"receive_from"`
					ExcludeServices []string `json:"exclude_services"`
					Priority       int      `json:"priority"`
				}{
					CanSend:     true,
					CanReceive:  true,
					SendToTypes: []string{"usrp", "whotalkie"},
					ReceiveFrom: []string{"usrp", "whotalkie"},
					Priority:    3,
				},
			},
		},
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}
	
	filename := "audio-router.json"
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}
	
	fmt.Printf("✅ Generated sample configuration: %s\n", filename)
	fmt.Println("\n📝 Next steps:")
	fmt.Println("1. Edit the configuration file with your settings")
	fmt.Println("2. Set your amateur radio callsign")
	fmt.Println("3. Configure service endpoints (AllStarLink, WhoTalkie, Discord)")
	fmt.Println("4. Enable the services you want to use")
	fmt.Printf("5. Run: go run cmd/audio-router/main.go -config %s\n", filename)
}