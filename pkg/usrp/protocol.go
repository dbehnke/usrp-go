// Package usrp provides a Go library for handling USRP (Universal Software Radio Protocol)
// packets used in amateur radio digital voice systems.
//
// This implementation is compatible with AllStarLink's chan_usrp.c and the official
// USRP protocol specification used in amateur radio applications.
package usrp

import (
	"fmt"
)

// Protocol constants based on official specification
const (
	USRPMagic      = "USRP" // 4-byte magic string
	HeaderSize     = 32     // Fixed 32-byte header
	VoiceFrameSize = 160    // 160 samples per voice frame (20ms at 8kHz)
	MaxPayloadSize = 1024   // Maximum payload size
)

// PacketType defines the type of USRP packet (matches official enum)
type PacketType uint32

const (
	USRP_TYPE_VOICE       PacketType = 0 // Voice audio data
	USRP_TYPE_DTMF        PacketType = 1 // DTMF signaling
	USRP_TYPE_TEXT        PacketType = 2 // Text/metadata
	USRP_TYPE_PING        PacketType = 3 // Ping/keepalive
	USRP_TYPE_TLV         PacketType = 4 // TLV (Type-Length-Value) data
	USRP_TYPE_VOICE_ADPCM PacketType = 5 // ADPCM voice
	USRP_TYPE_VOICE_ULAW  PacketType = 6 // μ-law voice
)

// TLV Tags for metadata (from specification)
type TLVTag uint8

const (
	TLV_TAG_SET_INFO TLVTag = 0x08 // Primary metadata tag
	TLV_TAG_AMBE     TLVTag = 0x01 // AMBE vocoder data
	TLV_TAG_DTMF     TLVTag = 0x02 // DTMF tone
)

// Header represents the official USRP packet header (32 bytes)
// Based on AllStarLink's _chan_usrp_bufhdr structure
type Header struct {
	Eye       [4]byte // "USRP" magic string
	Seq       uint32  // Sequence counter
	Memory    uint32  // Memory ID or zero (default)
	Keyup     uint32  // PTT state (1 = ON, 0 = OFF)
	TalkGroup uint32  // Trunk TG ID
	Type      uint32  // Packet type
	MpxID     uint32  // Future use
	Reserved  uint32  // Future use
}

// Message interface defines common operations for all USRP messages
type Message interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	GetType() PacketType
	Validate() error
}

// VoiceMessage represents voice audio data (USRP_TYPE_VOICE)
type VoiceMessage struct {
	Header    Header
	AudioData [VoiceFrameSize]int16 // 160 signed 16-bit samples, little-endian
}

// DTMFMessage represents DTMF signaling (USRP_TYPE_DTMF)
type DTMFMessage struct {
	Header Header
	Digit  byte // DTMF digit ('0'-'9', 'A'-'D', '*', '#')
}

// TextMessage represents text/metadata (USRP_TYPE_TEXT)
type TextMessage struct {
	Header Header
	Text   []byte // Text data
}

// PingMessage represents ping/keepalive (USRP_TYPE_PING)
type PingMessage struct {
	Header Header
}

// TLVMessage represents TLV data (USRP_TYPE_TLV)
type TLVMessage struct {
	Header Header
	TLVs   []TLVItem
}

// TLVItem represents a Type-Length-Value item
type TLVItem struct {
	Tag    TLVTag
	Length uint16
	Value  []byte
}

// VoiceULawMessage represents μ-law voice (USRP_TYPE_VOICE_ULAW)
type VoiceULawMessage struct {
	Header    Header
	AudioData [VoiceFrameSize]byte // 160 μ-law samples
}

// VoiceADPCMMessage represents ADPCM voice (USRP_TYPE_VOICE_ADPCM)
type VoiceADPCMMessage struct {
	Header    Header
	AudioData []byte // Variable length ADPCM data
}

// GetType implementations
func (v *VoiceMessage) GetType() PacketType      { return USRP_TYPE_VOICE }
func (d *DTMFMessage) GetType() PacketType       { return USRP_TYPE_DTMF }
func (t *TextMessage) GetType() PacketType       { return USRP_TYPE_TEXT }
func (p *PingMessage) GetType() PacketType       { return USRP_TYPE_PING }
func (tlv *TLVMessage) GetType() PacketType      { return USRP_TYPE_TLV }
func (u *VoiceULawMessage) GetType() PacketType  { return USRP_TYPE_VOICE_ULAW }
func (a *VoiceADPCMMessage) GetType() PacketType { return USRP_TYPE_VOICE_ADPCM }

// validateHeader checks header integrity
func validateHeader(h *Header) error {
	if string(h.Eye[:]) != USRPMagic {
		return fmt.Errorf("invalid magic string: got %s, expected %s", string(h.Eye[:]), USRPMagic)
	}
	return nil
}

// NewHeader creates a new USRP header with default values
func NewHeader(packetType PacketType, seq uint32) Header {
	h := Header{
		Seq:  seq,
		Type: uint32(packetType),
	}
	copy(h.Eye[:], USRPMagic)
	return h
}

// SetPTT sets the PTT (Push-To-Talk) state
func (h *Header) SetPTT(on bool) {
	if on {
		h.Keyup = 1
	} else {
		h.Keyup = 0
	}
}

// IsPTT returns true if PTT is active
func (h *Header) IsPTT() bool {
	return h.Keyup != 0
}

// Helper functions for common operations

// SetCallsign sets a callsign in a TLV message
func (tlv *TLVMessage) SetCallsign(callsign string) {
	tlv.AddTLV(TLV_TAG_SET_INFO, []byte(callsign))
}

// AddTLV adds a TLV item to the message
func (tlv *TLVMessage) AddTLV(tag TLVTag, value []byte) {
	tlv.TLVs = append(tlv.TLVs, TLVItem{
		Tag:    tag,
		Length: uint16(len(value)),
		Value:  make([]byte, len(value)),
	})
	copy(tlv.TLVs[len(tlv.TLVs)-1].Value, value)
}

// GetTLV retrieves the first TLV item with the specified tag
func (tlv *TLVMessage) GetTLV(tag TLVTag) ([]byte, bool) {
	for _, item := range tlv.TLVs {
		if item.Tag == tag {
			return item.Value, true
		}
	}
	return nil, false
}

// GetCallsign retrieves the callsign from a TLV message
func (tlv *TLVMessage) GetCallsign() (string, bool) {
	if value, ok := tlv.GetTLV(TLV_TAG_SET_INFO); ok {
		return string(value), true
	}
	return "", false
}
