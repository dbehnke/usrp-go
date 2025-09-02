// Package audio provides audio conversion utilities for USRP packets
// using FFmpeg for format conversion between PCM and compressed formats like Opus/Ogg
package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/dbehnke/usrp-go/pkg/usrp"
)

// Converter interface defines audio format conversion operations
type Converter interface {
	// Convert USRP voice packets to target format
	USRPToFormat(voiceMsg *usrp.VoiceMessage) ([]byte, error)
	
	// Convert target format data to USRP voice packets
	FormatToUSRP(data []byte) ([]*usrp.VoiceMessage, error)
	
	// Close and cleanup resources
	Close() error
}

// StreamingConverter handles real-time audio conversion using FFmpeg
type StreamingConverter struct {
	inputFormat   string        // FFmpeg input format (e.g., "s16le", "opus")
	outputFormat  string        // FFmpeg output format  
	inputRate     int           // Input sample rate
	outputRate    int           // Output sample rate
	channels      int           // Number of audio channels
	
	// FFmpeg processes for bidirectional conversion
	toFormatCmd    *exec.Cmd     // USRP -> Target format
	fromFormatCmd  *exec.Cmd     // Target format -> USRP
	
	toFormatIn     io.WriteCloser
	toFormatOut    io.ReadCloser
	fromFormatIn   io.WriteCloser
	fromFormatOut  io.ReadCloser
	
	// Buffers for handling streaming data
	pcmBuffer      []int16       // Accumulate PCM samples
	formatBuffer   bytes.Buffer  // Accumulate format data
	
	mutex          sync.Mutex    // Thread safety
	closed         bool
}

// ConverterConfig holds configuration for audio conversion
type ConverterConfig struct {
	InputFormat   string        // "s16le", "opus", "ogg", etc.
	OutputFormat  string        
	InputRate     int           // Sample rate (8000 for USRP)
	OutputRate    int
	Channels      int           // 1 for mono (USRP default)
	BitRate       int           // For compressed formats (kbps)
	FrameSize     time.Duration // Audio frame duration
}

// NewOpusConverter creates a converter for USRP <-> Opus conversion
func NewOpusConverter() (*StreamingConverter, error) {
	config := &ConverterConfig{
		InputFormat:  "s16le",
		OutputFormat: "opus",
		InputRate:    8000,  // USRP standard
		OutputRate:   8000,
		Channels:     1,     // Mono
		BitRate:      64,    // 64 kbps
		FrameSize:    20 * time.Millisecond, // 20ms frames (matches USRP)
	}
	return NewStreamingConverter(config)
}

// NewOggOpusConverter creates a converter for USRP <-> Ogg/Opus conversion
func NewOggOpusConverter() (*StreamingConverter, error) {
	config := &ConverterConfig{
		InputFormat:  "s16le",
		OutputFormat: "ogg",
		InputRate:    8000,
		OutputRate:   8000,
		Channels:     1,
		BitRate:      64,
		FrameSize:    20 * time.Millisecond,
	}
	return NewStreamingConverter(config)
}

// NewStreamingConverter creates a new streaming audio converter
func NewStreamingConverter(config *ConverterConfig) (*StreamingConverter, error) {
	sc := &StreamingConverter{
		inputFormat:  config.InputFormat,
		outputFormat: config.OutputFormat,
		inputRate:    config.InputRate,
		outputRate:   config.OutputRate,
		channels:     config.Channels,
		pcmBuffer:    make([]int16, 0, usrp.VoiceFrameSize*4), // Buffer multiple frames
	}
	
	// Initialize FFmpeg processes for both directions
	if err := sc.initFFmpegProcesses(config); err != nil {
		return nil, fmt.Errorf("failed to initialize FFmpeg: %w", err)
	}
	
	return sc, nil
}

// initFFmpegProcesses sets up FFmpeg processes for bidirectional conversion
func (sc *StreamingConverter) initFFmpegProcesses(config *ConverterConfig) error {
	// USRP (PCM) -> Target format
	sc.toFormatCmd = exec.Command("ffmpeg",
		"-f", "s16le",              // Input: signed 16-bit little-endian
		"-ar", fmt.Sprintf("%d", config.InputRate),  // Input sample rate
		"-ac", fmt.Sprintf("%d", config.Channels),   // Input channels
		"-i", "pipe:0",             // Read from stdin
		"-f", config.OutputFormat,  // Output format
		"-ar", fmt.Sprintf("%d", config.OutputRate), // Output sample rate
		"-ac", fmt.Sprintf("%d", config.Channels),   // Output channels
	)
	
	// Add codec-specific options
	if config.OutputFormat == "opus" || config.OutputFormat == "ogg" {
		sc.toFormatCmd.Args = append(sc.toFormatCmd.Args, 
			"-c:a", "libopus",
			"-b:a", fmt.Sprintf("%dk", config.BitRate),
			"-frame_duration", "20", // 20ms frames to match USRP
		)
	}
	
	sc.toFormatCmd.Args = append(sc.toFormatCmd.Args, "pipe:1") // Write to stdout
	
	// Target format -> USRP (PCM)
	sc.fromFormatCmd = exec.Command("ffmpeg",
		"-f", config.InputFormat,   // Input format
		"-i", "pipe:0",            // Read from stdin
		"-f", "s16le",             // Output: signed 16-bit little-endian
		"-ar", "8000",             // USRP sample rate
		"-ac", "1",                // USRP mono
		"pipe:1",                  // Write to stdout
	)
	
	// Set up pipes
	var err error
	if sc.toFormatIn, err = sc.toFormatCmd.StdinPipe(); err != nil {
		return err
	}
	if sc.toFormatOut, err = sc.toFormatCmd.StdoutPipe(); err != nil {
		return err
	}
	if sc.fromFormatIn, err = sc.fromFormatCmd.StdinPipe(); err != nil {
		return err
	}
	if sc.fromFormatOut, err = sc.fromFormatCmd.StdoutPipe(); err != nil {
		return err
	}
	
	// Start processes
	if err := sc.toFormatCmd.Start(); err != nil {
		return fmt.Errorf("failed to start to-format FFmpeg: %w", err)
	}
	if err := sc.fromFormatCmd.Start(); err != nil {
		return fmt.Errorf("failed to start from-format FFmpeg: %w", err)
	}
	
	return nil
}

// USRPToFormat converts USRP voice message to target format
func (sc *StreamingConverter) USRPToFormat(voiceMsg *usrp.VoiceMessage) ([]byte, error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	if sc.closed {
		return nil, fmt.Errorf("converter is closed")
	}
	
	// Convert int16 samples to bytes (little-endian)
	pcmBytes := make([]byte, len(voiceMsg.AudioData)*2)
	for i, sample := range voiceMsg.AudioData {
		binary.LittleEndian.PutUint16(pcmBytes[i*2:], uint16(sample))
	}
	
	// Send PCM data to FFmpeg
	if _, err := sc.toFormatIn.Write(pcmBytes); err != nil {
		return nil, fmt.Errorf("failed to write PCM data: %w", err)
	}
	
	// Read converted data (non-blocking with timeout)
	result := make([]byte, 4096) // Buffer for compressed data
	n, err := sc.readWithTimeout(sc.toFormatOut, result, 100*time.Millisecond)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read converted data: %w", err)
	}
	
	return result[:n], nil
}

// FormatToUSRP converts target format data to USRP voice messages
func (sc *StreamingConverter) FormatToUSRP(data []byte) ([]*usrp.VoiceMessage, error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	if sc.closed {
		return nil, fmt.Errorf("converter is closed")
	}
	
	// Send compressed data to FFmpeg
	if _, err := sc.fromFormatIn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write format data: %w", err)
	}
	
	// Read PCM data
	pcmBuffer := make([]byte, 8192) // Buffer for PCM output
	n, err := sc.readWithTimeout(sc.fromFormatOut, pcmBuffer, 100*time.Millisecond)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read PCM data: %w", err)
	}
	
	// Convert bytes to int16 samples
	samples := make([]int16, n/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcmBuffer[i*2:]))
	}
	
	// Add to buffer
	sc.pcmBuffer = append(sc.pcmBuffer, samples...)
	
	// Create USRP voice messages (160 samples each)
	var messages []*usrp.VoiceMessage
	seq := uint32(time.Now().Unix()) // Simple sequence numbering
	
	for len(sc.pcmBuffer) >= usrp.VoiceFrameSize {
		msg := &usrp.VoiceMessage{
			Header: usrp.NewHeader(usrp.USRP_TYPE_VOICE, seq),
		}
		
		// Copy 160 samples to message
		copy(msg.AudioData[:], sc.pcmBuffer[:usrp.VoiceFrameSize])
		
		// Remove consumed samples from buffer
		sc.pcmBuffer = sc.pcmBuffer[usrp.VoiceFrameSize:]
		
		messages = append(messages, msg)
		seq++
	}
	
	return messages, nil
}

// readWithTimeout reads from a reader with a timeout
func (sc *StreamingConverter) readWithTimeout(reader io.Reader, buf []byte, timeout time.Duration) (int, error) {
	type result struct {
		n   int
		err error
	}
	
	ch := make(chan result, 1)
	go func() {
		n, err := reader.Read(buf)
		ch <- result{n, err}
	}()
	
	select {
	case res := <-ch:
		return res.n, res.err
	case <-time.After(timeout):
		return 0, fmt.Errorf("read timeout after %v", timeout)
	}
}

// Close stops FFmpeg processes and cleans up resources
func (sc *StreamingConverter) Close() error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	if sc.closed {
		return nil
	}
	sc.closed = true
	
	// Close pipes
	if sc.toFormatIn != nil {
		sc.toFormatIn.Close()
	}
	if sc.toFormatOut != nil {
		sc.toFormatOut.Close()
	}
	if sc.fromFormatIn != nil {
		sc.fromFormatIn.Close()
	}
	if sc.fromFormatOut != nil {
		sc.fromFormatOut.Close()
	}
	
	// Stop processes
	if sc.toFormatCmd != nil {
		sc.toFormatCmd.Process.Kill()
		sc.toFormatCmd.Wait()
	}
	if sc.fromFormatCmd != nil {
		sc.fromFormatCmd.Process.Kill()
		sc.fromFormatCmd.Wait()
	}
	
	return nil
}

// AudioBridge provides high-level audio bridging between USRP and other formats
type AudioBridge struct {
	converter Converter
	
	// Channels for streaming data
	USRPToChan   chan []byte           // Converted audio data out
	ChanToUSRP   chan []*usrp.VoiceMessage // USRP messages out
	
	// Input channels  
	USRPIn       chan *usrp.VoiceMessage   // USRP messages in
	FormatIn     chan []byte               // Format data in
	
	stopChan     chan bool
	running      bool
	mutex        sync.Mutex
}

// NewAudioBridge creates a new audio bridge
func NewAudioBridge(converter Converter) *AudioBridge {
	return &AudioBridge{
		converter:    converter,
		USRPToChan:   make(chan []byte, 100),
		ChanToUSRP:   make(chan []*usrp.VoiceMessage, 100),
		USRPIn:       make(chan *usrp.VoiceMessage, 100),
		FormatIn:     make(chan []byte, 100),
		stopChan:     make(chan bool, 1),
	}
}

// Start begins the audio bridging process
func (ab *AudioBridge) Start() error {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	
	if ab.running {
		return fmt.Errorf("bridge already running")
	}
	ab.running = true
	
	// Start conversion goroutines
	go ab.usrpToFormatWorker()
	go ab.formatToUSRPWorker()
	
	return nil
}

// Stop stops the audio bridging
func (ab *AudioBridge) Stop() error {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	
	if !ab.running {
		return nil
	}
	ab.running = false
	
	ab.stopChan <- true
	ab.stopChan <- true // For both workers
	
	return ab.converter.Close()
}

// usrpToFormatWorker converts incoming USRP messages to target format
func (ab *AudioBridge) usrpToFormatWorker() {
	for {
		select {
		case <-ab.stopChan:
			return
		case voiceMsg := <-ab.USRPIn:
			if data, err := ab.converter.USRPToFormat(voiceMsg); err == nil {
				select {
				case ab.USRPToChan <- data:
				default:
					// Drop data if channel full
				}
			}
		}
	}
}

// formatToUSRPWorker converts incoming format data to USRP messages
func (ab *AudioBridge) formatToUSRPWorker() {
	for {
		select {
		case <-ab.stopChan:
			return
		case data := <-ab.FormatIn:
			if messages, err := ab.converter.FormatToUSRP(data); err == nil {
				select {
				case ab.ChanToUSRP <- messages:
				default:
					// Drop data if channel full
				}
			}
		}
	}
}