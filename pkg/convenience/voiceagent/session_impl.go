package voiceagent

import (
	"context"
	"strings"
	"sync"
	"time"
)

// voiceSession implements the Session interface.
type voiceSession struct {
	id           string
	agent        *convenienceVoiceAgent
	audioCh      chan []byte
	responsesCh  chan []byte
	transcript   string
	active       bool
	startTime    time.Time
	onTranscript TranscriptCallback
	onResponse   ResponseCallback
	onError      ErrorCallback

	mu         sync.RWMutex
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup // Wait group for goroutines
}

// ID returns the unique session identifier.
func (s *voiceSession) ID() string {
	return s.id
}

// Start begins the session and starts processing audio.
func (s *voiceSession) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return nil // Already started
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.active = true
	s.startTime = time.Now()
	s.mu.Unlock()

	// Start the audio processing goroutine
	s.wg.Add(1)
	go s.processAudioLoop(ctx)

	return nil
}

// Stop ends the session and releases resources.
func (s *voiceSession) Stop() error {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return nil // Already stopped
	}

	s.active = false
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	// Close the audio input channel to signal goroutines to exit
	close(s.audioCh)
	s.mu.Unlock()

	// Wait for goroutines to finish before closing response channel
	s.wg.Wait()

	// Now safe to close response channel
	close(s.responsesCh)

	// Record session stop with duration
	duration := time.Since(s.startTime)
	s.agent.metrics.RecordSessionStop(context.Background(), s.id, duration)

	return nil
}

// SendAudio sends audio data to the session for processing.
func (s *voiceSession) SendAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	audioCh := s.audioCh
	s.mu.RUnlock()

	if !active {
		return ErrSessionStopped
	}

	select {
	case audioCh <- audio:
		s.agent.metrics.RecordAudioChunk(ctx)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReceiveAudio returns a channel for receiving synthesized audio responses.
func (s *voiceSession) ReceiveAudio() <-chan []byte {
	return s.responsesCh
}

// GetTranscript returns the accumulated transcript of the session.
func (s *voiceSession) GetTranscript() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transcript
}

// IsActive returns whether the session is currently active.
func (s *voiceSession) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// processAudioLoop processes incoming audio in a loop.
func (s *voiceSession) processAudioLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case audio, ok := <-s.audioCh:
			if !ok {
				return
			}
			s.handleAudio(ctx, audio)
		}
	}
}

// handleAudio processes a single audio chunk.
func (s *voiceSession) handleAudio(ctx context.Context, audio []byte) {
	// Check if context is canceled early
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Check VAD if available
	if s.agent.vad != nil {
		hasVoice, err := s.agent.vad.Process(ctx, audio)
		if err != nil {
			if s.onError != nil {
				s.onError(err)
			}
			return
		}
		if !hasVoice {
			// No voice activity, skip processing
			return
		}
	}

	// Transcribe audio
	startSTT := time.Now()
	text, err := s.agent.stt.Transcribe(ctx, audio)
	sttLatency := time.Since(startSTT)
	if err != nil {
		s.agent.metrics.RecordTranscription(ctx, sttLatency, false)
		if s.onError != nil {
			s.onError(err)
		}
		return
	}
	s.agent.metrics.RecordTranscription(ctx, sttLatency, true)

	// Update transcript
	s.mu.Lock()
	if s.transcript != "" {
		s.transcript += " "
	}
	s.transcript += text
	s.mu.Unlock()

	if s.onTranscript != nil {
		s.onTranscript(text, true)
	}

	// Check if context is canceled before LLM
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Process with LLM if available
	var response string
	if s.agent.llm != nil {
		response, err = s.agent.processWithLLM(ctx, text)
		if err != nil {
			if s.onError != nil {
				s.onError(err)
			}
			return
		}
	} else {
		// No LLM, just echo back the text
		response = text
	}

	if s.onResponse != nil {
		s.onResponse(response)
	}

	// Check if context is canceled before TTS
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Synthesize response to audio
	startTTS := time.Now()
	audioResponse, err := s.agent.tts.GenerateSpeech(ctx, response)
	ttsLatency := time.Since(startTTS)
	if err != nil {
		s.agent.metrics.RecordSynthesis(ctx, ttsLatency, false)
		if s.onError != nil {
			s.onError(err)
		}
		return
	}
	s.agent.metrics.RecordSynthesis(ctx, ttsLatency, true)

	// Send audio response - check active state safely
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()

	if active {
		select {
		case s.responsesCh <- audioResponse:
		case <-ctx.Done():
		default:
			// Channel full, drop the response
		}
	}
}

// appendTranscript appends text to the session transcript.
func (s *voiceSession) appendTranscript(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	if s.transcript != "" {
		s.transcript += " "
	}
	s.transcript += text
}
