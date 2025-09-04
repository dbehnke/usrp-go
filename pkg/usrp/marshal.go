package usrp

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Marshal serializes VoiceMessage to binary format (network byte order)
func (v *VoiceMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write 32-byte header in network byte order (big-endian)
	buf.Write(v.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, v.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, v.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, v.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, v.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, v.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, v.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, v.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	// Write 160 audio samples in little-endian (as per specification)
	for i, sample := range v.AudioData {
		if err := binary.Write(buf, binary.LittleEndian, sample); err != nil {
			return nil, fmt.Errorf("error writing sample %d: %w", i, err)
		}
	}
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into VoiceMessage
func (v *VoiceMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short: %d bytes (need at least %d)", len(data), HeaderSize)
	}
	
	buf := bytes.NewReader(data)
	
	// Read 32-byte header in network byte order
	if _, err := buf.Read(v.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &v.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	// Validate header
	if err := validateHeader(&v.Header); err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}
	
	// Read audio samples in little-endian (160 samples = 320 bytes)
	expectedAudioSize := VoiceFrameSize * 2 // 2 bytes per sample
	if len(data) < HeaderSize+expectedAudioSize {
		return fmt.Errorf("insufficient audio data: got %d bytes, need %d", 
			len(data)-HeaderSize, expectedAudioSize)
	}
	
	for i := range v.AudioData {
		if err := binary.Read(buf, binary.LittleEndian, &v.AudioData[i]); err != nil {
			return fmt.Errorf("failed to read audio sample %d: %w", i, err)
		}
	}
	
	return nil
}

// Validate checks VoiceMessage for consistency
func (v *VoiceMessage) Validate() error {
	if PacketType(v.Header.Type) != USRP_TYPE_VOICE {
		return fmt.Errorf("invalid packet type for voice message: %d", v.Header.Type)
	}
	return nil
}

// Marshal serializes DTMFMessage to binary format
func (d *DTMFMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write 32-byte header
	buf.Write(d.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, d.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, d.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, d.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, d.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, d.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, d.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, d.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	// Write DTMF digit
	buf.WriteByte(d.Digit)
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into DTMFMessage
func (d *DTMFMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize+1 {
		return fmt.Errorf("data too short for DTMF message: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	if _, err := buf.Read(d.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &d.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	if err := validateHeader(&d.Header); err != nil {
		return err
	}
	
	// Read DTMF digit
	var err error
	d.Digit, err = buf.ReadByte()
	return err
}

// Validate checks DTMFMessage for consistency
func (d *DTMFMessage) Validate() error {
	if PacketType(d.Header.Type) != USRP_TYPE_DTMF {
		return fmt.Errorf("invalid packet type for DTMF message: %d", d.Header.Type)
	}
	
	// Validate DTMF digit
	valid := (d.Digit >= '0' && d.Digit <= '9') ||
		(d.Digit >= 'A' && d.Digit <= 'D') ||
		d.Digit == '*' || d.Digit == '#'
	
	if !valid {
		return fmt.Errorf("invalid DTMF digit: %c", d.Digit)
	}
	
	return nil
}

// Marshal serializes TextMessage to binary format
func (t *TextMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write header
	buf.Write(t.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, t.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	// Write text data
	buf.Write(t.Text)
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into TextMessage
func (t *TextMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short for text message: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	if _, err := buf.Read(t.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	if err := validateHeader(&t.Header); err != nil {
		return err
	}
	
	// Read remaining text data
	remaining := len(data) - HeaderSize
	if remaining > 0 {
		t.Text = make([]byte, remaining)
		if _, err := buf.Read(t.Text); err != nil {
			return fmt.Errorf("error reading text: %w", err)
		}
	}
	
	return nil
}

// Validate checks TextMessage for consistency
func (t *TextMessage) Validate() error {
	if PacketType(t.Header.Type) != USRP_TYPE_TEXT {
		return fmt.Errorf("invalid packet type for text message: %d", t.Header.Type)
	}
	return nil
}

// Marshal serializes PingMessage to binary format
func (p *PingMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write header (ping has no payload)
	buf.Write(p.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, p.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, p.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into PingMessage
func (p *PingMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short for ping message: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	if _, err := buf.Read(p.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &p.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	return validateHeader(&p.Header)
}

// Validate checks PingMessage for consistency
func (p *PingMessage) Validate() error {
	if PacketType(p.Header.Type) != USRP_TYPE_PING {
		return fmt.Errorf("invalid packet type for ping message: %d", p.Header.Type)
	}
	return nil
}

// Marshal serializes TLVMessage to binary format
func (tlv *TLVMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write header
	buf.Write(tlv.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, tlv.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	// Write TLV items
	for _, item := range tlv.TLVs {
		buf.WriteByte(byte(item.Tag))
		if err := binary.Write(buf, binary.BigEndian, item.Length); err != nil {
			return nil, fmt.Errorf("error writing tlv length: %w", err)
		}
		buf.Write(item.Value)
	}
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into TLVMessage
func (tlv *TLVMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short for TLV message: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	if _, err := buf.Read(tlv.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &tlv.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	if err := validateHeader(&tlv.Header); err != nil {
		return err
	}
	
	// Parse TLV items
	tlv.TLVs = nil
	for buf.Len() > 0 {
		if buf.Len() < 3 { // Need at least tag(1) + length(2)
			break
		}
		
		var item TLVItem
		tag, _ := buf.ReadByte()
		item.Tag = TLVTag(tag)
		if err := binary.Read(buf, binary.BigEndian, &item.Length); err != nil {
			return fmt.Errorf("error reading tlv length: %w", err)
		}
		
		if buf.Len() < int(item.Length) {
			return fmt.Errorf("TLV item length exceeds remaining data")
		}
		
		item.Value = make([]byte, item.Length)
		if _, err := buf.Read(item.Value); err != nil {
			return fmt.Errorf("error reading TLV value: %w", err)
		}
		tlv.TLVs = append(tlv.TLVs, item)
	}
	
	return nil
}

// Validate checks TLVMessage for consistency
func (tlv *TLVMessage) Validate() error {
	if PacketType(tlv.Header.Type) != USRP_TYPE_TLV {
		return fmt.Errorf("invalid packet type for TLV message: %d", tlv.Header.Type)
	}
	return nil
}

// Marshal serializes VoiceULawMessage to binary format
func (u *VoiceULawMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write header
	buf.Write(u.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, u.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, u.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, u.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, u.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, u.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, u.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, u.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	// Write μ-law samples
	buf.Write(u.AudioData[:])
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into VoiceULawMessage
func (u *VoiceULawMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize+VoiceFrameSize {
		return fmt.Errorf("data too short for μ-law voice: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	if _, err := buf.Read(u.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &u.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	if err := validateHeader(&u.Header); err != nil {
		return err
	}
	
	// Read μ-law audio data
	if _, err := buf.Read(u.AudioData[:]); err != nil {
		return fmt.Errorf("error reading audio data: %w", err)
	}
	
	return nil
}

// Validate checks VoiceULawMessage for consistency
func (u *VoiceULawMessage) Validate() error {
	if PacketType(u.Header.Type) != USRP_TYPE_VOICE_ULAW {
		return fmt.Errorf("invalid packet type for μ-law voice message: %d", u.Header.Type)
	}
	return nil
}

// Marshal serializes VoiceADPCMMessage to binary format
func (a *VoiceADPCMMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write header
	buf.Write(a.Header.Eye[:])
	if err := binary.Write(buf, binary.BigEndian, a.Header.Seq); err != nil {
		return nil, fmt.Errorf("error writing seq: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, a.Header.Memory); err != nil {
		return nil, fmt.Errorf("error writing memory: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, a.Header.Keyup); err != nil {
		return nil, fmt.Errorf("error writing keyup: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, a.Header.TalkGroup); err != nil {
		return nil, fmt.Errorf("error writing talk group: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, a.Header.Type); err != nil {
		return nil, fmt.Errorf("error writing type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, a.Header.MpxID); err != nil {
		return nil, fmt.Errorf("error writing mpx id: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, a.Header.Reserved); err != nil {
		return nil, fmt.Errorf("error writing reserved: %w", err)
	}
	
	// Write ADPCM data
	buf.Write(a.AudioData)
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into VoiceADPCMMessage
func (a *VoiceADPCMMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short for ADPCM voice: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	if _, err := buf.Read(a.Header.Eye[:]); err != nil {
		return fmt.Errorf("error reading eye: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.Seq); err != nil {
		return fmt.Errorf("error reading seq: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.Memory); err != nil {
		return fmt.Errorf("error reading memory: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.Keyup); err != nil {
		return fmt.Errorf("error reading keyup: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.TalkGroup); err != nil {
		return fmt.Errorf("error reading talk group: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.Type); err != nil {
		return fmt.Errorf("error reading type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.MpxID); err != nil {
		return fmt.Errorf("error reading mpx id: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &a.Header.Reserved); err != nil {
		return fmt.Errorf("error reading reserved: %w", err)
	}
	
	if err := validateHeader(&a.Header); err != nil {
		return err
	}
	
	// Read ADPCM data
	remaining := len(data) - HeaderSize
	if remaining > 0 {
		a.AudioData = make([]byte, remaining)
		if _, err := buf.Read(a.AudioData); err != nil {
			return fmt.Errorf("error reading ADPCM data: %w", err)
		}
	}
	
	return nil
}

// Validate checks VoiceADPCMMessage for consistency
func (a *VoiceADPCMMessage) Validate() error {
	if PacketType(a.Header.Type) != USRP_TYPE_VOICE_ADPCM {
		return fmt.Errorf("invalid packet type for ADPCM voice message: %d", a.Header.Type)
	}
	return nil
}