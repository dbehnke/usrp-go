package transport

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// Connection interface defines the contract for USRP connections
type Connection interface {
	Connect() error
	SendMessage(usrp.Message) error
	ReceiveMessage() (usrp.Message, error)
	RegisterHandler(usrp.PacketType, MessageHandler)
	Start(context.Context) error
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// MessageHandler function type for handling received messages
type MessageHandler func(usrp.Message) error

// UDPConnection implements Connection interface using UDP transport
type UDPConnection struct {
	conn         *net.UDPConn
	localAddr    *net.UDPAddr
	remoteAddr   *net.UDPAddr
	handlers     map[usrp.PacketType]MessageHandler
	handlerMutex sync.RWMutex
	sequenceNum  uint32
	seqMutex     sync.Mutex
	bufferPool   sync.Pool
	closed       bool
	closeMutex   sync.Mutex
}

// ConnectionConfig holds configuration for UDP connections
type ConnectionConfig struct {
	LocalAddr       string
	RemoteAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ReadBufferSize  int
	WriteBufferSize int
}

// DefaultConfig returns a default connection configuration
func DefaultConfig() *ConnectionConfig {
	return &ConnectionConfig{
		LocalAddr:       ":0",
		RemoteAddr:      "",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 64 * 1024,
	}
}

// NewUDPConnection creates a new UDP connection with the given configuration
func NewUDPConnection(config *ConnectionConfig) (*UDPConnection, error) {
	if config == nil {
		config = DefaultConfig()
	}

	localAddr, err := net.ResolveUDPAddr("udp", config.LocalAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve local address: %w", err)
	}

	var remoteAddr *net.UDPAddr
	if config.RemoteAddr != "" {
		remoteAddr, err = net.ResolveUDPAddr("udp", config.RemoteAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve remote address: %w", err)
		}
	}

	uc := &UDPConnection{
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		handlers:   make(map[usrp.PacketType]MessageHandler),
		bufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, usrp.MaxPayloadSize+64) // Header + max payload
				return &buf
			},
		},
	}

	return uc, nil
}

// Connect establishes the UDP connection
func (uc *UDPConnection) Connect() error {
	uc.closeMutex.Lock()
	defer uc.closeMutex.Unlock()

	if uc.closed {
		return fmt.Errorf("connection is closed")
	}

	conn, err := net.ListenUDP("udp", uc.localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	uc.conn = conn
	uc.localAddr = conn.LocalAddr().(*net.UDPAddr)

	return nil
}

// SendMessage sends a USRP message over UDP
func (uc *UDPConnection) SendMessage(msg usrp.Message) error {
	if uc.conn == nil {
		return fmt.Errorf("connection not established")
	}

	if uc.remoteAddr == nil {
		return fmt.Errorf("no remote address configured")
	}

	// Validate message
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Set sequence number
	uc.seqMutex.Lock()
	uc.sequenceNum++
	seq := uc.sequenceNum
	uc.seqMutex.Unlock()

	// Set sequence number in message header
	switch m := msg.(type) {
	case *usrp.VoiceMessage:
		m.Header.Seq = seq
	case *usrp.DTMFMessage:
		m.Header.Seq = seq
	case *usrp.TextMessage:
		m.Header.Seq = seq
	case *usrp.PingMessage:
		m.Header.Seq = seq
	case *usrp.TLVMessage:
		m.Header.Seq = seq
	case *usrp.VoiceULawMessage:
		m.Header.Seq = seq
	case *usrp.VoiceADPCMMessage:
		m.Header.Seq = seq
	}

	// Marshal message
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send data
	_, err = uc.conn.WriteToUDP(data, uc.remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to send UDP packet: %w", err)
	}

	return nil
}

// ReceiveMessage receives and parses a USRP message from UDP
func (uc *UDPConnection) ReceiveMessage() (usrp.Message, error) {
	if uc.conn == nil {
		return nil, fmt.Errorf("connection not established")
	}

	// Get buffer from pool
	bufferPtr := uc.bufferPool.Get().(*[]byte)
	buffer := *bufferPtr

	// Read from UDP
	n, addr, err := uc.conn.ReadFromUDP(buffer)
	if err != nil {
		uc.bufferPool.Put(bufferPtr)
		return nil, fmt.Errorf("failed to read UDP packet: %w", err)
	}

	// Update remote address if not set
	if uc.remoteAddr == nil {
		uc.remoteAddr = addr
	}

	// Parse packet type from header
	if n < usrp.HeaderSize { // Minimum header size is 32 bytes
		uc.bufferPool.Put(bufferPtr)
		return nil, fmt.Errorf("packet too small: %d bytes", n)
	}

	// Packet type is at offset 20 in the 32-byte header (after Eye, Seq, Memory, Keyup, TalkGroup)
	packetType := usrp.PacketType(binary.BigEndian.Uint32(buffer[20:24]))

	// Create appropriate message type and unmarshal
	var msg usrp.Message
	switch packetType {
	case usrp.USRP_TYPE_VOICE:
		msg = &usrp.VoiceMessage{}
	case usrp.USRP_TYPE_DTMF:
		msg = &usrp.DTMFMessage{}
	case usrp.USRP_TYPE_TEXT:
		msg = &usrp.TextMessage{}
	case usrp.USRP_TYPE_PING:
		msg = &usrp.PingMessage{}
	case usrp.USRP_TYPE_TLV:
		msg = &usrp.TLVMessage{}
	case usrp.USRP_TYPE_VOICE_ULAW:
		msg = &usrp.VoiceULawMessage{}
	case usrp.USRP_TYPE_VOICE_ADPCM:
		msg = &usrp.VoiceADPCMMessage{}
	default:
		uc.bufferPool.Put(bufferPtr)
		return nil, fmt.Errorf("unknown packet type: %d", packetType)
	}

	if err := msg.Unmarshal(buffer[:n]); err != nil {
		uc.bufferPool.Put(bufferPtr)
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	uc.bufferPool.Put(bufferPtr)
	return msg, nil
}

// RegisterHandler registers a handler function for a specific packet type
func (uc *UDPConnection) RegisterHandler(packetType usrp.PacketType, handler MessageHandler) {
	uc.handlerMutex.Lock()
	defer uc.handlerMutex.Unlock()
	uc.handlers[packetType] = handler
}

// Start begins the message processing loop
func (uc *UDPConnection) Start(ctx context.Context) error {
	if uc.conn == nil {
		return fmt.Errorf("connection not established")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Set read timeout
			if err := uc.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				return fmt.Errorf("failed to set read deadline: %w", err)
			}

			msg, err := uc.ReceiveMessage()
			if err != nil {
				// Check if it's a timeout
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("failed to receive message: %w", err)
			}

			// Handle message
			uc.handlerMutex.RLock()
			handler, exists := uc.handlers[msg.GetType()]
			uc.handlerMutex.RUnlock()

			if exists {
				go func() {
					if err := handler(msg); err != nil {
						// In a production system, you'd want proper logging here
						fmt.Printf("Handler error: %v\n", err)
					}
				}()
			}
		}
	}
}

// Close closes the UDP connection and cleans up resources
func (uc *UDPConnection) Close() error {
	uc.closeMutex.Lock()
	defer uc.closeMutex.Unlock()

	if uc.closed {
		return nil
	}

	uc.closed = true

	if uc.conn != nil {
		return uc.conn.Close()
	}

	return nil
}

// LocalAddr returns the local network address
func (uc *UDPConnection) LocalAddr() net.Addr {
	if uc.conn != nil {
		return uc.conn.LocalAddr()
	}
	return uc.localAddr
}

// RemoteAddr returns the remote network address
func (uc *UDPConnection) RemoteAddr() net.Addr {
	return uc.remoteAddr
}
