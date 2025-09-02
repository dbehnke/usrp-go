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
	binary.Write(buf, binary.BigEndian, v.Header.Seq)
	binary.Write(buf, binary.BigEndian, v.Header.Memory)
	binary.Write(buf, binary.BigEndian, v.Header.Keyup)
	binary.Write(buf, binary.BigEndian, v.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, v.Header.Type)
	binary.Write(buf, binary.BigEndian, v.Header.MpxID)
	binary.Write(buf, binary.BigEndian, v.Header.Reserved)
	
	// Write 160 audio samples in little-endian (as per specification)
	for _, sample := range v.AudioData {
		binary.Write(buf, binary.LittleEndian, sample)
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
	buf.Read(v.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &v.Header.Seq)
	binary.Read(buf, binary.BigEndian, &v.Header.Memory)
	binary.Read(buf, binary.BigEndian, &v.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &v.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &v.Header.Type)
	binary.Read(buf, binary.BigEndian, &v.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &v.Header.Reserved)
	
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
	binary.Write(buf, binary.BigEndian, d.Header.Seq)
	binary.Write(buf, binary.BigEndian, d.Header.Memory)
	binary.Write(buf, binary.BigEndian, d.Header.Keyup)
	binary.Write(buf, binary.BigEndian, d.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, d.Header.Type)
	binary.Write(buf, binary.BigEndian, d.Header.MpxID)
	binary.Write(buf, binary.BigEndian, d.Header.Reserved)
	
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
	buf.Read(d.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &d.Header.Seq)
	binary.Read(buf, binary.BigEndian, &d.Header.Memory)
	binary.Read(buf, binary.BigEndian, &d.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &d.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &d.Header.Type)
	binary.Read(buf, binary.BigEndian, &d.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &d.Header.Reserved)
	
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
	binary.Write(buf, binary.BigEndian, t.Header.Seq)
	binary.Write(buf, binary.BigEndian, t.Header.Memory)
	binary.Write(buf, binary.BigEndian, t.Header.Keyup)
	binary.Write(buf, binary.BigEndian, t.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, t.Header.Type)
	binary.Write(buf, binary.BigEndian, t.Header.MpxID)
	binary.Write(buf, binary.BigEndian, t.Header.Reserved)
	
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
	buf.Read(t.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &t.Header.Seq)
	binary.Read(buf, binary.BigEndian, &t.Header.Memory)
	binary.Read(buf, binary.BigEndian, &t.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &t.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &t.Header.Type)
	binary.Read(buf, binary.BigEndian, &t.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &t.Header.Reserved)
	
	if err := validateHeader(&t.Header); err != nil {
		return err
	}
	
	// Read remaining text data
	remaining := len(data) - HeaderSize
	if remaining > 0 {
		t.Text = make([]byte, remaining)
		buf.Read(t.Text)
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
	binary.Write(buf, binary.BigEndian, p.Header.Seq)
	binary.Write(buf, binary.BigEndian, p.Header.Memory)
	binary.Write(buf, binary.BigEndian, p.Header.Keyup)
	binary.Write(buf, binary.BigEndian, p.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, p.Header.Type)
	binary.Write(buf, binary.BigEndian, p.Header.MpxID)
	binary.Write(buf, binary.BigEndian, p.Header.Reserved)
	
	return buf.Bytes(), nil
}

// Unmarshal deserializes binary data into PingMessage
func (p *PingMessage) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("data too short for ping message: %d bytes", len(data))
	}
	
	buf := bytes.NewReader(data)
	
	// Read header
	buf.Read(p.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &p.Header.Seq)
	binary.Read(buf, binary.BigEndian, &p.Header.Memory)
	binary.Read(buf, binary.BigEndian, &p.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &p.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &p.Header.Type)
	binary.Read(buf, binary.BigEndian, &p.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &p.Header.Reserved)
	
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
	binary.Write(buf, binary.BigEndian, tlv.Header.Seq)
	binary.Write(buf, binary.BigEndian, tlv.Header.Memory)
	binary.Write(buf, binary.BigEndian, tlv.Header.Keyup)
	binary.Write(buf, binary.BigEndian, tlv.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, tlv.Header.Type)
	binary.Write(buf, binary.BigEndian, tlv.Header.MpxID)
	binary.Write(buf, binary.BigEndian, tlv.Header.Reserved)
	
	// Write TLV items
	for _, item := range tlv.TLVs {
		buf.WriteByte(byte(item.Tag))
		binary.Write(buf, binary.BigEndian, item.Length)
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
	buf.Read(tlv.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &tlv.Header.Seq)
	binary.Read(buf, binary.BigEndian, &tlv.Header.Memory)
	binary.Read(buf, binary.BigEndian, &tlv.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &tlv.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &tlv.Header.Type)
	binary.Read(buf, binary.BigEndian, &tlv.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &tlv.Header.Reserved)
	
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
		binary.Read(buf, binary.BigEndian, &item.Length)
		
		if buf.Len() < int(item.Length) {
			return fmt.Errorf("TLV item length exceeds remaining data")
		}
		
		item.Value = make([]byte, item.Length)
		buf.Read(item.Value)
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
	binary.Write(buf, binary.BigEndian, u.Header.Seq)
	binary.Write(buf, binary.BigEndian, u.Header.Memory)
	binary.Write(buf, binary.BigEndian, u.Header.Keyup)
	binary.Write(buf, binary.BigEndian, u.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, u.Header.Type)
	binary.Write(buf, binary.BigEndian, u.Header.MpxID)
	binary.Write(buf, binary.BigEndian, u.Header.Reserved)
	
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
	buf.Read(u.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &u.Header.Seq)
	binary.Read(buf, binary.BigEndian, &u.Header.Memory)
	binary.Read(buf, binary.BigEndian, &u.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &u.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &u.Header.Type)
	binary.Read(buf, binary.BigEndian, &u.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &u.Header.Reserved)
	
	if err := validateHeader(&u.Header); err != nil {
		return err
	}
	
	// Read μ-law audio data
	buf.Read(u.AudioData[:])
	
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
	binary.Write(buf, binary.BigEndian, a.Header.Seq)
	binary.Write(buf, binary.BigEndian, a.Header.Memory)
	binary.Write(buf, binary.BigEndian, a.Header.Keyup)
	binary.Write(buf, binary.BigEndian, a.Header.TalkGroup)
	binary.Write(buf, binary.BigEndian, a.Header.Type)
	binary.Write(buf, binary.BigEndian, a.Header.MpxID)
	binary.Write(buf, binary.BigEndian, a.Header.Reserved)
	
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
	buf.Read(a.Header.Eye[:])
	binary.Read(buf, binary.BigEndian, &a.Header.Seq)
	binary.Read(buf, binary.BigEndian, &a.Header.Memory)
	binary.Read(buf, binary.BigEndian, &a.Header.Keyup)
	binary.Read(buf, binary.BigEndian, &a.Header.TalkGroup)
	binary.Read(buf, binary.BigEndian, &a.Header.Type)
	binary.Read(buf, binary.BigEndian, &a.Header.MpxID)
	binary.Read(buf, binary.BigEndian, &a.Header.Reserved)
	
	if err := validateHeader(&a.Header); err != nil {
		return err
	}
	
	// Read ADPCM data
	remaining := len(data) - HeaderSize
	if remaining > 0 {
		a.AudioData = make([]byte, remaining)
		buf.Read(a.AudioData)
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