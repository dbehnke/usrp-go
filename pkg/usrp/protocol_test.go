package usrp

import (
	"testing"
)

func TestVoiceMessage_MarshalUnmarshal(t *testing.T) {
	original := &VoiceMessage{
		Header: NewHeader(USRP_TYPE_VOICE, 1234),
	}
	original.Header.SetPTT(true)
	original.Header.TalkGroup = 5678

	// Fill audio data with test pattern
	for i := range original.AudioData {
		original.AudioData[i] = int16(i % 32767)
	}

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Should be header (32 bytes) + audio (320 bytes) = 352 bytes
	expectedSize := HeaderSize + VoiceFrameSize*2
	if len(data) != expectedSize {
		t.Errorf("Unexpected data size: got %d, want %d", len(data), expectedSize)
	}

	// Unmarshal
	decoded := &VoiceMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if string(decoded.Header.Eye[:]) != USRPMagic {
		t.Errorf("Magic mismatch: got %s, want %s", string(decoded.Header.Eye[:]), USRPMagic)
	}
	if decoded.Header.Seq != original.Header.Seq {
		t.Errorf("Sequence mismatch: got %d, want %d", decoded.Header.Seq, original.Header.Seq)
	}
	if decoded.Header.Keyup != original.Header.Keyup {
		t.Errorf("Keyup mismatch: got %d, want %d", decoded.Header.Keyup, original.Header.Keyup)
	}
	if decoded.Header.TalkGroup != original.Header.TalkGroup {
		t.Errorf("TalkGroup mismatch: got %d, want %d", decoded.Header.TalkGroup, original.Header.TalkGroup)
	}
	if decoded.Header.Type != original.Header.Type {
		t.Errorf("Type mismatch: got %d, want %d", decoded.Header.Type, original.Header.Type)
	}

	// Verify audio data
	for i, sample := range decoded.AudioData {
		if sample != original.AudioData[i] {
			t.Errorf("AudioData[%d] mismatch: got %d, want %d", i, sample, original.AudioData[i])
		}
	}
}

func TestDTMFMessage_MarshalUnmarshal(t *testing.T) {
	original := &DTMFMessage{
		Header: NewHeader(USRP_TYPE_DTMF, 5555),
		Digit:  '5',
	}
	original.Header.SetPTT(false)

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Should be header (32 bytes) + digit (1 byte) = 33 bytes
	expectedSize := HeaderSize + 1
	if len(data) != expectedSize {
		t.Errorf("Unexpected data size: got %d, want %d", len(data), expectedSize)
	}

	// Unmarshal
	decoded := &DTMFMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.Header.Seq != original.Header.Seq {
		t.Errorf("Sequence mismatch: got %d, want %d", decoded.Header.Seq, original.Header.Seq)
	}
	if decoded.Digit != original.Digit {
		t.Errorf("Digit mismatch: got %c, want %c", decoded.Digit, original.Digit)
	}
	if decoded.Header.IsPTT() != false {
		t.Errorf("PTT should be false")
	}
}

func TestTextMessage_MarshalUnmarshal(t *testing.T) {
	testText := "Hello, USRP!"
	original := &TextMessage{
		Header: NewHeader(USRP_TYPE_TEXT, 7777),
		Text:   []byte(testText),
	}

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	decoded := &TextMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.Header.Seq != original.Header.Seq {
		t.Errorf("Sequence mismatch: got %d, want %d", decoded.Header.Seq, original.Header.Seq)
	}
	if string(decoded.Text) != testText {
		t.Errorf("Text mismatch: got %s, want %s", string(decoded.Text), testText)
	}
}

func TestPingMessage_MarshalUnmarshal(t *testing.T) {
	original := &PingMessage{
		Header: NewHeader(USRP_TYPE_PING, 9999),
	}

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Should be exactly header size (no payload)
	if len(data) != HeaderSize {
		t.Errorf("Unexpected data size: got %d, want %d", len(data), HeaderSize)
	}

	// Unmarshal
	decoded := &PingMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.Header.Seq != original.Header.Seq {
		t.Errorf("Sequence mismatch: got %d, want %d", decoded.Header.Seq, original.Header.Seq)
	}
	if PacketType(decoded.Header.Type) != USRP_TYPE_PING {
		t.Errorf("Type should be PING")
	}
}

func TestTLVMessage_MarshalUnmarshal(t *testing.T) {
	original := &TLVMessage{
		Header: NewHeader(USRP_TYPE_TLV, 1111),
	}

	// Add some TLV items
	original.SetCallsign("W1AW")
	original.AddTLV(TLV_TAG_AMBE, []byte{0x01, 0x02, 0x03})

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	decoded := &TLVMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify callsign
	callsign, ok := decoded.GetCallsign()
	if !ok {
		t.Error("Callsign not found in TLV data")
	}
	if callsign != "W1AW" {
		t.Errorf("Callsign mismatch: got %s, want W1AW", callsign)
	}

	// Verify AMBE data
	ambeData, ok := decoded.GetTLV(TLV_TAG_AMBE)
	if !ok {
		t.Error("AMBE data not found in TLV")
	}
	expected := []byte{0x01, 0x02, 0x03}
	if len(ambeData) != len(expected) {
		t.Errorf("AMBE data length mismatch: got %d, want %d", len(ambeData), len(expected))
	}
	for i, b := range ambeData {
		if b != expected[i] {
			t.Errorf("AMBE data[%d] mismatch: got 0x%02x, want 0x%02x", i, b, expected[i])
		}
	}
}

func TestVoiceULawMessage_MarshalUnmarshal(t *testing.T) {
	original := &VoiceULawMessage{
		Header: NewHeader(USRP_TYPE_VOICE_ULAW, 2222),
	}

	// Fill with μ-law test pattern
	for i := range original.AudioData {
		original.AudioData[i] = byte(i % 256)
	}

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Should be header + 160 μ-law bytes
	expectedSize := HeaderSize + VoiceFrameSize
	if len(data) != expectedSize {
		t.Errorf("Unexpected data size: got %d, want %d", len(data), expectedSize)
	}

	// Unmarshal
	decoded := &VoiceULawMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify audio data
	for i, sample := range decoded.AudioData {
		if sample != original.AudioData[i] {
			t.Errorf("AudioData[%d] mismatch: got %d, want %d", i, sample, original.AudioData[i])
		}
	}
}

func TestVoiceADPCMMessage_MarshalUnmarshal(t *testing.T) {
	testADPCM := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	original := &VoiceADPCMMessage{
		Header:    NewHeader(USRP_TYPE_VOICE_ADPCM, 3333),
		AudioData: testADPCM,
	}

	// Marshal
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	decoded := &VoiceADPCMMessage{}
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify audio data
	if len(decoded.AudioData) != len(testADPCM) {
		t.Errorf("AudioData length mismatch: got %d, want %d", len(decoded.AudioData), len(testADPCM))
	}
	for i, sample := range decoded.AudioData {
		if sample != testADPCM[i] {
			t.Errorf("AudioData[%d] mismatch: got 0x%02x, want 0x%02x", i, sample, testADPCM[i])
		}
	}
}

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
	}{
		{
			name: "valid voice message",
			msg: &VoiceMessage{
				Header: NewHeader(USRP_TYPE_VOICE, 1),
			},
			wantErr: false,
		},
		{
			name: "voice message with wrong type",
			msg: &VoiceMessage{
				Header: Header{Type: uint32(USRP_TYPE_DTMF)}, // Wrong type
			},
			wantErr: true,
		},
		{
			name: "valid DTMF message",
			msg: &DTMFMessage{
				Header: NewHeader(USRP_TYPE_DTMF, 1),
				Digit:  '5',
			},
			wantErr: false,
		},
		{
			name: "DTMF with invalid digit",
			msg: &DTMFMessage{
				Header: NewHeader(USRP_TYPE_DTMF, 1),
				Digit:  'X', // Invalid DTMF digit
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Message.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaderOperations(t *testing.T) {
	h := NewHeader(USRP_TYPE_VOICE, 42)

	// Test magic string
	if string(h.Eye[:]) != USRPMagic {
		t.Errorf("Magic mismatch: got %s, want %s", string(h.Eye[:]), USRPMagic)
	}

	// Test sequence
	if h.Seq != 42 {
		t.Errorf("Sequence mismatch: got %d, want 42", h.Seq)
	}

	// Test packet type
	if PacketType(h.Type) != USRP_TYPE_VOICE {
		t.Errorf("Type mismatch: got %d, want %d", h.Type, USRP_TYPE_VOICE)
	}

	// Test PTT operations
	if h.IsPTT() {
		t.Error("PTT should initially be false")
	}

	h.SetPTT(true)
	if !h.IsPTT() {
		t.Error("PTT should be true after setting")
	}

	h.SetPTT(false)
	if h.IsPTT() {
		t.Error("PTT should be false after clearing")
	}
}

func TestInvalidPacket(t *testing.T) {
	// Test unmarshaling invalid data
	invalidData := []byte{0x00, 0x01, 0x02} // Too short

	msg := &VoiceMessage{}
	err := msg.Unmarshal(invalidData)
	if err == nil {
		t.Error("Expected error for invalid data, got nil")
	}
}

func BenchmarkVoiceMessage_Marshal(b *testing.B) {
	msg := &VoiceMessage{
		Header: NewHeader(USRP_TYPE_VOICE, 1),
	}
	// Fill with test pattern
	for i := range msg.AudioData {
		msg.AudioData[i] = int16(i % 32767)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := msg.Marshal()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVoiceMessage_Unmarshal(b *testing.B) {
	msg := &VoiceMessage{
		Header: NewHeader(USRP_TYPE_VOICE, 1),
	}
	for i := range msg.AudioData {
		msg.AudioData[i] = int16(i % 32767)
	}

	data, _ := msg.Marshal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoded := &VoiceMessage{}
		err := decoded.Unmarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
